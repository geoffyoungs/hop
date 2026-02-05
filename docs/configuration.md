# Configuration Reference

Hop can load hosts from multiple configuration sources including INI files, Ansible inventory, SSH config, and Vagrant VMs.

## Host Sources

| Source | Description | Read | Write |
|--------|-------------|------|-------|
| `ini` | INI config files (`hosts.ini`) | Yes | Yes |
| `ansible` | Ansible inventory files (`inventory.yml`, `hosts.yml`) | Yes | No |
| `sshconfig` | SSH config files (`~/.ssh/config`) | Yes | No |
| `vagrant` | Vagrant VMs (via `vagrant ssh-config`) | Yes | No |

### Using Multiple Sources

```bash
# List hosts from all enabled sources
hop --list --all-sources

# List hosts from specific sources
hop --list --sources ini,ansible

# Connect using source-qualified name
hop ini:production
hop ansible:webserver
```

## File Locations

### INI Config Files

| Flag | Path | Description |
|------|------|-------------|
| (default) | `~/.config/hop/hosts.ini` | User configuration (XDG compliant) |
| `--local` | `./hosts.ini` | Project-local configuration |
| `--config <path>` | Custom path | Explicit configuration file |

The `--local`, `--user`, and `--config` flags are mutually exclusive.

### Settings File

Global settings are stored in `~/.config/hop/settings.toml`:

```toml
[sources]
ini = true        # Default: true
ansible = true    # Default: true
sshconfig = false # Default: false (opt-in, can be noisy)
vagrant = true    # Default: true

# Priority order for name collisions (first wins)
priority = ["ini", "ansible", "vagrant", "sshconfig"]

# Custom paths per source
[sources.paths]
ansible = ["~/.ansible/inventory.yml", "/etc/ansible/hosts"]
sshconfig = ["~/.ssh/config"]

[collisions]
strategy = "first"  # "first", "qualify", or "error"
```

### Collision Strategies

When multiple sources define the same host name:

| Strategy | Behavior |
|----------|----------|
| `first` | Use the first source in priority order (default) |
| `qualify` | Keep both as `source:hostname` (e.g., `ini:web`, `ansible:web`) |
| `error` | Fail and list conflicts |

## INI Format

Each host is defined as an INI section:

```ini
[section-name]
key = value
key2 = value2
```

Section names become the host alias used with hop commands.

### Inheritance

Hosts can inherit from other hosts using `extends`:

```ini
[base-k8s]
type = k8s
context = prod-cluster
namespace = production

[api]
extends = base-k8s
pod_grep = api-server

[worker]
extends = base-k8s
pod_grep = worker
```

### Environment Variables

Values support environment variable expansion:

```ini
[production]
host = $PROD_HOST
user = ${DEPLOY_USER}
identity = ~/.ssh/id_rsa
```

## Ansible Inventory Format

Hop can read Ansible YAML inventory files (`inventory.yml`, `hosts.yml`):

```yaml
all:
  hosts:
    webserver:
      ansible_host: 192.168.1.10
      ansible_user: deploy
      ansible_port: 22
    dbserver:
      ansible_host: 192.168.1.20
      ansible_user: admin
      ansible_ssh_private_key_file: ~/.ssh/db_key
  children:
    production:
      hosts:
        prodweb:
          ansible_host: 10.0.0.100
```

### Ansible to Hop Mapping

| Ansible Variable | Hop Property |
|-----------------|--------------|
| `ansible_host` | `host` |
| `ansible_user` | `user` |
| `ansible_port` | `port` |
| `ansible_ssh_private_key_file` | `identity` |

## SSH Config Format

Hop can read `~/.ssh/config` files (disabled by default):

```
Host myserver
    HostName 192.168.1.100
    User admin
    Port 2222
    IdentityFile ~/.ssh/mykey

Host jumpbox
    HostName jump.example.com
    User jump
    ProxyJump bastion
    ForwardAgent yes
```

### SSH Config to Hop Mapping

| SSH Config | Hop Property |
|------------|--------------|
| `HostName` | `host` |
| `User` | `user` |
| `Port` | `port` |
| `IdentityFile` | `identity` |
| `ProxyJump` | `jump` |
| `ForwardAgent` | `agent_forward` |

**Note:** Wildcard patterns (e.g., `Host *`) are ignored.

## Vagrant Discovery

Hop automatically discovers Vagrant VMs by running `vagrant ssh-config` in directories containing a `Vagrantfile`. VMs are named with a `vagrant-` prefix (e.g., `vagrant-default`, `vagrant-web`).

Enable vagrant discovery in settings:

```toml
[sources]
vagrant = true
```

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
| `jump` | No | - | Jump host for ProxyJump (-J) |
| `agent_forward` | No | no | Enable SSH agent forwarding (-A) |

## Docker Fields

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `container` | One of | - | Container name or ID |
| `label` | these | - | Container label selector (e.g., `app=myapp`) |
| `image` | required | - | Container image name (exact match) |
| `image_grep` | | - | Container image pattern (grep match) |
| `shell` | No | `/bin/sh` | Shell to execute |

Priority order: `container` > `label` > `image` > `image_grep`

## Kubernetes Fields

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `pod` | One of | - | Exact pod name |
| `selector` | these | - | Label selector (e.g., `app=myapp`) |
| `pod_grep` | required | - | Pod name pattern (grep match) |
| `deployment` | | - | Deployment name (finds first pod) |
| `namespace` | No | `default` | Kubernetes namespace |
| `container` | No | - | Container name (for multi-container pods) |
| `context` | No | - | Kubernetes context |
| `shell` | No | `/bin/sh` | Shell to execute |

Priority order: `pod` > `selector` > `pod_grep` > `deployment`

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
