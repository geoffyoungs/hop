package backend

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// K8sBackend implements the Backend interface for Kubernetes pods
type K8sBackend struct{}

func init() {
	Register(&K8sBackend{})
}

func (k *K8sBackend) Name() string {
	return "k8s"
}

func (k *K8sBackend) Connect(ctx context.Context, host *Host) error {
	pod, err := k.resolvePod(ctx, host)
	if err != nil {
		return err
	}

	args := k.buildBaseArgs(host)
	args = append(args, "exec", "-it", pod)

	if container := host.Properties["container"]; container != "" {
		args = append(args, "-c", container)
	}

	shell := host.Properties["shell"]
	if shell == "" {
		shell = "/bin/sh"
	}
	args = append(args, "--", shell)

	cmd := exec.CommandContext(ctx, "kubectl", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func (k *K8sBackend) Copy(ctx context.Context, host *Host, localPath, remotePath, direction string) error {
	pod, err := k.resolvePod(ctx, host)
	if err != nil {
		return err
	}

	args := k.buildBaseArgs(host)
	args = append(args, "cp")

	namespace := host.Properties["namespace"]
	if namespace == "" {
		namespace = "default"
	}

	podPath := fmt.Sprintf("%s/%s:%s", namespace, pod, remotePath)
	if container := host.Properties["container"]; container != "" {
		args = append(args, "-c", container)
	}

	if direction == "to" {
		args = append(args, localPath, podPath)
	} else {
		args = append(args, podPath, localPath)
	}

	cmd := exec.CommandContext(ctx, "kubectl", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func (k *K8sBackend) ForwardPort(ctx context.Context, host *Host, localPort, remotePort int) error {
	pod, err := k.resolvePod(ctx, host)
	if err != nil {
		return err
	}

	args := k.buildBaseArgs(host)
	args = append(args, "port-forward")
	args = append(args, fmt.Sprintf("pod/%s", pod))
	args = append(args, fmt.Sprintf("%d:%d", localPort, remotePort))

	cmd := exec.CommandContext(ctx, "kubectl", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func (k *K8sBackend) Check() (*CheckResult, error) {
	cmd := exec.Command("kubectl", "version", "--client", "-o", "json")
	output, err := cmd.Output()
	if err != nil {
		// Try older format
		cmd = exec.Command("kubectl", "version", "--client", "--short")
		output, err = cmd.Output()
		if err != nil {
			return &CheckResult{
				Available: false,
				Missing:   []string{"kubectl"},
			}, nil
		}
	}

	version := parseKubectlVersion(string(output))
	return &CheckResult{
		Available: true,
		Version:   version,
	}, nil
}

func (k *K8sBackend) Validate(host *Host) error {
	pod := host.Properties["pod"]
	selector := host.Properties["selector"]
	podGrep := host.Properties["pod_grep"]
	deployment := host.Properties["deployment"]

	if pod == "" && selector == "" && podGrep == "" && deployment == "" {
		return fmt.Errorf("one of pod, selector, pod_grep, or deployment is required for Kubernetes connections")
	}
	return nil
}

func (k *K8sBackend) BuildConnectCommand(ctx context.Context, host *Host) (string, []string, error) {
	pod, err := k.resolvePod(ctx, host)
	if err != nil {
		return "", nil, err
	}

	args := k.buildBaseArgs(host)
	args = append(args, "exec", "-it", pod)

	if container := host.Properties["container"]; container != "" {
		args = append(args, "-c", container)
	}

	shell := host.Properties["shell"]
	if shell == "" {
		shell = "/bin/sh"
	}
	args = append(args, "--", shell)

	return "kubectl", args, nil
}

func (k *K8sBackend) Exec(ctx context.Context, host *Host, command string) error {
	pod, err := k.resolvePod(ctx, host)
	if err != nil {
		return err
	}

	args := k.buildBaseArgs(host)
	args = append(args, "exec", pod)

	if container := host.Properties["container"]; container != "" {
		args = append(args, "-c", container)
	}

	// No -it for non-interactive
	args = append(args, "--", "/bin/sh", "-c", command)

	cmd := exec.CommandContext(ctx, "kubectl", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// resolvePod determines the pod name using the configured method.
// Priority: pod (exact) > selector (label) > deployment > pod_grep (name pattern)
func (k *K8sBackend) resolvePod(ctx context.Context, host *Host) (string, error) {
	// Direct pod name takes priority
	if pod := host.Properties["pod"]; pod != "" {
		return pod, nil
	}

	baseArgs := k.buildBaseArgs(host)

	// Label selector
	if selector := host.Properties["selector"]; selector != "" {
		args := append(baseArgs, "get", "pods", "-l", selector, "-o", "name", "--no-headers")
		cmd := exec.CommandContext(ctx, "kubectl", args...)
		output, err := cmd.Output()
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				return "", fmt.Errorf("failed to find pod with selector %q: %s", selector, exitErr.Stderr)
			}
			return "", fmt.Errorf("failed to find pod with selector %q: %w", selector, err)
		}

		pod := k.firstPodFromOutput(output)
		if pod == "" {
			return "", fmt.Errorf("no running pod found with selector %q", selector)
		}
		return pod, nil
	}

	// Deployment name - uses app label convention
	if deployment := host.Properties["deployment"]; deployment != "" {
		// First try app=<deployment> label (common convention)
		args := append(baseArgs, "get", "pods", "-l", "app="+deployment, "-o", "name", "--no-headers")
		cmd := exec.CommandContext(ctx, "kubectl", args...)
		output, err := cmd.Output()
		if err == nil {
			pod := k.firstPodFromOutput(output)
			if pod != "" {
				return pod, nil
			}
		}

		// Fallback: try app.kubernetes.io/name=<deployment>
		args = append(baseArgs[:len(baseArgs):len(baseArgs)], "get", "pods", "-l", "app.kubernetes.io/name="+deployment, "-o", "name", "--no-headers")
		cmd = exec.CommandContext(ctx, "kubectl", args...)
		output, err = cmd.Output()
		if err == nil {
			pod := k.firstPodFromOutput(output)
			if pod != "" {
				return pod, nil
			}
		}

		return "", fmt.Errorf("no running pod found for deployment %q", deployment)
	}

	// Pod name grep pattern
	if pattern := host.Properties["pod_grep"]; pattern != "" {
		args := append(baseArgs, "get", "pods", "-o", "name", "--no-headers")
		cmd := exec.CommandContext(ctx, "kubectl", args...)
		output, err := cmd.Output()
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				return "", fmt.Errorf("failed to list pods: %s", exitErr.Stderr)
			}
			return "", fmt.Errorf("failed to list pods: %w", err)
		}

		pod := k.findPodByPattern(output, pattern)
		if pod == "" {
			return "", fmt.Errorf("no running pod found matching pattern %q", pattern)
		}
		return pod, nil
	}

	return "", fmt.Errorf("no pod, selector, deployment, or pod_grep specified")
}

// firstPodFromOutput extracts the first pod name from kubectl output
func (k *K8sBackend) firstPodFromOutput(output []byte) string {
	scanner := bufio.NewScanner(bytes.NewReader(output))
	if scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// kubectl returns "pod/podname", strip the prefix
		return strings.TrimPrefix(line, "pod/")
	}
	return ""
}

// findPodByPattern finds the first pod matching the given pattern
func (k *K8sBackend) findPodByPattern(output []byte, pattern string) string {
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		podName := strings.TrimPrefix(line, "pod/")
		if strings.Contains(podName, pattern) {
			return podName
		}
	}
	return ""
}

func (k *K8sBackend) buildBaseArgs(host *Host) []string {
	var args []string

	if namespace := host.Properties["namespace"]; namespace != "" {
		args = append(args, "-n", namespace)
	}
	if context := host.Properties["context"]; context != "" {
		args = append(args, "--context", context)
	}

	return args
}

func parseKubectlVersion(output string) string {
	// Try to extract version from JSON output
	re := regexp.MustCompile(`"gitVersion":\s*"v?([\d.]+)"`)
	matches := re.FindStringSubmatch(output)
	if len(matches) > 1 {
		return matches[1]
	}

	// Try short format: "Client Version: v1.28.0"
	re = regexp.MustCompile(`Client Version:?\s*v?([\d.]+)`)
	matches = re.FindStringSubmatch(output)
	if len(matches) > 1 {
		return matches[1]
	}

	return strings.TrimSpace(output)
}
