package config

import "github.com/geoff/hop/internal/backend"

// HostConfig represents the parsed configuration for a single host
type HostConfig struct {
	// Common fields
	Name    string // Section name from INI file
	Type    string // Backend type: ssh, docker, k8s (defaults to "ssh")
	Default bool   // Whether this is the default host

	// SSH fields
	Host         string // Hostname or IP address
	User         string // SSH username
	Port         int    // SSH port (default: 22)
	Identity     string // Path to SSH identity file
	Jump         string // Jump host for ProxyJump (-J)
	AgentForward bool   // Enable SSH agent forwarding (-A)

	// Docker fields
	Container  string // Container name or ID
	Shell      string // Shell to use (default: /bin/sh)
	Label      string // Container label selector
	Image      string // Container image name
	ImageGrep  string // Container image pattern

	// K8s fields
	Namespace  string // Kubernetes namespace (default: "default")
	Pod        string // Pod name
	Context    string // Kubernetes context
	Selector   string // Label selector for pods
	PodGrep    string // Pod name pattern
	Deployment string // Deployment name for pod discovery

	// Port forwarding
	LocalPort  int // Local port for forwarding
	RemotePort int // Remote port for forwarding

	// Inheritance
	Extends string // Name of host to inherit from
}

// ToMap returns the host configuration as a map of property names to values
func (h *HostConfig) ToMap() map[string]string {
	props := make(map[string]string)

	if h.Type != "" && h.Type != "ssh" {
		props["type"] = h.Type
	}
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
	if h.Jump != "" {
		props["jump"] = h.Jump
	}
	if h.AgentForward {
		props["agent_forward"] = "yes"
	}
	if h.Container != "" {
		props["container"] = h.Container
	}
	if h.Shell != "" && h.Shell != "/bin/sh" {
		props["shell"] = h.Shell
	}
	if h.Label != "" {
		props["label"] = h.Label
	}
	if h.Image != "" {
		props["image"] = h.Image
	}
	if h.ImageGrep != "" {
		props["image_grep"] = h.ImageGrep
	}
	if h.Namespace != "" && h.Namespace != "default" {
		props["namespace"] = h.Namespace
	}
	if h.Pod != "" {
		props["pod"] = h.Pod
	}
	if h.Context != "" {
		props["context"] = h.Context
	}
	if h.Selector != "" {
		props["selector"] = h.Selector
	}
	if h.PodGrep != "" {
		props["pod_grep"] = h.PodGrep
	}
	if h.Deployment != "" {
		props["deployment"] = h.Deployment
	}
	if h.LocalPort != 0 {
		props["local_port"] = itoa(h.LocalPort)
	}
	if h.RemotePort != 0 {
		props["remote_port"] = itoa(h.RemotePort)
	}
	if h.Default {
		props["default"] = "yes"
	}
	if h.Extends != "" {
		props["extends"] = h.Extends
	}

	return props
}

// ToHost converts a HostConfig to a backend.Host
func (h *HostConfig) ToHost() *backend.Host {
	props := make(map[string]string)

	// SSH fields
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
	if h.Jump != "" {
		props["jump"] = h.Jump
	}
	if h.AgentForward {
		props["agent_forward"] = "yes"
	}

	// Docker fields
	if h.Container != "" {
		props["container"] = h.Container
	}
	if h.Shell != "" {
		props["shell"] = h.Shell
	}
	if h.Label != "" {
		props["label"] = h.Label
	}
	if h.Image != "" {
		props["image"] = h.Image
	}
	if h.ImageGrep != "" {
		props["image_grep"] = h.ImageGrep
	}

	// K8s fields
	if h.Namespace != "" {
		props["namespace"] = h.Namespace
	}
	if h.Pod != "" {
		props["pod"] = h.Pod
	}
	if h.Context != "" {
		props["context"] = h.Context
	}
	if h.Selector != "" {
		props["selector"] = h.Selector
	}
	if h.PodGrep != "" {
		props["pod_grep"] = h.PodGrep
	}
	if h.Deployment != "" {
		props["deployment"] = h.Deployment
	}

	// Port forwarding
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
