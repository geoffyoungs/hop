# SSH Backend

The SSH backend provides secure shell connections to remote servers using the standard OpenSSH client.

## Configuration Fields

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `type` | No | `ssh` | Backend type (can be omitted for SSH) |
| `host` | **Yes** | - | Hostname or IP address |
| `user` | No | Current user | SSH username |
| `port` | No | 22 | SSH port |
| `identity` | No | - | Path to SSH identity file (private key) |

## Examples

### Basic SSH Connection

```ini
[production]
host = 192.168.1.100
user = admin
```

### Custom Port and Identity File

```ini
[staging]
host = staging.example.com
user = deploy
port = 2222
identity = ~/.ssh/staging_key
```

### IP Address with User

```ini
[webserver]
host = 10.0.0.50
user = root
```

## Usage

### Connect to a Host

```bash
hop production
```

### Copy Files

```bash
# Upload file to remote server
hop --copy localfile.txt production:/remote/path/

# Download file from remote server
hop --copy production:/remote/file.txt ./local/
```

### Port Forwarding

```bash
# Forward local port 8080 to remote port 80
hop --forward "production 8080:80"
```

## Requirements

- OpenSSH client (`ssh`, `scp`) must be installed
- Run `hop --check` to verify installation

## Troubleshooting

### Connection Refused

- Verify the host is reachable: `ping <host>`
- Check if SSH is running on the remote server
- Verify the port is correct

### Permission Denied

- Check username is correct
- Verify SSH key permissions: `chmod 600 ~/.ssh/id_rsa`
- Ensure your public key is in the remote `~/.ssh/authorized_keys`

### Host Key Verification Failed

- The remote server's host key has changed
- If expected, remove the old key: `ssh-keygen -R <host>`

### Identity File Issues

- Verify the path to the identity file is correct
- Check file permissions: `chmod 600 <identity_file>`
- Ensure the corresponding public key is on the server
