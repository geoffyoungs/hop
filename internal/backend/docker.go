package backend

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// DockerBackend implements the Backend interface for Docker containers
type DockerBackend struct{}

func init() {
	Register(&DockerBackend{})
}

func (d *DockerBackend) Name() string {
	return "docker"
}

func (d *DockerBackend) Connect(ctx context.Context, host *Host) error {
	container, err := d.resolveContainer(ctx, host)
	if err != nil {
		return err
	}

	shell := host.Properties["shell"]
	if shell == "" {
		shell = "/bin/sh"
	}

	args := []string{"exec", "-it", container, shell}

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func (d *DockerBackend) Copy(ctx context.Context, host *Host, localPath, remotePath, direction string) error {
	container, err := d.resolveContainer(ctx, host)
	if err != nil {
		return err
	}

	var args []string
	if direction == "to" {
		args = []string{"cp", localPath, fmt.Sprintf("%s:%s", container, remotePath)}
	} else {
		args = []string{"cp", fmt.Sprintf("%s:%s", container, remotePath), localPath}
	}

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// resolveContainer determines the container name using the configured method.
// Priority: container (exact) > label > image > image_grep
func (d *DockerBackend) resolveContainer(ctx context.Context, host *Host) (string, error) {
	// Direct container name takes priority
	if container := host.Properties["container"]; container != "" {
		return container, nil
	}

	// Label selector
	if label := host.Properties["label"]; label != "" {
		cmd := exec.CommandContext(ctx, "docker", "ps", "--filter", "label="+label, "--format", "{{.Names}}", "-n", "1")
		output, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("failed to find container with label %q: %w", label, err)
		}
		container := strings.TrimSpace(string(output))
		if container == "" {
			return "", fmt.Errorf("no running container found with label %q", label)
		}
		return container, nil
	}

	// Image name (exact match via ancestor filter)
	if image := host.Properties["image"]; image != "" {
		cmd := exec.CommandContext(ctx, "docker", "ps", "--filter", "ancestor="+image, "--format", "{{.Names}}", "-n", "1")
		output, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("failed to find container with image %q: %w", image, err)
		}
		container := strings.TrimSpace(string(output))
		if container == "" {
			return "", fmt.Errorf("no running container found with image %q", image)
		}
		return container, nil
	}

	// Image grep pattern
	if pattern := host.Properties["image_grep"]; pattern != "" {
		cmd := exec.CommandContext(ctx, "docker", "ps", "--format", "{{.Names}}\t{{.Image}}")
		output, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("failed to list containers: %w", err)
		}

		for _, line := range strings.Split(string(output), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, "\t", 2)
			if len(parts) == 2 && strings.Contains(parts[1], pattern) {
				return parts[0], nil
			}
		}
		return "", fmt.Errorf("no running container found with image matching %q", pattern)
	}

	return "", fmt.Errorf("no container, label, image, or image_grep specified")
}

func (d *DockerBackend) ForwardPort(ctx context.Context, host *Host, localPort, remotePort int) error {
	// Docker doesn't support dynamic port forwarding to running containers
	// This would require socat or similar inside the container
	return ErrNotSupported
}

func (d *DockerBackend) Check() (*CheckResult, error) {
	cmd := exec.Command("docker", "version", "--format", "{{.Client.Version}}")
	output, err := cmd.Output()
	if err != nil {
		return &CheckResult{
			Available: false,
			Missing:   []string{"docker"},
		}, nil
	}

	version := strings.TrimSpace(string(output))

	// Also check if Docker daemon is running
	checkCmd := exec.Command("docker", "info")
	if err := checkCmd.Run(); err != nil {
		return &CheckResult{
			Available: true,
			Version:   version,
			Warnings:  []string{"Docker daemon is not running"},
		}, nil
	}

	return &CheckResult{
		Available: true,
		Version:   version,
	}, nil
}

func (d *DockerBackend) Validate(host *Host) error {
	container := host.Properties["container"]
	label := host.Properties["label"]
	image := host.Properties["image"]
	imageGrep := host.Properties["image_grep"]

	if container == "" && label == "" && image == "" && imageGrep == "" {
		return fmt.Errorf("one of container, label, image, or image_grep is required for Docker connections")
	}
	return nil
}

func (d *DockerBackend) BuildConnectCommand(ctx context.Context, host *Host) (string, []string, error) {
	container, err := d.resolveContainer(ctx, host)
	if err != nil {
		return "", nil, err
	}

	shell := host.Properties["shell"]
	if shell == "" {
		shell = "/bin/sh"
	}

	args := []string{"exec", "-it", container, shell}
	return "docker", args, nil
}

func (d *DockerBackend) Exec(ctx context.Context, host *Host, command string) error {
	container, err := d.resolveContainer(ctx, host)
	if err != nil {
		return err
	}

	// Split command into shell execution (no -it for non-interactive)
	args := []string{"exec", container, "/bin/sh", "-c", command}

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func parseDockerVersion(output string) string {
	re := regexp.MustCompile(`Docker version ([\d.]+)`)
	matches := re.FindStringSubmatch(output)
	if len(matches) > 1 {
		return matches[1]
	}
	return strings.TrimSpace(output)
}
