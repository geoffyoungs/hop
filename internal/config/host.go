package config

import "github.com/geoff/hop/internal/backend"

// HostConfig represents the parsed configuration for a single host
type HostConfig struct {
	// Common fields
	Name    string // Section name from INI file
	Type    string // Backend type: ssh, docker, k8s (defaults to "ssh")
	Default bool   // Whether this is the default host

	// SSH fields
	Host     string // Hostname or IP address
	User     string // SSH username
	Port     int    // SSH port (default: 22)
	Identity string // Path to SSH identity file

	// Docker fields
	Container string // Container name or ID
	Shell     string // Shell to use (default: /bin/sh)

	// K8s fields
	Namespace string // Kubernetes namespace (default: "default")
	Pod       string // Pod name
	Context   string // Kubernetes context

	// Port forwarding
	LocalPort  int // Local port for forwarding
	RemotePort int // Remote port for forwarding
}

// ToHost converts a HostConfig to a backend.Host
func (h *HostConfig) ToHost() *backend.Host {
	props := make(map[string]string)

	// Add all non-empty fields
	if h.Host != "" {
		props["host"] = h.Host
	}
	if h.User != "" {
		props["user"] = h.User
	}
	if h.Port != 0 {
		props["port"] = itoa(h.Port)
	}
	if h.Identity != "" {
		props["identity"] = h.Identity
	}
	if h.Container != "" {
		props["container"] = h.Container
	}
	if h.Shell != "" {
		props["shell"] = h.Shell
	}
	if h.Namespace != "" {
		props["namespace"] = h.Namespace
	}
	if h.Pod != "" {
		props["pod"] = h.Pod
	}
	if h.Context != "" {
		props["context"] = h.Context
	}
	if h.LocalPort != 0 {
		props["local_port"] = itoa(h.LocalPort)
	}
	if h.RemotePort != 0 {
		props["remote_port"] = itoa(h.RemotePort)
	}

	hostType := h.Type
	if hostType == "" {
		hostType = "ssh" // Default to SSH
	}

	return &backend.Host{
		Name:       h.Name,
		Type:       hostType,
		Properties: props,
	}
}

func itoa(i int) string {
	if i == 0 {
		return ""
	}
	// Simple int to string conversion
	if i < 0 {
		return "-" + uitoa(uint(-i))
	}
	return uitoa(uint(i))
}

func uitoa(val uint) string {
	if val == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf) - 1
	for val > 0 {
		buf[i] = byte('0' + val%10)
		val /= 10
		i--
	}
	return string(buf[i+1:])
}
