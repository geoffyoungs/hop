# Docker Backend

The Docker backend provides shell access to running Docker containers.

## Configuration Fields

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `type` | **Yes** | - | Must be `docker` |
| `container` | **Yes** | - | Container name or ID |
| `shell` | No | `/bin/sh` | Shell to execute |

## Examples

### Basic Container Access

```ini
[myapp]
type = docker
container = my_application
```

### Container with Bash Shell

```ini
[webserver]
type = docker
container = nginx_server
shell = /bin/bash
```

### Container by ID

```ini
[database]
type = docker
container = a1b2c3d4e5f6
shell = /bin/sh
```

## Usage

### Connect to a Container

```bash
hop myapp
```

### Copy Files

```bash
# Upload file to container
hop --copy localfile.txt myapp:/app/

# Download file from container
hop --copy myapp:/app/config.json ./
```

## Requirements

- Docker CLI must be installed
- Docker daemon must be running
- Run `hop --check` to verify installation

## Limitations

- **Port Forwarding**: Docker containers don't support dynamic port forwarding through hop. Use `docker run -p` when starting containers, or configure ports in Docker Compose.
- **Container Must Be Running**: The target container must be in a running state.

## Troubleshooting

### Container Not Found

- Verify container name: `docker ps`
- Check if using correct container ID or name
- Ensure container is running, not stopped

### Shell Not Found

- The default shell `/bin/sh` might not exist in minimal images
- Try specifying a different shell: `shell = /bin/bash` or `shell = /bin/ash`

### Permission Denied

- Check Docker daemon permissions
- User may need to be in the `docker` group: `sudo usermod -aG docker $USER`

### Docker Daemon Not Running

- Start Docker daemon: `sudo systemctl start docker`
- Check Docker Desktop is running (macOS/Windows)
- Verify with: `docker info`
