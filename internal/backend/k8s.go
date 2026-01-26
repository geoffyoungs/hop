package backend

import (
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
	args := k.buildBaseArgs(host)
	args = append(args, "exec", "-it")

	pod := host.Properties["pod"]
	args = append(args, pod)

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
	args := k.buildBaseArgs(host)
	args = append(args, "cp")

	pod := host.Properties["pod"]
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
	args := k.buildBaseArgs(host)
	args = append(args, "port-forward")

	pod := host.Properties["pod"]
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
	if host.Properties["pod"] == "" {
		return fmt.Errorf("pod is required for Kubernetes connections")
	}
	return nil
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
