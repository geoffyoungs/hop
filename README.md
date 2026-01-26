# hop

A CLI tool for connecting to named hosts using an INI configuration file.

Supports SSH, Docker, and Kubernetes backends.

## Installation

### Homebrew (macOS/Linux)
```bash
brew install geoff/tap/hop
```

### Go
```bash
go install github.com/geoff/hop/cmd/hop@latest
```

### From releases
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

## Usage

### Connect to a host
```bash
hop production          # SSH to production server
hop mycontainer         # Shell into Docker container
hop mypod               # Exec into Kubernetes pod
```

### List configured hosts
```bash
hop --list
```

### Check available backends
```bash
hop --check
```

### Copy files
```bash
hop --copy localfile.txt production:/remote/path    # Upload
hop --copy production:/remote/file.txt ./local      # Download
```

### Port forwarding
```bash
hop --forward "production 8080:80"    # Forward localhost:8080 to remote:80
```

### Shell completion
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

## License

MIT
