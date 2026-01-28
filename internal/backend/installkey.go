package backend

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

// KeyInstaller is an optional interface backends can implement
// to support installing SSH public keys on the remote host
type KeyInstaller interface {
	InstallKey(ctx context.Context, host *Host, pubKeyPath string) error
}

// InstallKey installs a public key on the remote host if the backend supports it
func InstallKey(ctx context.Context, b Backend, host *Host, pubKeyPath string) error {
	installer, ok := b.(KeyInstaller)
	if !ok {
		return fmt.Errorf("backend %q does not support key installation", b.Name())
	}
	return installer.InstallKey(ctx, host, pubKeyPath)
}

// DiscoverPublicKey finds the best public key to install for the given host.
// It checks the host's identity property first, then falls back to common key paths.
func DiscoverPublicKey(host *Host) (string, error) {
	// Check host's identity property with .pub suffix
	if identity := host.Properties["identity"]; identity != "" {
		pubPath := identity + ".pub"
		if _, err := os.Stat(pubPath); err == nil {
			return pubPath, nil
		}
	}

	// Fall back to common key paths
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}

	candidates := []string{
		home + "/.ssh/id_ed25519.pub",
		home + "/.ssh/id_ecdsa.pub",
		home + "/.ssh/id_rsa.pub",
		home + "/.ssh/id_dsa.pub",
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("no SSH public key found; generate one with: ssh-keygen -t ed25519")
}

// SSH backend key installation
func (s *SSHBackend) InstallKey(ctx context.Context, host *Host, pubKeyPath string) error {
	var args []string

	args = append(args, "-i", pubKeyPath)

	if port := host.Properties["port"]; port != "" && port != "22" {
		args = append(args, "-p", port)
	}
	if jump := host.Properties["jump"]; jump != "" {
		args = append(args, "-o", "ProxyJump="+jump)
	}

	args = append(args, s.buildDestination(host))

	cmd := exec.CommandContext(ctx, "ssh-copy-id", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
