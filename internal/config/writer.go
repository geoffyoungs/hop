package config

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"gopkg.in/ini.v1"
)

// ValidSSHProperties lists valid properties for SSH backend
var ValidSSHProperties = []string{"type", "host", "user", "port", "identity", "jump", "agent_forward"}

// ValidDockerProperties lists valid properties for Docker backend
var ValidDockerProperties = []string{"type", "container", "shell", "label", "image", "image_grep"}

// ValidK8sProperties lists valid properties for Kubernetes backend
var ValidK8sProperties = []string{"type", "namespace", "pod", "container", "context", "shell", "selector", "pod_grep", "deployment"}

// AllValidProperties returns all valid property names across all backends
var AllValidProperties = []string{
	"type", "host", "user", "port", "identity", "jump", "agent_forward",
	"container", "shell", "namespace", "pod", "context",
	"local_port", "remote_port", "selector", "pod_grep",
	"label", "image", "image_grep", "deployment", "default", "extends",
}

// AddHost adds a new host to the config file
// If the file doesn't exist, it will be created
// If the host already exists, an error is returned
func AddHost(path, name string, props map[string]string) error {
	if name == "" {
		return fmt.Errorf("host name cannot be empty")
	}

	if err := validateHostName(name); err != nil {
		return err
	}

	if err := validateHostProperties(props); err != nil {
		return err
	}

	// Ensure the config directory exists
	if err := EnsureConfigDir(path); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Load existing config or create new one
	var cfg *ini.File
	var err error

	if ConfigExists(path) {
		cfg, err = ini.Load(path)
		if err != nil {
			return fmt.Errorf("failed to load config file: %w", err)
		}

		// Check if host already exists
		if cfg.HasSection(name) {
			return fmt.Errorf("host %q already exists in config", name)
		}
	} else {
		cfg = ini.Empty()
	}

	// Create new section for the host
	section, err := cfg.NewSection(name)
	if err != nil {
		return fmt.Errorf("failed to create section: %w", err)
	}

	// Add properties in sorted order for consistent output
	keys := make([]string, 0, len(props))
	for k := range props {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		_, err := section.NewKey(key, props[key])
		if err != nil {
			return fmt.Errorf("failed to set property %q: %w", key, err)
		}
	}

	// Save the file
	if err := cfg.SaveTo(path); err != nil {
		return fmt.Errorf("failed to save config file: %w", err)
	}

	return nil
}

// UpdateHost updates an existing host in the config file
func UpdateHost(path, name string, props map[string]string) error {
	if name == "" {
		return fmt.Errorf("host name cannot be empty")
	}

	if err := validateHostProperties(props); err != nil {
		return err
	}

	if !ConfigExists(path) {
		return fmt.Errorf("config file does not exist: %s", path)
	}

	cfg, err := ini.Load(path)
	if err != nil {
		return fmt.Errorf("failed to load config file: %w", err)
	}

	if !cfg.HasSection(name) {
		return fmt.Errorf("host %q not found in config", name)
	}

	section := cfg.Section(name)

	// Update properties
	for key, value := range props {
		section.Key(key).SetValue(value)
	}

	if err := cfg.SaveTo(path); err != nil {
		return fmt.Errorf("failed to save config file: %w", err)
	}

	return nil
}

// RemoveHost removes a host from the config file
func RemoveHost(path, name string) error {
	if name == "" {
		return fmt.Errorf("host name cannot be empty")
	}

	if !ConfigExists(path) {
		return fmt.Errorf("config file does not exist: %s", path)
	}

	cfg, err := ini.Load(path)
	if err != nil {
		return fmt.Errorf("failed to load config file: %w", err)
	}

	if !cfg.HasSection(name) {
		return fmt.Errorf("host %q not found in config", name)
	}

	cfg.DeleteSection(name)

	if err := cfg.SaveTo(path); err != nil {
		return fmt.Errorf("failed to save config file: %w", err)
	}

	return nil
}

// validateHostName checks if the host name is valid for INI sections
func validateHostName(name string) error {
	if name == "DEFAULT" {
		return fmt.Errorf("host name cannot be 'DEFAULT' (reserved)")
	}

	if strings.ContainsAny(name, "[]") {
		return fmt.Errorf("host name cannot contain '[' or ']'")
	}

	if strings.TrimSpace(name) != name {
		return fmt.Errorf("host name cannot have leading or trailing whitespace")
	}

	return nil
}

// validateHostProperties validates the properties for a host
func validateHostProperties(props map[string]string) error {
	// Check for unknown properties
	validProps := make(map[string]bool)
	for _, p := range AllValidProperties {
		validProps[p] = true
	}

	for key := range props {
		if !validProps[key] {
			return fmt.Errorf("unknown property %q", key)
		}
	}

	// Determine backend type
	backendType := props["type"]
	if backendType == "" {
		backendType = "ssh"
	}

	// Validate required fields based on backend type
	switch backendType {
	case "ssh":
		if props["host"] == "" {
			return fmt.Errorf("'host' is required for SSH backend")
		}
	case "docker":
		if props["container"] == "" && props["label"] == "" && props["image"] == "" && props["image_grep"] == "" {
			return fmt.Errorf("one of 'container', 'label', 'image', or 'image_grep' is required for Docker backend")
		}
	case "k8s":
		if props["pod"] == "" && props["selector"] == "" && props["pod_grep"] == "" && props["deployment"] == "" {
			return fmt.Errorf("one of 'pod', 'selector', 'pod_grep', or 'deployment' is required for Kubernetes backend")
		}
	default:
		return fmt.Errorf("unknown backend type %q (must be ssh, docker, or k8s)", backendType)
	}

	return nil
}

// ParseKeyValue parses a "key=value" string
func ParseKeyValue(s string) (key, value string, err error) {
	parts := strings.SplitN(s, "=", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid format %q (expected key=value)", s)
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), nil
}

// ParseKeyValuePairs parses multiple "key=value" strings into a map
func ParseKeyValuePairs(args []string) (map[string]string, error) {
	props := make(map[string]string)
	for _, arg := range args {
		key, value, err := ParseKeyValue(arg)
		if err != nil {
			return nil, err
		}
		props[key] = value
	}
	return props, nil
}

// FormatHostEntry formats a host entry for display
func FormatHostEntry(name string, props map[string]string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("[%s]\n", name))

	keys := make([]string, 0, len(props))
	for k := range props {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		sb.WriteString(fmt.Sprintf("%s = %s\n", key, props[key]))
	}

	return sb.String()
}

// HostExists checks if a host exists in the config file
func HostExists(path, name string) (bool, error) {
	if !ConfigExists(path) {
		return false, nil
	}

	cfg, err := ini.Load(path)
	if err != nil {
		return false, fmt.Errorf("failed to load config file: %w", err)
	}

	return cfg.HasSection(name), nil
}

// CreateEmptyConfig creates an empty config file
func CreateEmptyConfig(path string) error {
	if err := EnsureConfigDir(path); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	return file.Close()
}
