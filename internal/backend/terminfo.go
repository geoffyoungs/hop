package backend

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
)

// TerminfoSyncer is an optional interface backends can implement
// to support syncing terminal info to the remote host
type TerminfoSyncer interface {
	SyncTerminfo(ctx context.Context, host *Host) error
}

// GetTerminfo runs infocmp to get the current terminal's terminfo entry
func GetTerminfo() ([]byte, error) {
	term := os.Getenv("TERM")
	if term == "" {
		return nil, fmt.Errorf("TERM environment variable not set")
	}

	cmd := exec.Command("infocmp", "-x")
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("infocmp failed: %s", exitErr.Stderr)
		}
		return nil, fmt.Errorf("infocmp failed: %w", err)
	}

	return output, nil
}

// SyncTerminfo syncs terminfo to the remote host if the backend supports it
func SyncTerminfo(ctx context.Context, b Backend, host *Host) error {
	syncer, ok := b.(TerminfoSyncer)
	if !ok {
		return fmt.Errorf("backend %q does not support terminfo sync", b.Name())
	}
	return syncer.SyncTerminfo(ctx, host)
}

// SSH backend terminfo sync
func (s *SSHBackend) SyncTerminfo(ctx context.Context, host *Host) error {
	terminfo, err := GetTerminfo()
	if err != nil {
		return err
	}

	args := s.buildSSHArgs(host)
	args = append(args, s.buildDestination(host), "tic", "-x", "-")

	cmd := exec.CommandContext(ctx, "ssh", args...)
	cmd.Stdin = bytes.NewReader(terminfo)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// Docker backend terminfo sync
func (d *DockerBackend) SyncTerminfo(ctx context.Context, host *Host) error {
	terminfo, err := GetTerminfo()
	if err != nil {
		return err
	}

	container := host.Properties["container"]
	args := []string{"exec", "-i", container, "tic", "-x", "-"}

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdin = bytes.NewReader(terminfo)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// K8s backend terminfo sync
func (k *K8sBackend) SyncTerminfo(ctx context.Context, host *Host) error {
	pod, err := k.resolvePod(ctx, host)
	if err != nil {
		return err
	}

	terminfo, err := GetTerminfo()
	if err != nil {
		return err
	}

	args := k.buildBaseArgs(host)
	args = append(args, "exec", "-i", pod)

	if container := host.Properties["container"]; container != "" {
		args = append(args, "-c", container)
	}

	args = append(args, "--", "tic", "-x", "-")

	cmd := exec.CommandContext(ctx, "kubectl", args...)
	cmd.Stdin = bytes.NewReader(terminfo)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
