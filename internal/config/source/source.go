// Package source provides interfaces and implementations for loading hosts from various file formats.
package source

// HostEntry represents a single host loaded from a source.
// This is an intermediate representation before conversion to HostConfig.
type HostEntry struct {
	Name       string            // Host name/alias
	Type       string            // Backend type: ssh, docker, k8s
	Properties map[string]string // All properties as key-value pairs
}

// HostSource defines the interface for loading hosts from different file formats.
type HostSource interface {
	// Name returns the unique identifier for this source (e.g., "ini", "ansible", "sshconfig")
	Name() string

	// FilePatterns returns glob patterns for files this source can handle (e.g., "*.ini", "inventory.yml")
	FilePatterns() []string

	// CanLoad returns true if this source can handle the given file
	CanLoad(path string) bool

	// Load reads hosts from the given file path
	Load(path string) ([]*HostEntry, error)

	// IsWritable returns true if this source supports write operations (add, remove, update hosts)
	IsWritable() bool

	// DefaultPaths returns the default paths this source searches when not given an explicit path
	DefaultPaths() []string
}
