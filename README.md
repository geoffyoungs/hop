# hop

[![Go Reference](https://pkg.go.dev/badge/github.com/geoffyoungs/hop.svg)](https://pkg.go.dev/github.com/geoffyoungs/hop)
[![Go Report Card](https://goreportcard.com/badge/github.com/geoffyoungs/hop)](https://goreportcard.com/report/github.com/geoffyoungs/hop)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A CLI tool for connecting to named hosts using an INI configuration file. Supports SSH, Docker, and Kubernetes backends.

## Quick Start

```bash
# Install (macOS/Linux)
brew install geoffyoungs/tap/hop

# Add a host
hop --add production host=192.168.1.100 user=admin

# Connect
hop production
```

## Installation

### Homebrew (macOS/Linux)

```bash
# Add the tap and install
brew install geoffyoungs/tap/hop

# Or in one command
brew install geoffyoungs/tap/hop
```

This works on:
- macOS (Intel and Apple Silicon)
- Linux (x86_64 and ARM64)

### Debian/Ubuntu (apt)

Download the `.deb` package from the [releases page](https://github.com/geoffyoungs/hop/releases) and install:

```bash
# For x86_64/amd64
curl -LO https://github.com/geoffyoungs/hop/releases/download/v0.1.0-beta/hop_0.1.0-beta_linux_amd64.deb
sudo dpkg -i hop_0.1.0-beta_linux_amd64.deb

# For ARM64
curl -LO https://github.com/geoffyoungs/hop/releases/download/v0.1.0-beta/hop_0.1.0-beta_linux_arm64.deb
sudo dpkg -i hop_0.1.0-beta_linux_arm64.deb
```

### Fedora/RHEL/CentOS (rpm)

Download the `.rpm` package from the [releases page](https://github.com/geoffyoungs/hop/releases) and install:

```bash
# For x86_64/amd64
curl -LO https://github.com/geoffyoungs/hop/releases/download/v0.1.0-beta/hop_0.1.0-beta_linux_amd64.rpm
sudo rpm -i hop_0.1.0-beta_linux_amd64.rpm

# For ARM64
curl -LO https://github.com/geoffyoungs/hop/releases/download/v0.1.0-beta/hop_0.1.0-beta_linux_arm64.rpm
sudo rpm -i hop_0.1.0-beta_linux_arm64.rpm
```

### Binary Download (All Platforms)

Download the appropriate archive from the [releases page](https://github.com/geoffyoungs/hop/releases):

| Platform | Architecture | Download |
|----------|--------------|----------|
| macOS | Intel (x86_64) | [hop_0.1.0-beta_darwin_amd64.tar.gz](https://github.com/geoffyoungs/hop/releases/download/v0.1.0-beta/hop_0.1.0-beta_darwin_amd64.tar.gz) |
| macOS | Apple Silicon (ARM64) | [hop_0.1.0-beta_darwin_arm64.tar.gz](https://github.com/geoffyoungs/hop/releases/download/v0.1.0-beta/hop_0.1.0-beta_darwin_arm64.tar.gz) |
| Linux | x86_64 | [hop_0.1.0-beta_linux_amd64.tar.gz](https://github.com/geoffyoungs/hop/releases/download/v0.1.0-beta/hop_0.1.0-beta_linux_amd64.tar.gz) |
| Linux | ARM64 | [hop_0.1.0-beta_linux_arm64.tar.gz](https://github.com/geoffyoungs/hop/releases/download/v0.1.0-beta/hop_0.1.0-beta_linux_arm64.tar.gz) |

Extract and install:

```bash
# Example for macOS ARM64
tar -xzf hop_0.1.0-beta_darwin_arm64.tar.gz
sudo mv hop /usr/local/bin/

# Optional: install man pages
sudo mkdir -p /usr/local/share/man/man1
sudo mv docs/man/*.1 /usr/local/share/man/man1/
```

### Go Install

```bash
go install github.com/geoffyoungs/hop/cmd/hop@latest
```

### Build from Source

```bash
git clone https://github.com/geoffyoungs/hop.git
cd hop
make build
sudo mv build/hop /usr/local/bin/
```

## Configuration

Create a configuration file at `~/.config/hop/hosts.ini`:

```ini
[production]
host = 192.168.1.100
user = admin

[staging]
type = ssh
host = 10.0.0.50
user = deploy
port = 2222
identity = ~/.ssh/staging_key

[mycontainer]
type = docker
container = my_app
shell = /bin/bash

[mypod]
type = k8s
namespace = default
pod = my-pod
container = app
context = my-cluster
```

Or use `--add` to add hosts from the command line:

```bash
hop --add myserver host=10.0.0.1 user=admin
hop --add redis type=docker container=redis_cache
hop --add api type=k8s namespace=prod pod=api-server-12345
```

## Usage

| Command | Description |
|---------|-------------|
| `hop <alias>` | Connect to a host |
| `hop --list` | List configured hosts |
| `hop --check` | Check backend availability |
| `hop --add <name> <key=value>...` | Add a new host |
| `hop --copy <src> <dst>` | Copy files to/from host |
| `hop --forward "<alias> <local>:<remote>"` | Port forwarding |

### Examples

```bash
# Connect to hosts
hop production          # SSH to production server
hop mycontainer         # Shell into Docker container
hop mypod               # Exec into Kubernetes pod

# Copy files
hop --copy localfile.txt production:/remote/path    # Upload
hop --copy production:/remote/file.txt ./local      # Download

# Port forwarding
hop --forward "production 8080:80"    # Forward localhost:8080 to remote:80

# Use local config file (./hosts.ini)
hop --local --list
hop --local --add devserver host=localhost
```

### Shell Completion

```bash
# Bash
eval "$(hop completion bash)"

# Zsh
eval "$(hop completion zsh)"

# Fish
hop completion fish | source
```

## Backend Types

| Type | Required Fields | Optional Fields |
|------|-----------------|-----------------|
| `ssh` (default) | `host` | `user`, `port`, `identity` |
| `docker` | `container` | `shell` |
| `k8s` | `pod` OR `selector` OR `pod_grep` | `namespace`, `container`, `context`, `shell` |

### Kubernetes Dynamic Pod Discovery

For Kubernetes, you can specify pods in three ways:

1. **Exact pod name** - use `pod` for a specific pod
2. **Label selector** - use `selector` to find pods by labels (recommended)
3. **Name pattern** - use `pod_grep` to find pods by name pattern

```ini
# Exact pod name
[api-pod]
type = k8s
pod = api-server-7d8f6c9b5-abc12
namespace = production

# Label selector (recommended - finds any pod matching the labels)
[api]
type = k8s
selector = app=api-server
namespace = production
container = app
context = prod-cluster

# Name pattern (grep-style matching)
[scheduler]
type = k8s
pod_grep = background-scheduler
namespace = production
container = resque-scheduler
context = prod-cluster
shell = bash
```

Priority order: `pod` > `selector` > `pod_grep`

This is useful for connecting to pods with dynamic names (e.g., deployments with random suffixes like `my-app-7d8f6c9b5-abc12`).

## Documentation

- [SSH Backend](docs/ssh.md) - SSH connection configuration and troubleshooting
- [Docker Backend](docs/docker.md) - Docker container access
- [Kubernetes Backend](docs/k8s.md) - Kubernetes pod access
- [Configuration Reference](docs/configuration.md) - Full configuration options
- [Commands Reference](docs/commands.md) - All CLI commands and flags

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development

```bash
# Build
make build

# Run tests
make test

# Generate man pages
make docs

# Build release packages
make dist
```

## License

MIT
