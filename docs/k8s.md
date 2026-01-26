# Kubernetes Backend

The Kubernetes backend provides shell access to pods in Kubernetes clusters using `kubectl`.

## Configuration Fields

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `type` | **Yes** | - | Must be `k8s` |
| `pod` | One of these | - | Exact pod name |
| `selector` | One of these | - | Label selector (e.g., `app=myapp`) |
| `pod_grep` | One of these | - | Name pattern to grep for |
| `namespace` | No | `default` | Kubernetes namespace |
| `container` | No | - | Container name (required for multi-container pods) |
| `context` | No | Current context | Kubernetes context |
| `shell` | No | `/bin/sh` | Shell to execute |

**Note:** You must specify one of `pod`, `selector`, or `pod_grep`. Priority order: `pod` > `selector` > `pod_grep`.

## Examples

### Basic Pod Access

```ini
[myapp]
type = k8s
pod = my-application-7d8f9b6c5-x2j4k
```

### Pod in Specific Namespace

```ini
[production-api]
type = k8s
namespace = production
pod = api-server-5f6d7c8b9-abc12
shell = /bin/bash
```

### Multi-Container Pod

```ini
[sidecar-app]
type = k8s
namespace = default
pod = my-app-with-sidecar-12345
container = main-app
```

### Multi-Cluster Setup

```ini
[staging-api]
type = k8s
context = staging-cluster
namespace = api
pod = api-server-abc123

[production-api]
type = k8s
context = production-cluster
namespace = api
pod = api-server-xyz789
```

## Dynamic Pod Discovery

When working with deployments, pod names include random suffixes (e.g., `myapp-7d8f9b6c5-x2j4k`). Rather than updating the config every time a pod restarts, use `selector` or `pod_grep` for dynamic discovery.

### Using Label Selectors (Recommended)

Label selectors use Kubernetes-native pod selection. This is the recommended approach as it's reliable and follows Kubernetes best practices.

```ini
[api]
type = k8s
selector = app=api-server
namespace = production
container = app
context = prod-cluster
```

This runs: `kubectl get pods -l app=api-server -o name` and uses the first matching pod.

You can use any valid Kubernetes label selector:

```ini
# Single label
selector = app=myapp

# Multiple labels (AND)
selector = app=myapp,environment=prod

# Set-based selectors
selector = app in (web,api)
```

### Using Name Patterns

If your pods don't have consistent labels, use `pod_grep` to match by name pattern:

```ini
[scheduler]
type = k8s
pod_grep = background-scheduler
namespace = production
container = resque-scheduler
shell = bash
```

This runs: `kubectl get pods -o name` and finds the first pod containing "background-scheduler" in its name.

### Real-World Example

Replace shell scripts like this:

```bash
#!/bin/bash
kubectl --context=prod-cluster exec \
  $(kubectl --context=prod-cluster get pods | grep web-scheduler | awk '{print $1}') \
  -c resque-scheduler -it -- bash
```

With a simple hop config:

```ini
[web-scheduler]
type = k8s
context = prod-cluster
pod_grep = web-scheduler
container = resque-scheduler
shell = bash
```

Then just run:

```bash
hop web-scheduler
```

## Usage

### Connect to a Pod

```bash
hop myapp
```

### Copy Files

```bash
# Upload file to pod
hop --copy localfile.txt myapp:/app/

# Download file from pod
hop --copy myapp:/app/config.yaml ./
```

### Port Forwarding

```bash
# Forward local port 8080 to pod port 80
hop --forward "myapp 8080:80"
```

## Requirements

- `kubectl` must be installed
- Valid kubeconfig with cluster access
- Run `hop --check` to verify installation

## Working with Multiple Clusters

Use the `context` field to connect to different clusters:

```ini
[dev-frontend]
type = k8s
context = dev-cluster
namespace = frontend
pod = web-app-12345

[prod-frontend]
type = k8s
context = prod-cluster
namespace = frontend
pod = web-app-67890
```

List available contexts:
```bash
kubectl config get-contexts
```

## Troubleshooting

### Pod Not Found

- Verify pod name: `kubectl get pods -n <namespace>`
- Pod names include random suffixes (e.g., `myapp-7d8f9b6c5-x2j4k`)
- Check if pod is running: `kubectl get pods -n <namespace> | grep <pod>`

### Container Not Found

- List containers in pod: `kubectl get pod <pod> -n <namespace> -o jsonpath='{.spec.containers[*].name}'`
- Specify the correct container name in config

### Context Not Found

- List contexts: `kubectl config get-contexts`
- Set correct context: `kubectl config use-context <context>`

### Namespace Not Found

- List namespaces: `kubectl get namespaces`
- Verify namespace spelling

### Permission Denied

- Check RBAC permissions: `kubectl auth can-i exec pods -n <namespace>`
- Contact cluster administrator for access

### Shell Not Available

- Try different shells: `/bin/sh`, `/bin/bash`, `/bin/ash`
- Some minimal images only have `/bin/sh`
