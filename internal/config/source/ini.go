package source

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/ini.v1"
)

// INISource loads hosts from INI configuration files
type INISource struct{}

func init() {
	Register(&INISource{})
}

// Name returns "ini"
func (s *INISource) Name() string {
	return "ini"
}

// FilePatterns returns patterns for INI files
func (s *INISource) FilePatterns() []string {
	return []string{"*.ini", "*.conf", "hosts.ini", "hosts.conf"}
}

// CanLoad returns true for .ini and .conf files
func (s *INISource) CanLoad(path string) bool {
	base := filepath.Base(path)
	ext := strings.ToLower(filepath.Ext(path))

	// Accept hosts.ini or hosts.conf explicitly
	if base == "hosts.ini" || base == "hosts.conf" {
		return true
	}

	// Accept any .ini file
	return ext == ".ini"
}

// Load reads hosts from an INI file
func (s *INISource) Load(path string) ([]*HostEntry, error) {
	cfg, err := ini.Load(path)
	if err != nil {
		return nil, err
	}

	// Pass 1: Load all sections into raw maps
	rawHosts := make(map[string]map[string]string)
	for _, section := range cfg.Sections() {
		name := section.Name()
		if name == "DEFAULT" {
			continue
		}

		props := make(map[string]string)
		for _, key := range section.Keys() {
			props[key.Name()] = key.String()
		}
		rawHosts[name] = props
	}

	// Pass 2: Resolve inheritance chains and expand env vars
	resolvedHosts := make(map[string]map[string]string)
	for name := range rawHosts {
		resolved, err := resolveInheritance(name, rawHosts, make(map[string]bool))
		if err != nil {
			return nil, err
		}
		resolvedHosts[name] = resolved
	}

	// Pass 3: Convert to HostEntry structs
	entries := make([]*HostEntry, 0, len(resolvedHosts))
	for name, props := range resolvedHosts {
		hostType := props["type"]
		if hostType == "" {
			hostType = "ssh"
		}

		entry := &HostEntry{
			Name:       name,
			Type:       hostType,
			Properties: props,
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// IsWritable returns true - INI files support write operations
func (s *INISource) IsWritable() bool {
	return true
}

// DefaultPaths returns the default INI config paths
func (s *INISource) DefaultPaths() []string {
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return []string{"hosts.ini"}
		}
		configDir = filepath.Join(home, ".config")
	}
	return []string{filepath.Join(configDir, "hop", "hosts.ini")}
}

// resolveInheritance resolves the extends chain for a host
func resolveInheritance(name string, rawHosts map[string]map[string]string, visited map[string]bool) (map[string]string, error) {
	if visited[name] {
		return nil, &CircularInheritanceError{Host: name}
	}
	visited[name] = true

	props, ok := rawHosts[name]
	if !ok {
		return nil, &HostNotFoundError{Host: name}
	}

	// Check if this host extends another
	extends := props["extends"]
	if extends == "" {
		// No inheritance, just expand env vars and return
		result := make(map[string]string)
		for k, v := range props {
			result[k] = os.ExpandEnv(v)
		}
		return result, nil
	}

	// Resolve parent first
	parent, err := resolveInheritance(extends, rawHosts, visited)
	if err != nil {
		return nil, err
	}

	// Merge: parent properties first, then child overrides
	result := make(map[string]string)
	for k, v := range parent {
		result[k] = v
	}
	for k, v := range props {
		if k != "extends" { // Don't copy extends property
			result[k] = os.ExpandEnv(v)
		}
	}

	return result, nil
}

// CircularInheritanceError indicates a circular extends chain
type CircularInheritanceError struct {
	Host string
}

func (e *CircularInheritanceError) Error() string {
	return "circular inheritance detected involving " + e.Host
}

// HostNotFoundError indicates a referenced host was not found
type HostNotFoundError struct {
	Host string
}

func (e *HostNotFoundError) Error() string {
	return "host not found: " + e.Host
}
