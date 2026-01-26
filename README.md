# hop

[![Go Reference](https://pkg.go.dev/badge/github.com/geoff/hop.svg)](https://pkg.go.dev/github.com/geoff/hop)
[![Go Report Card](https://goreportcard.com/badge/github.com/geoff/hop)](https://goreportcard.com/report/github.com/geoff/hop)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A CLI tool for connecting to named hosts using an INI configuration file. Supports SSH, Docker, and Kubernetes backends.

## Quick Start

```bash
# Install
brew install geoff/tap/hop

# Add a host
hop --add production host=192.168.1.100 user=admin

# Connect
hop production
```

## Installation

### Homebrew (macOS/Linux)

```bash
brew install geoff/tap/hop
```

### Go

```bash
go install github.com/geoff/hop/cmd/hop@latest
```

### From Releases

Download the appropriate binary from the [releases page](https://github.com/geoff/hop/releases).

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
| `k8s` | `pod` | `namespace`, `container`, `context`, `shell` |

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
go build ./cmd/hop

# Run tests
go test ./...

# Generate man pages
go run ./cmd/gendocs

# Build release packages
goreleaser build --snapshot --clean
```

## License

MIT
