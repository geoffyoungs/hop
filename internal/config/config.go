package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/geoff/hop/internal/config/source"
	"gopkg.in/ini.v1"
)

// Config holds all loaded host configurations
type Config struct {
	Hosts    map[string]*HostConfig
	Settings *Settings // Settings used to load this config (nil if loaded via LoadFromPath)
}

// DefaultConfigPath returns the default config file path following XDG conventions
func DefaultConfigPath() string {
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "hosts.ini"
		}
		configDir = filepath.Join(home, ".config")
	}
	return filepath.Join(configDir, "hop", "hosts.ini")
}

// Load reads the configuration from the default path
func Load() (*Config, error) {
	return LoadFromPath(DefaultConfigPath())
}

// LoadFromPath reads the configuration from a specific path
func LoadFromPath(path string) (*Config, error) {
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
			return nil, fmt.Errorf("error resolving host %q: %w", name, err)
		}
		resolvedHosts[name] = resolved
	}

	// Pass 3: Convert to HostConfig structs
	config := &Config{
		Hosts: make(map[string]*HostConfig),
	}

	for name, props := range resolvedHosts {
		host := hostConfigFromMap(name, props)
		config.Hosts[name] = host
	}

	return config, nil
}

// resolveInheritance resolves the extends chain for a host
func resolveInheritance(name string, rawHosts map[string]map[string]string, visited map[string]bool) (map[string]string, error) {
	if visited[name] {
		return nil, fmt.Errorf("circular inheritance detected involving %q", name)
	}
	visited[name] = true

	props, ok := rawHosts[name]
	if !ok {
		return nil, fmt.Errorf("host %q not found", name)
	}

	// Check if this host extends another
	extends := props["extends"]
	if extends == "" {
		// No inheritance, just expand env vars and return
		result := make(map[string]string)
		for k, v := range props {
			result[k] = ExpandEnv(v)
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
			result[k] = ExpandEnv(v)
		}
	}

	return result, nil
}

// hostConfigFromMap creates a HostConfig from a properties map
func hostConfigFromMap(name string, props map[string]string) *HostConfig {
	host := &HostConfig{
		Name:      name,
		Type:      getStringOrDefault(props, "type", "ssh"),
		Default:   parseAgentForward(props["default"]), // reuses bool parsing
		// SSH fields
		Host:         props["host"],
		User:         props["user"],
		Port:         parsePort(props["port"]),
		Identity:     props["identity"],
		Jump:         props["jump"],
		AgentForward: parseAgentForward(props["agent_forward"]),
		// Docker fields
		Container: props["container"],
		Shell:     getStringOrDefault(props, "shell", "/bin/sh"),
		Label:     props["label"],
		Image:     props["image"],
		ImageGrep: props["image_grep"],
		// K8s fields
		Namespace:  getStringOrDefault(props, "namespace", "default"),
		Pod:        props["pod"],
		Context:    props["context"],
		Selector:   props["selector"],
		PodGrep:    props["pod_grep"],
		Deployment: props["deployment"],
		// Inheritance (not stored, already resolved)
		Extends: "",
		// Port forwarding
		LocalPort:  parsePort(props["local_port"]),
		RemotePort: parsePort(props["remote_port"]),
	}
	return host
}

func getStringOrDefault(props map[string]string, key, defaultVal string) string {
	if v, ok := props[key]; ok && v != "" {
		return v
	}
	return defaultVal
}

func parsePort(s string) int {
	if s == "" {
		return 0
	}
	p, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return p
}

// Get retrieves a host configuration by name
func (c *Config) Get(name string) (*HostConfig, bool) {
	host, ok := c.Hosts[name]
	return host, ok
}

// GetByPrefix retrieves a host by exact name or unique prefix.
// Returns the host and nil error if found uniquely.
// Returns nil and an error if ambiguous (multiple matches) or not found.
func (c *Config) GetByPrefix(prefix string) (*HostConfig, error) {
	// Exact match takes priority
	if host, ok := c.Hosts[prefix]; ok {
		return host, nil
	}

	// Find all hosts that start with prefix
	var matches []*HostConfig
	for name, host := range c.Hosts {
		if strings.HasPrefix(name, prefix) {
			matches = append(matches, host)
		}
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("host %q not found", prefix)
	}

	if len(matches) == 1 {
		return matches[0], nil
	}

	// Ambiguous - multiple matches
	names := make([]string, len(matches))
	for i, m := range matches {
		names[i] = m.Name
	}
	sort.Strings(names)
	return nil, fmt.Errorf("ambiguous host %q matches: %s", prefix, strings.Join(names, ", "))
}

// Names returns all configured host names
func (c *Config) Names() []string {
	names := make([]string, 0, len(c.Hosts))
	for name := range c.Hosts {
		names = append(names, name)
	}
	return names
}

// DefaultHost returns the default host if one is configured, or the only host if there's just one.
// Returns the host config and true if found, nil and false otherwise.
func (c *Config) DefaultHost() (*HostConfig, bool) {
	names := c.Names()

	// If only one host, return it
	if len(names) == 1 {
		return c.Hosts[names[0]], true
	}

	// Look for explicitly marked default
	for _, host := range c.Hosts {
		if host.Default {
			return host, true
		}
	}

	return nil, false
}

// parseAgentForward parses agent_forward value (yes/true/1 -> true)
func parseAgentForward(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "yes" || s == "true" || s == "1"
}

// LoadAll loads hosts from all enabled sources based on settings
func LoadAll() (*Config, error) {
	return LoadAllWithSettings(nil)
}

// LoadAllWithSettings loads hosts from all enabled sources using provided settings
func LoadAllWithSettings(settings *Settings) (*Config, error) {
	if settings == nil {
		var err error
		settings, err = LoadSettings()
		if err != nil {
			return nil, fmt.Errorf("failed to load settings: %w", err)
		}
	}

	config := &Config{
		Hosts:    make(map[string]*HostConfig),
		Settings: settings,
	}

	// Track hosts by name for collision detection
	hostSources := make(map[string]string) // hostname -> source name

	// Load sources in priority order
	for _, sourceName := range settings.GetPriority() {
		if !settings.IsSourceEnabled(sourceName) {
			continue
		}

		src, err := source.Get(sourceName)
		if err != nil {
			continue // Source not registered
		}

		// Get paths to load from
		paths := settings.GetSourcePaths(sourceName)
		if paths == nil {
			paths = src.DefaultPaths()
		}

		// Load from each path
		for _, path := range paths {
			// Expand ~ in path
			path = expandPath(path)

			// Skip if file doesn't exist
			if _, err := os.Stat(path); os.IsNotExist(err) {
				continue
			}

			entries, err := src.Load(path)
			if err != nil {
				// Log but continue - don't fail entire load for one bad source
				continue
			}

			for _, entry := range entries {
				host := hostConfigFromEntry(entry, sourceName, path, !src.IsWritable())

				// Handle collisions based on strategy
				if existingSource, exists := hostSources[entry.Name]; exists {
					switch settings.GetCollisionStrategy() {
					case CollisionFirst:
						// Skip - first source wins
						continue
					case CollisionQualify:
						// Rename both with source prefix
						qualifiedName := sourceName + ":" + entry.Name
						host.Name = qualifiedName
						// Also rename the existing one if not already qualified
						if existing, ok := config.Hosts[entry.Name]; ok {
							delete(config.Hosts, entry.Name)
							existing.Name = existingSource + ":" + entry.Name
							config.Hosts[existing.Name] = existing
						}
					case CollisionError:
						return nil, fmt.Errorf("duplicate host %q found in sources %q and %q",
							entry.Name, existingSource, sourceName)
					}
				}

				config.Hosts[host.Name] = host
				hostSources[entry.Name] = sourceName
			}
		}
	}

	return config, nil
}

// LoadFromSources loads hosts from specific sources only
func LoadFromSources(sourceNames []string) (*Config, error) {
	settings := DefaultSettings()

	// Disable all sources
	f := false
	settings.Sources.INI = &f
	settings.Sources.Ansible = &f
	settings.Sources.SSHConfig = &f
	settings.Sources.Vagrant = &f

	// Enable only requested sources
	t := true
	for _, name := range sourceNames {
		switch name {
		case "ini":
			settings.Sources.INI = &t
		case "ansible":
			settings.Sources.Ansible = &t
		case "sshconfig":
			settings.Sources.SSHConfig = &t
		case "vagrant":
			settings.Sources.Vagrant = &t
		}
	}

	// Use the requested sources as priority
	settings.Sources.Priority = sourceNames

	return LoadAllWithSettings(settings)
}

// hostConfigFromEntry converts a source.HostEntry to a HostConfig
func hostConfigFromEntry(entry *source.HostEntry, sourceName, sourcePath string, readOnly bool) *HostConfig {
	host := hostConfigFromMap(entry.Name, entry.Properties)
	host.SourceName = sourceName
	host.SourcePath = sourcePath
	host.SourceReadOnly = readOnly
	return host
}

// expandPath expands ~ and environment variables in a path
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			path = filepath.Join(home, path[2:])
		}
	}
	return os.ExpandEnv(path)
}

// GetByQualifiedName retrieves a host by source-qualified name (e.g., "ini:myhost")
func (c *Config) GetByQualifiedName(name string) (*HostConfig, error) {
	// Check for source:name format
	if idx := strings.Index(name, ":"); idx > 0 {
		sourceName := name[:idx]
		hostName := name[idx+1:]

		// Look for exact match with qualified name
		if host, ok := c.Hosts[name]; ok {
			return host, nil
		}

		// Look for host from specific source
		for _, host := range c.Hosts {
			if host.SourceName == sourceName && host.Name == hostName {
				return host, nil
			}
		}

		return nil, fmt.Errorf("host %q not found in source %q", hostName, sourceName)
	}

	// Fall back to regular lookup
	return c.GetByPrefix(name)
}
