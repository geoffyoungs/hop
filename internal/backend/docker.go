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
	container := host.Properties["container"]
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
	container := host.Properties["container"]

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
	if host.Properties["container"] == "" {
		return fmt.Errorf("container is required for Docker connections")
	}
	return nil
}

func parseDockerVersion(output string) string {
	re := regexp.MustCompile(`Docker version ([\d.]+)`)
	matches := re.FindStringSubmatch(output)
	if len(matches) > 1 {
		return matches[1]
	}
	return strings.TrimSpace(output)
}
