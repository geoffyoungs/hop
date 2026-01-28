package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/geoff/hop/internal/backend"
)

// parseRsyncArgs extracts rsync options and positional args from the args slice.
// Returns parsed options, source, destination, and any error.
func parseRsyncArgs(args []string) (*backend.RsyncOptions, string, string, error) {
	opts := &backend.RsyncOptions{}
	var positional []string

	for i := 0; i < len(args); i++ {
		arg := args[i]

		if arg == "--" {
			// Everything after -- is positional
			positional = append(positional, args[i+1:]...)
			break
		}

		if strings.HasPrefix(arg, "--exclude=") {
			opts.Exclude = append(opts.Exclude, strings.TrimPrefix(arg, "--exclude="))
			continue
		}

		if arg == "--exclude" {
			if i+1 >= len(args) {
				return nil, "", "", fmt.Errorf("--exclude requires a pattern argument")
			}
			i++
			opts.Exclude = append(opts.Exclude, args[i])
			continue
		}

		if arg == "--archive" {
			opts.Archive = true
			continue
		}
		if arg == "--verbose" {
			opts.Verbose = true
			continue
		}
		if arg == "--compress" {
			opts.Compress = true
			continue
		}
		if arg == "--recursive" {
			opts.Recursive = true
			continue
		}
		if arg == "--dry-run" {
			opts.DryRun = true
			continue
		}
		if arg == "--delete" {
			opts.Delete = true
			continue
		}

		// Handle short flags (may be combined, e.g. -avz)
		if strings.HasPrefix(arg, "-") && !strings.HasPrefix(arg, "--") {
			flags := arg[1:]
			recognized := true
			for _, ch := range flags {
				switch ch {
				case 'a':
					opts.Archive = true
				case 'v':
					opts.Verbose = true
				case 'z':
					opts.Compress = true
				case 'r':
					opts.Recursive = true
				case 'n':
					opts.DryRun = true
				default:
					recognized = false
				}
			}
			if !recognized {
				// Entire flag group goes to Extra (passthrough for SSH)
				opts.Extra = append(opts.Extra, arg)
			}
			continue
		}

		// Non-flag argument
		positional = append(positional, arg)
	}

	if len(positional) != 2 {
		return nil, "", "", fmt.Errorf("rsync requires exactly 2 positional arguments (source and dest), got %d", len(positional))
	}

	return opts, positional[0], positional[1], nil
}

func runRsync(args []string) error {
	opts, src, dst, err := parseRsyncArgs(args)
	if err != nil {
		return err
	}

	// Honor hop's --dry-run flag in addition to rsync's -n/--dry-run
	if dryRunFlag {
		opts.DryRun = true
	}

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

	return backend.Rsync(context.Background(), b, host, localPath, remotePath, direction, opts)
}
