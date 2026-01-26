// Package cli provides the hop CLI commands.
package cli

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/geoff/hop/internal/backend"
	"github.com/geoff/hop/internal/config"
	"github.com/spf13/cobra"
)

var (
	// Version information, set by ldflags
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

var (
	checkFlag   bool
	listFlag    bool
	copyFlag    bool
	forwardFlag string
	configPath  string
	localFlag   bool
	userFlag    bool
	addFlag     bool
)

// RootCmd is the base command for hop
var RootCmd = &cobra.Command{
	Use:   "hop [alias]",
	Short: "Connect to named hosts via SSH, Docker, or Kubernetes",
	Long: `Hop connects to named hosts defined in an INI configuration file.

Backends:
  ssh     Connect to remote servers via SSH (default)
  docker  Shell into running Docker containers
  k8s     Exec into Kubernetes pods

Examples:
  hop production                              # Connect to host
  hop --list                                  # List all hosts
  hop --check                                 # Check backend availability
  hop --copy file.txt server:/path            # Upload file
  hop --copy server:/path file.txt            # Download file
  hop --forward "server 8080:80"              # Port forward
  hop --add myhost host=10.0.0.1 user=admin   # Add new host
  hop --local --list                          # List hosts from ./hosts.ini

Configuration:
  Default: ~/.config/hop/hosts.ini
  Local:   ./hosts.ini (with --local flag)
  Custom:  Use --config to specify a path`,
	Args:              cobra.ArbitraryArgs,
	RunE:              runRoot,
	ValidArgsFunction: completeHostNames,
}

func init() {
	RootCmd.Flags().BoolVar(&checkFlag, "check", false, "Check if required tools are installed")
	RootCmd.Flags().BoolVar(&listFlag, "list", false, "List configured hosts")
	RootCmd.Flags().BoolVar(&copyFlag, "copy", false, "Copy files to/from host (usage: --copy file alias:path)")
	RootCmd.Flags().StringVar(&forwardFlag, "forward", "", "Forward ports (usage: --forward \"alias local:remote\")")
	RootCmd.Flags().StringVar(&configPath, "config", "", "Path to config file")
	RootCmd.Flags().BoolVar(&localFlag, "local", false, "Use local config file (./hosts.ini)")
	RootCmd.Flags().BoolVar(&userFlag, "user", false, "Use user config file (~/.config/hop/hosts.ini)")
	RootCmd.Flags().BoolVar(&addFlag, "add", false, "Add a new host (usage: --add name key=value ...)")

	// Mark mutually exclusive flags
	RootCmd.MarkFlagsMutuallyExclusive("local", "user", "config")

	RootCmd.AddCommand(completionCmd)
}

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion scripts",
	Long: `Generate shell completion scripts for hop.

To load completions:

Bash:
  $ source <(hop completion bash)
  # To load completions for each session, execute once:
  # Linux:
  $ hop completion bash > /etc/bash_completion.d/hop
  # macOS:
  $ hop completion bash > $(brew --prefix)/etc/bash_completion.d/hop

Zsh:
  $ source <(hop completion zsh)
  # To load completions for each session, execute once:
  $ hop completion zsh > "${fpath[1]}/_hop"

Fish:
  $ hop completion fish | source
  # To load completions for each session, execute once:
  $ hop completion fish > ~/.config/fish/completions/hop.fish

PowerShell:
  PS> hop completion powershell | Out-String | Invoke-Expression
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return RootCmd.GenBashCompletion(os.Stdout)
		case "zsh":
			return RootCmd.GenZshCompletion(os.Stdout)
		case "fish":
			return RootCmd.GenFishCompletion(os.Stdout, true)
		case "powershell":
			return RootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
		default:
			return fmt.Errorf("unknown shell: %s", args[0])
		}
	},
}

// SetVersion sets version information for the command
func SetVersion(version, commit, date string) {
	Version = version
	Commit = commit
	Date = date
	RootCmd.Version = fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date)
}

func runRoot(cmd *cobra.Command, args []string) error {
	if checkFlag {
		return runCheck()
	}

	if addFlag {
		return runAdd(args)
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	if listFlag {
		return runList(cfg)
	}

	if len(args) == 0 {
		return cmd.Help()
	}

	if copyFlag {
		return runCopy(cfg, args)
	}

	if forwardFlag != "" {
		return runForward(cfg, forwardFlag)
	}

	return runConnect(cfg, args[0])
}

func getPathMode() config.PathMode {
	if configPath != "" {
		return config.ModeExplicit
	}
	if localFlag {
		return config.ModeLocal
	}
	if userFlag {
		return config.ModeUser
	}
	return config.ModeDefault
}

func getConfigPath() string {
	return config.ResolvePath(getPathMode(), configPath)
}

func loadConfig() (*config.Config, error) {
	path := getConfigPath()
	return config.LoadFromPath(path)
}

func runAdd(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: hop --add <name> <key=value> [key=value ...]")
	}

	name := args[0]
	props, err := config.ParseKeyValuePairs(args[1:])
	if err != nil {
		return fmt.Errorf("invalid property: %w", err)
	}

	path := getConfigPath()

	if err := config.AddHost(path, name, props); err != nil {
		return err
	}

	fmt.Printf("Added host %q to %s\n", name, path)
	return nil
}

func runCheck() error {
	fmt.Println("Checking installed backends...")
	fmt.Println()

	backends := backend.All()
	names := make([]string, 0, len(backends))
	for name := range backends {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		b := backends[name]
		result, err := b.Check()
		if err != nil {
			fmt.Printf("  %s: error checking (%v)\n", name, err)
			continue
		}

		if result.Available {
			fmt.Printf("  %s: installed (version %s)\n", name, result.Version)
			for _, warn := range result.Warnings {
				fmt.Printf("    warning: %s\n", warn)
			}
		} else {
			fmt.Printf("  %s: not installed\n", name)
			for _, missing := range result.Missing {
				fmt.Printf("    missing: %s\n", missing)
			}
		}
	}

	return nil
}

func runList(cfg *config.Config) error {
	names := cfg.Names()
	sort.Strings(names)

	for _, name := range names {
		host, _ := cfg.Get(name)
		hostType := host.Type
		if hostType == "" {
			hostType = "ssh"
		}
		fmt.Printf("  %s (%s)\n", name, hostType)
	}

	return nil
}

func runConnect(cfg *config.Config, alias string) error {
	hostCfg, ok := cfg.Get(alias)
	if !ok {
		return fmt.Errorf("host %q not found in configuration", alias)
	}

	host := hostCfg.ToHost()

	b, err := backend.Get(host.Type)
	if err != nil {
		return fmt.Errorf("unknown backend type %q", host.Type)
	}

	if err := b.Validate(host); err != nil {
		return fmt.Errorf("invalid configuration for %q: %v", alias, err)
	}

	return b.Connect(context.Background(), host)
}

func runCopy(cfg *config.Config, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("copy requires source and destination arguments")
	}

	src, dst := args[0], args[1]

	var alias, localPath, remotePath, direction string

	if strings.Contains(src, ":") {
		parts := strings.SplitN(src, ":", 2)
		alias = parts[0]
		remotePath = parts[1]
		localPath = dst
		direction = "from"
	} else if strings.Contains(dst, ":") {
		parts := strings.SplitN(dst, ":", 2)
		alias = parts[0]
		remotePath = parts[1]
		localPath = src
		direction = "to"
	} else {
		return fmt.Errorf("one argument must be in alias:path format")
	}

	hostCfg, ok := cfg.Get(alias)
	if !ok {
		return fmt.Errorf("host %q not found in configuration", alias)
	}

	host := hostCfg.ToHost()

	b, err := backend.Get(host.Type)
	if err != nil {
		return fmt.Errorf("unknown backend type %q", host.Type)
	}

	if err := b.Validate(host); err != nil {
		return fmt.Errorf("invalid configuration for %q: %v", alias, err)
	}

	return b.Copy(context.Background(), host, localPath, remotePath, direction)
}

func runForward(cfg *config.Config, forward string) error {
	parts := strings.Fields(forward)
	if len(parts) != 2 {
		return fmt.Errorf("forward requires alias and local:remote format")
	}

	alias := parts[0]
	portParts := strings.Split(parts[1], ":")
	if len(portParts) != 2 {
		return fmt.Errorf("port format must be local:remote")
	}

	localPort, err := strconv.Atoi(portParts[0])
	if err != nil {
		return fmt.Errorf("invalid local port: %v", err)
	}

	remotePort, err := strconv.Atoi(portParts[1])
	if err != nil {
		return fmt.Errorf("invalid remote port: %v", err)
	}

	hostCfg, ok := cfg.Get(alias)
	if !ok {
		return fmt.Errorf("host %q not found in configuration", alias)
	}

	host := hostCfg.ToHost()

	b, err := backend.Get(host.Type)
	if err != nil {
		return fmt.Errorf("unknown backend type %q", host.Type)
	}

	if err := b.Validate(host); err != nil {
		return fmt.Errorf("invalid configuration for %q: %v", alias, err)
	}

	fmt.Printf("Forwarding localhost:%d -> %s:%d\n", localPort, alias, remotePort)
	fmt.Println("Press Ctrl+C to stop")

	return b.ForwardPort(context.Background(), host, localPort, remotePort)
}

func completeHostNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	cfg, err := loadConfig()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	var names []string
	for _, name := range cfg.Names() {
		if strings.HasPrefix(name, toComplete) {
			names = append(names, name)
		}
	}

	return names, cobra.ShellCompDirectiveNoFileComp
}
