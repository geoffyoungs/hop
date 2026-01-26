package backend

import (
	"context"
)

// Host represents a configured host entry from the INI file
type Host struct {
	Name       string
	Type       string            // ssh, docker, k8s
	Properties map[string]string // Backend-specific properties
}

// CheckResult contains the result of a backend dependency check
type CheckResult struct {
	Available bool
	Version   string
	Missing   []string // Missing required tools
	Warnings  []string // Non-fatal issues
}

// Backend defines the interface all connection backends must implement
type Backend interface {
	// Name returns the backend identifier (e.g., "ssh", "docker", "k8s")
	Name() string

	// Connect establishes an interactive shell session
	Connect(ctx context.Context, host *Host) error

	// Copy transfers files to/from the remote host
	// direction: "to" (local->remote) or "from" (remote->local)
	Copy(ctx context.Context, host *Host, localPath, remotePath, direction string) error

	// ForwardPort sets up port forwarding (if supported)
	// Returns ErrNotSupported if the backend doesn't support port forwarding
	ForwardPort(ctx context.Context, host *Host, localPort, remotePort int) error

	// Check verifies the backend's dependencies are installed
	Check() (*CheckResult, error)

	// Validate ensures host configuration is valid for this backend
	Validate(host *Host) error
}
