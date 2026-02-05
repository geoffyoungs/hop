# Commands Reference

## Synopsis

```
hop [alias] [flags]
hop [command]
```

## Commands

| Command | Description |
|---------|-------------|
| `hop <alias>` | Connect to the specified host |
| `hop <source>:<alias>` | Connect to host from a specific source |
| `hop completion <shell>` | Generate shell completion scripts |
| `hop help` | Help about any command |

## Global Flags

| Flag | Description |
|------|-------------|
| `--config <path>` | Use a specific config file (disables multi-source) |
| `--local` | Use local config file (`./hosts.ini`) |
| `--user` | Use user config file (`~/.config/hop/hosts.ini`) |
| `--sources <list>` | Load from specific sources (comma-separated) |
| `--all-sources` | Load from all enabled sources (default) |
| `-h, --help` | Help for hop |
| `-v, --version` | Version information |

## Host Sources

Hop loads hosts from multiple sources by default:

| Source | Description | Read | Write |
|--------|-------------|------|-------|
| `ini` | INI config files (`hosts.ini`) | Yes | Yes |
| `ansible` | Ansible inventory (`inventory.yml`) | Yes | No |
| `sshconfig` | SSH config (`~/.ssh/config`) | Yes | No |
| `vagrant` | Vagrant VMs | Yes | No |

Configure sources in `~/.config/hop/settings.toml`:

```toml
[sources]
ini = true
ansible = true
sshconfig = false  # opt-in
vagrant = true
priority = ["ini", "ansible", "vagrant", "sshconfig"]

[collisions]
strategy = "first"  # "first", "qualify", or "error"
```

## Operation Flags

### `--check`

Check if required backend tools are installed.

```bash
hop --check
```

Output:
```
Checking installed backends...

  docker: installed (version 24.0.6)
  k8s: installed (version 1.28.0)
  ssh: installed (version 9.0p1)
```

### `--list`

List all configured hosts.

```bash
hop --list
```

Output (shows source tags when hosts come from multiple sources):
```
  mycontainer (docker) [ini]
  mypod (k8s) [ini]
  production (ssh) [ini]
  webserver (ssh) [ansible]
```

List from specific sources:
```bash
hop --list --sources ini,ansible
```

### `--add`

Add a new host to the configuration.

```bash
hop --add <name> <key=value> [key=value ...]
```

Examples:
```bash
# Add SSH host
hop --add webserver host=192.168.1.100 user=admin

# Add Docker container
hop --add redis type=docker container=redis_cache

# Add Kubernetes pod
hop --add api type=k8s namespace=production pod=api-server-12345

# Add to local config
hop --local --add devserver host=localhost port=2222
```

### `--copy`

Copy files to or from a host.

```bash
# Upload (local to remote)
hop --copy <local-file> <alias>:<remote-path>

# Download (remote to local)
hop --copy <alias>:<remote-path> <local-file>
```

Examples:
```bash
# Upload file
hop --copy config.json production:/app/config.json

# Download file
hop --copy production:/var/log/app.log ./app.log

# Upload to Docker container
hop --copy script.sh mycontainer:/scripts/

# Download from Kubernetes pod
hop --copy mypod:/app/data.json ./data.json
```

### `--forward`

Forward a local port to a remote port.

```bash
hop --forward "<alias> <local-port>:<remote-port>"
```

Examples:
```bash
# Forward local 8080 to remote 80
hop --forward "production 8080:80"

# Forward local 5432 to remote PostgreSQL
hop --forward "database 5432:5432"

# Forward to Kubernetes pod
hop --forward "mypod 3000:3000"
```

Note: Port forwarding is not supported for Docker containers.

## Shell Completion

Generate completion scripts for your shell:

### Bash

```bash
# Load for current session
source <(hop completion bash)

# Install permanently (Linux)
hop completion bash > /etc/bash_completion.d/hop

# Install permanently (macOS with Homebrew)
hop completion bash > $(brew --prefix)/etc/bash_completion.d/hop
```

### Zsh

```bash
# Load for current session
source <(hop completion zsh)

# Install permanently
hop completion zsh > "${fpath[1]}/_hop"
```

### Fish

```bash
# Load for current session
hop completion fish | source

# Install permanently
hop completion fish > ~/.config/fish/completions/hop.fish
```

### PowerShell

```powershell
hop completion powershell | Out-String | Invoke-Expression
```

## Examples

### Daily Workflow

```bash
# List available hosts
hop --list

# Connect to production server
hop production

# Check logs and disconnect
# (inside remote shell)
tail -f /var/log/app.log
exit

# Copy logs locally
hop --copy production:/var/log/app.log ./logs/

# Forward remote service for local debugging
hop --forward "staging 8080:80"
```

### Multi-Environment Setup

```bash
# Add hosts for different environments
hop --add prod-api host=prod.example.com user=deploy
hop --add staging-api host=staging.example.com user=deploy
hop --add dev-api host=localhost port=2222 user=dev

# Quick access
hop prod-api
hop staging-api
hop dev-api
```

### Project-Local Configuration

```bash
# Create local config for a project
hop --local --add db type=docker container=project_postgres
hop --local --add api type=docker container=project_api

# Use local hosts
hop --local --list
hop --local db
```
