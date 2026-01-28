// Package cli provides the hop CLI commands.
package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
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
	checkFlag    bool
	listFlag     bool
	copyFlag     bool
	forwardFlag  string
	configPath   string
	localFlag    bool
	userFlag     bool
	addFlag      bool
	terminalFlag bool
	removeFlag     string
	showFlag       string
	editFlag       bool
	dryRunFlag     bool
	executeFlag    string
	installKeyFlag bool
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
  hop                                         # Connect to default/only host
  hop .                                       # Connect to default in project config
  hop ~                                       # Connect to default in user config
  hop production                              # Connect to named host
  hop .production                             # Use project config only
  hop ~production                             # Use user config only
  hop --list                                  # List all hosts
  hop --check                                 # Check backend availability
  hop --copy file.txt server:/path            # Upload file
  hop --copy server:/path file.txt            # Download file
  hop --forward "server 8080:80"              # Port forward
  hop --add myhost host=10.0.0.1 user=admin   # Add new host
  hop --remove myhost                         # Remove a host
  hop --show myhost                           # Show host configuration
  hop --edit                                  # Open config in $EDITOR
  hop --dry-run myhost                        # Show command without executing
  hop myhost -e "whoami"                      # Execute command on remote host
  hop --local --list                          # List hosts from ./hosts.ini
  hop --terminal production                   # Sync terminfo then connect
  hop --install-key server                    # Install SSH public key on host
  hop --install-key server ~/.ssh/id_custom.pub  # Install a specific key

Configuration:
  Search order: ./hosts.ini -> parent dirs (up to .git) -> ~/.config/hop/hosts.ini

  Prefix syntax:
    hop production      Search project config first, then user config
    hop .production     Use project config only (walk up to .git)
    hop ~production     Use user config only (~/.config/hop/hosts.ini)

  Default host:
    If only one host is configured, it becomes the default.
    Or mark a host as default with: default = yes

  Flags:
    --local             Use ./hosts.ini only (no directory walking)
    --user              Use ~/.config/hop/hosts.ini only
    --config PATH       Use explicit config file path`,
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
	RootCmd.Flags().BoolVar(&terminalFlag, "terminal", false, "Sync local terminfo to remote host before connecting")
	RootCmd.Flags().StringVar(&removeFlag, "remove", "", "Remove a host from config")
	RootCmd.Flags().StringVar(&showFlag, "show", "", "Show configuration for a host")
	RootCmd.Flags().BoolVar(&editFlag, "edit", false, "Open config file in editor")
	RootCmd.Flags().BoolVar(&dryRunFlag, "dry-run", false, "Show the command that would be executed")
	RootCmd.Flags().StringVarP(&executeFlag, "execute", "e", "", "Execute a command on the remote host")
	RootCmd.Flags().BoolVar(&installKeyFlag, "install-key", false, "Install SSH public key on remote host")

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

	if removeFlag != "" {
		return runRemove(removeFlag)
	}

	if showFlag != "" {
		return runShow(showFlag)
	}

	if editFlag {
		return runEdit()
	}

	if installKeyFlag {
		return runInstallKey(args)
	}

	if listFlag {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		return runList(cfg)
	}

	if len(args) == 0 {
		// Try to find default target
		cfg, err := loadConfig()
		if err != nil {
			return cmd.Help()
		}
		host, ok := cfg.DefaultHost()
		if !ok {
			return cmd.Help()
		}
		return runConnectWithHost(host)
	}

	if copyFlag {
		return runCopy(args)
	}

	if forwardFlag != "" {
		return runForward(forwardFlag)
	}

	return runConnect(args[0])
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

// parseAliasPrefix extracts a prefix from the alias and returns the clean alias and mode
// ".alias" -> alias, ModeProject (project config only)
// "~alias" -> alias, ModeUser (user config only)
// "alias"  -> alias, ModeDefault (search both)
func parseAliasPrefix(alias string) (string, config.PathMode) {
	if strings.HasPrefix(alias, ".") {
		return alias[1:], config.ModeProject
	}
	if strings.HasPrefix(alias, "~") {
		return alias[1:], config.ModeUser
	}
	return alias, config.ModeDefault
}

func loadConfig() (*config.Config, error) {
	path := getConfigPath()
	return config.LoadFromPath(path)
}

// loadConfigForAlias loads config based on alias prefix, returning the config and cleaned alias
func loadConfigForAlias(alias string) (*config.Config, string, error) {
	cleanAlias, mode := parseAliasPrefix(alias)

	// If flags are set, they override prefix
	if configPath != "" || localFlag || userFlag {
		cfg, err := loadConfig()
		return cfg, cleanAlias, err
	}

	path := config.ResolvePath(mode, "")

	// Handle errors for prefix modes that require specific configs
	if mode == config.ModeProject && path == "" {
		return nil, cleanAlias, fmt.Errorf("no project config found (walked up to .git or filesystem root)")
	}
	if mode == config.ModeUser {
		if !config.ConfigExists(path) {
			return nil, cleanAlias, fmt.Errorf("user config not found at %s", config.UserConfigPath())
		}
	}

	cfg, err := config.LoadFromPath(path)
	return cfg, cleanAlias, err
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

func runRemove(alias string) error {
	cfg, cleanAlias, err := loadConfigForAlias(alias)
	if err != nil {
		return err
	}

	// Verify host exists
	_, err = cfg.GetByPrefix(cleanAlias)
	if err != nil {
		return err
	}

	path := getConfigPath()
	if err := config.RemoveHost(path, cleanAlias); err != nil {
		return err
	}

	fmt.Printf("Removed host %q from %s\n", cleanAlias, path)
	return nil
}

func runShow(alias string) error {
	cfg, cleanAlias, err := loadConfigForAlias(alias)
	if err != nil {
		return err
	}

	hostCfg, err := cfg.GetByPrefix(cleanAlias)
	if err != nil {
		return err
	}

	props := hostCfg.ToMap()
	fmt.Print(config.FormatHostEntry(hostCfg.Name, props))
	return nil
}

func runEdit() error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	path := getConfigPath()
	cmd := exec.Command(editor, path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func runInstallKey(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: hop --install-key <alias> [public-key-path]")
	}

	alias := args[0]
	cfg, cleanAlias, err := loadConfigForAlias(alias)
	if err != nil {
		return err
	}

	hostCfg, err := cfg.GetByPrefix(cleanAlias)
	if err != nil {
		return err
	}

	host := hostCfg.ToHost()

	b, err := backend.Get(host.Type)
	if err != nil {
		return fmt.Errorf("unknown backend type %q", host.Type)
	}

	if err := b.Validate(host); err != nil {
		return fmt.Errorf("invalid configuration for %q: %v", hostCfg.Name, err)
	}

	var pubKeyPath string
	if len(args) >= 2 {
		pubKeyPath = args[1]
	} else {
		pubKeyPath, err = backend.DiscoverPublicKey(host)
		if err != nil {
			return err
		}
	}

	fmt.Printf("Installing %s on %s...\n", pubKeyPath, hostCfg.Name)
	ctx := context.Background()
	return backend.InstallKey(ctx, b, host, pubKeyPath)
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

func runConnect(alias string) error {
	cfg, cleanAlias, err := loadConfigForAlias(alias)
	if err != nil {
		return err
	}

	// If no alias specified, find default
	if cleanAlias == "" {
		host, ok := cfg.DefaultHost()
		if !ok {
			names := cfg.Names()
			if len(names) == 0 {
				return fmt.Errorf("no hosts configured")
			}
			sort.Strings(names)
			return fmt.Errorf("multiple hosts configured, please specify one: %s", strings.Join(names, ", "))
		}
		return runConnectWithHost(host)
	}

	hostCfg, err := cfg.GetByPrefix(cleanAlias)
	if err != nil {
		return err
	}

	return runConnectWithHost(hostCfg)
}

func runConnectWithHost(hostCfg *config.HostConfig) error {
	host := hostCfg.ToHost()

	b, err := backend.Get(host.Type)
	if err != nil {
		return fmt.Errorf("unknown backend type %q", host.Type)
	}

	if err := b.Validate(host); err != nil {
		return fmt.Errorf("invalid configuration for %q: %v", hostCfg.Name, err)
	}

	ctx := context.Background()

	// Handle --dry-run
	if dryRunFlag {
		cmd, args, err := b.BuildConnectCommand(ctx, host)
		if err != nil {
			return fmt.Errorf("failed to build command: %v", err)
		}
		// Print command in a way that can be copy-pasted
		fmt.Printf("%s %s\n", cmd, strings.Join(args, " "))
		return nil
	}

	// Handle --execute / -e
	if executeFlag != "" {
		return b.Exec(ctx, host, executeFlag)
	}

	if terminalFlag {
		if err := backend.SyncTerminfo(ctx, b, host); err != nil {
			return fmt.Errorf("failed to sync terminfo: %v", err)
		}
	}

	return b.Connect(ctx, host)
}

func runCopy(args []string) error {
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

	cfg, cleanAlias, err := loadConfigForAlias(alias)
	if err != nil {
		return err
	}

	hostCfg, err := cfg.GetByPrefix(cleanAlias)
	if err != nil {
		return err
	}

	host := hostCfg.ToHost()

	b, err := backend.Get(host.Type)
	if err != nil {
		return fmt.Errorf("unknown backend type %q", host.Type)
	}

	if err := b.Validate(host); err != nil {
		return fmt.Errorf("invalid configuration for %q: %v", hostCfg.Name, err)
	}

	return b.Copy(context.Background(), host, localPath, remotePath, direction)
}

func runForward(forward string) error {
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

	cfg, cleanAlias, err := loadConfigForAlias(alias)
	if err != nil {
		return err
	}

	hostCfg, err := cfg.GetByPrefix(cleanAlias)
	if err != nil {
		return err
	}

	host := hostCfg.ToHost()

	b, err := backend.Get(host.Type)
	if err != nil {
		return fmt.Errorf("unknown backend type %q", host.Type)
	}

	if err := b.Validate(host); err != nil {
		return fmt.Errorf("invalid configuration for %q: %v", hostCfg.Name, err)
	}

	fmt.Printf("Forwarding localhost:%d -> %s:%d\n", localPort, hostCfg.Name, remotePort)
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
