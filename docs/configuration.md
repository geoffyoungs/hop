# Configuration Reference

Hop uses INI-format configuration files to define hosts.

## File Locations

| Flag | Path | Description |
|------|------|-------------|
| (default) | `~/.config/hop/hosts.ini` | User configuration (XDG compliant) |
| `--local` | `./hosts.ini` | Project-local configuration |
| `--config <path>` | Custom path | Explicit configuration file |

The `--local`, `--user`, and `--config` flags are mutually exclusive.

## INI Format

Each host is defined as an INI section:

```ini
[section-name]
key = value
key2 = value2
```

Section names become the host alias used with hop commands.

## Common Fields

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Backend type: `ssh`, `docker`, or `k8s` (default: `ssh`) |

## SSH Fields

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `host` | Yes | - | Hostname or IP address |
| `user` | No | - | SSH username |
| `port` | No | 22 | SSH port |
| `identity` | No | - | Path to private key file |

## Docker Fields

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `container` | Yes | - | Container name or ID |
| `shell` | No | `/bin/sh` | Shell to execute |

## Kubernetes Fields

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `pod` | Yes | - | Pod name |
| `namespace` | No | `default` | Kubernetes namespace |
| `container` | No | - | Container name (for multi-container pods) |
| `context` | No | - | Kubernetes context |
| `shell` | No | `/bin/sh` | Shell to execute |

## Port Forwarding Fields

| Field | Type | Description |
|-------|------|-------------|
| `local_port` | int | Local port for default forwarding |
| `remote_port` | int | Remote port for default forwarding |

## Complete Example

```ini
# SSH hosts
[production]
host = 192.168.1.100
user = admin

[staging]
type = ssh
host = staging.example.com
user = deploy
port = 2222
identity = ~/.ssh/staging_key

# Docker containers
[redis]
type = docker
container = redis_cache
shell = /bin/sh

[webapp]
type = docker
container = my_web_app
shell = /bin/bash

# Kubernetes pods
[api-prod]
type = k8s
context = production
namespace = api
pod = api-server-5f6d7c8b9-x2j4k
container = api

[api-staging]
type = k8s
context = staging
namespace = api
pod = api-server-abc123
shell = /bin/bash
```

## Adding Hosts via CLI

Use `--add` to add hosts without editing the file:

```bash
# Add SSH host
hop --add myserver host=10.0.0.1 user=admin

# Add Docker container
hop --add mycontainer type=docker container=my_app shell=/bin/bash

# Add Kubernetes pod
hop --add mypod type=k8s namespace=prod pod=my-pod-12345

# Add to local config
hop --local --add localserver host=localhost user=dev
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `XDG_CONFIG_HOME` | Base directory for user config (default: `~/.config`) |

## Tips

1. **Use descriptive names**: Choose host aliases that are easy to remember and type.

2. **Group by environment**: Use prefixes like `prod-`, `staging-`, `dev-` for organization.

3. **Local configs for projects**: Use `--local` for project-specific hosts that shouldn't be in your global config.

4. **Shell completion**: Enable shell completion for faster host name entry:
   ```bash
   eval "$(hop completion bash)"
   ```
