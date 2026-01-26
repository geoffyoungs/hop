package backend

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// SSHBackend implements the Backend interface for SSH connections
type SSHBackend struct{}

func init() {
	Register(&SSHBackend{})
}

func (s *SSHBackend) Name() string {
	return "ssh"
}

func (s *SSHBackend) Connect(ctx context.Context, host *Host) error {
	args := s.buildSSHArgs(host)
	args = append(args, s.buildDestination(host))

	cmd := exec.CommandContext(ctx, "ssh", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func (s *SSHBackend) Copy(ctx context.Context, host *Host, localPath, remotePath, direction string) error {
	args := s.buildSCPArgs(host)

	dest := s.buildDestination(host)
	if direction == "to" {
		args = append(args, localPath, fmt.Sprintf("%s:%s", dest, remotePath))
	} else {
		args = append(args, fmt.Sprintf("%s:%s", dest, remotePath), localPath)
	}

	cmd := exec.CommandContext(ctx, "scp", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func (s *SSHBackend) ForwardPort(ctx context.Context, host *Host, localPort, remotePort int) error {
	args := s.buildSSHArgs(host)
	args = append(args,
		"-L", fmt.Sprintf("%d:localhost:%d", localPort, remotePort),
		"-N", // Don't execute remote command
		s.buildDestination(host),
	)

	cmd := exec.CommandContext(ctx, "ssh", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func (s *SSHBackend) Check() (*CheckResult, error) {
	cmd := exec.Command("ssh", "-V")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return &CheckResult{
			Available: false,
			Missing:   []string{"ssh"},
		}, nil
	}

	version := parseSSHVersion(string(output))
	return &CheckResult{
		Available: true,
		Version:   version,
	}, nil
}

func (s *SSHBackend) Validate(host *Host) error {
	if host.Properties["host"] == "" {
		return fmt.Errorf("host is required for SSH connections")
	}
	return nil
}

func (s *SSHBackend) buildSSHArgs(host *Host) []string {
	var args []string

	if port := host.Properties["port"]; port != "" && port != "22" {
		args = append(args, "-p", port)
	}
	if identity := host.Properties["identity"]; identity != "" {
		args = append(args, "-i", identity)
	}

	return args
}

func (s *SSHBackend) buildSCPArgs(host *Host) []string {
	var args []string

	if port := host.Properties["port"]; port != "" && port != "22" {
		args = append(args, "-P", port) // Note: scp uses uppercase -P
	}
	if identity := host.Properties["identity"]; identity != "" {
		args = append(args, "-i", identity)
	}

	return args
}

func (s *SSHBackend) buildDestination(host *Host) string {
	dest := host.Properties["host"]
	if user := host.Properties["user"]; user != "" {
		dest = user + "@" + dest
	}
	return dest
}

func parseSSHVersion(output string) string {
	// SSH version output is typically: "OpenSSH_9.0p1, LibreSSL 3.3.6"
	re := regexp.MustCompile(`OpenSSH[_\s]*([\d.]+\w*)`)
	matches := re.FindStringSubmatch(output)
	if len(matches) > 1 {
		return matches[1]
	}
	return strings.TrimSpace(output)
}
