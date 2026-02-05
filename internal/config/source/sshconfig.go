package source

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// SSHConfigSource loads hosts from SSH config files
type SSHConfigSource struct{}

func init() {
	Register(&SSHConfigSource{})
}

// Name returns "sshconfig"
func (s *SSHConfigSource) Name() string {
	return "sshconfig"
}

// FilePatterns returns patterns for SSH config files
func (s *SSHConfigSource) FilePatterns() []string {
	return []string{"config", "ssh_config"}
}

// CanLoad returns true for SSH config files
func (s *SSHConfigSource) CanLoad(path string) bool {
	base := filepath.Base(path)
	dir := filepath.Base(filepath.Dir(path))

	// ~/.ssh/config
	if dir == ".ssh" && base == "config" {
		return true
	}

	// /etc/ssh/ssh_config
	if dir == "ssh" && (base == "ssh_config" || base == "config") {
		return true
	}

	return false
}

// Load reads hosts from an SSH config file
func (s *SSHConfigSource) Load(path string) ([]*HostEntry, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	entries := make([]*HostEntry, 0)
	var currentHost *sshHostBlock

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse key value (handle both "Key Value" and "Key=Value" formats)
		key, value := parseSSHConfigLine(line)
		if key == "" {
			continue
		}

		key = strings.ToLower(key)

		// New Host block starts
		if key == "host" {
			// Save previous host if valid
			if currentHost != nil && currentHost.isValid() {
				entry := currentHost.toHostEntry()
				if entry != nil {
					entries = append(entries, entry)
				}
			}

			// Start new host block
			currentHost = &sshHostBlock{
				patterns: strings.Fields(value),
				props:    make(map[string]string),
			}
			continue
		}

		// Add property to current host
		if currentHost != nil {
			currentHost.props[key] = value
		}
	}

	// Don't forget the last host
	if currentHost != nil && currentHost.isValid() {
		entry := currentHost.toHostEntry()
		if entry != nil {
			entries = append(entries, entry)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return entries, nil
}

// IsWritable returns false - SSH config files are read-only for hop
func (s *SSHConfigSource) IsWritable() bool {
	return false
}

// DefaultPaths returns the default SSH config paths
func (s *SSHConfigSource) DefaultPaths() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	return []string{
		filepath.Join(home, ".ssh", "config"),
	}
}

// sshHostBlock represents a parsed Host block from SSH config
type sshHostBlock struct {
	patterns []string          // Host patterns (can be multiple)
	props    map[string]string // SSH config properties
}

// isValid returns true if this host block should be imported
func (h *sshHostBlock) isValid() bool {
	// Skip wildcard-only patterns
	for _, pattern := range h.patterns {
		if pattern == "*" {
			return false
		}
		// Skip patterns with wildcards
		if strings.Contains(pattern, "*") || strings.Contains(pattern, "?") {
			continue
		}
		return true
	}
	return false
}

// toHostEntry converts the SSH config block to a HostEntry
func (h *sshHostBlock) toHostEntry() *HostEntry {
	// Find the first non-wildcard pattern to use as the name
	var name string
	for _, pattern := range h.patterns {
		if !strings.Contains(pattern, "*") && !strings.Contains(pattern, "?") {
			name = pattern
			break
		}
	}
	if name == "" {
		return nil
	}

	props := make(map[string]string)

	// Map SSH config options to hop properties
	if v, ok := h.props["hostname"]; ok {
		props["host"] = v
	}
	if v, ok := h.props["user"]; ok {
		props["user"] = v
	}
	if v, ok := h.props["port"]; ok {
		if _, err := strconv.Atoi(v); err == nil {
			props["port"] = v
		}
	}
	if v, ok := h.props["identityfile"]; ok {
		// Expand ~ in identity file path
		props["identity"] = expandTilde(v)
	}
	if v, ok := h.props["proxyjump"]; ok {
		props["jump"] = v
	}
	if v, ok := h.props["forwardagent"]; ok {
		if strings.ToLower(v) == "yes" {
			props["agent_forward"] = "yes"
		}
	}

	// If no HostName specified, use the alias as the host
	if props["host"] == "" {
		props["host"] = name
	}

	return &HostEntry{
		Name:       name,
		Type:       "ssh",
		Properties: props,
	}
}

// parseSSHConfigLine parses a line from SSH config into key and value
func parseSSHConfigLine(line string) (string, string) {
	// Handle "Key=Value" format
	if idx := strings.Index(line, "="); idx > 0 {
		return strings.TrimSpace(line[:idx]), strings.TrimSpace(line[idx+1:])
	}

	// Handle "Key Value" format (space or tab separated)
	fields := strings.Fields(line)
	if len(fields) >= 2 {
		return fields[0], strings.Join(fields[1:], " ")
	}
	if len(fields) == 1 {
		return fields[0], ""
	}

	return "", ""
}

// expandTilde expands ~ to the user's home directory
func expandTilde(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}
