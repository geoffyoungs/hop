package source

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// AnsibleSource loads hosts from Ansible inventory files
type AnsibleSource struct{}

func init() {
	Register(&AnsibleSource{})
}

// Name returns "ansible"
func (s *AnsibleSource) Name() string {
	return "ansible"
}

// FilePatterns returns patterns for Ansible inventory files
func (s *AnsibleSource) FilePatterns() []string {
	return []string{"inventory.yml", "inventory.yaml", "hosts.yml", "hosts.yaml"}
}

// CanLoad returns true for Ansible inventory YAML files
func (s *AnsibleSource) CanLoad(path string) bool {
	base := filepath.Base(path)
	ext := strings.ToLower(filepath.Ext(path))

	// Check for standard Ansible inventory file names
	if base == "inventory.yml" || base == "inventory.yaml" ||
		base == "hosts.yml" || base == "hosts.yaml" {
		return true
	}

	// Check if it's a YAML file in an ansible or inventory directory
	if ext == ".yml" || ext == ".yaml" {
		dir := filepath.Base(filepath.Dir(path))
		if dir == "ansible" || dir == "inventory" || dir == "inventories" {
			return true
		}
	}

	return false
}

// Load reads hosts from an Ansible inventory YAML file
func (s *AnsibleSource) Load(path string) ([]*HostEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var inventory map[string]interface{}
	if err := yaml.Unmarshal(data, &inventory); err != nil {
		return nil, err
	}

	entries := make([]*HostEntry, 0)

	// Parse the inventory structure
	// Ansible inventory can have:
	// 1. "all" group with hosts and children
	// 2. Direct group definitions with hosts
	for groupName, groupValue := range inventory {
		groupEntries := s.parseGroup(groupName, groupValue)
		entries = append(entries, groupEntries...)
	}

	return entries, nil
}

// parseGroup parses an Ansible group and returns host entries
func (s *AnsibleSource) parseGroup(groupName string, groupValue interface{}) []*HostEntry {
	entries := make([]*HostEntry, 0)

	groupMap, ok := groupValue.(map[string]interface{})
	if !ok {
		return entries
	}

	// Parse hosts in this group
	if hosts, ok := groupMap["hosts"].(map[string]interface{}); ok {
		for hostName, hostVars := range hosts {
			entry := s.parseHost(hostName, hostVars, groupName)
			if entry != nil {
				entries = append(entries, entry)
			}
		}
	}

	// Parse children groups recursively
	if children, ok := groupMap["children"].(map[string]interface{}); ok {
		for childName, childValue := range children {
			childEntries := s.parseGroup(childName, childValue)
			entries = append(entries, childEntries...)
		}
	}

	return entries
}

// parseHost parses a single Ansible host definition
func (s *AnsibleSource) parseHost(hostName string, hostVars interface{}, groupName string) *HostEntry {
	props := make(map[string]string)

	// Map Ansible variables to hop properties
	if vars, ok := hostVars.(map[string]interface{}); ok {
		for k, v := range vars {
			switch k {
			case "ansible_host":
				props["host"] = toString(v)
			case "ansible_user":
				props["user"] = toString(v)
			case "ansible_port":
				props["port"] = toString(v)
			case "ansible_ssh_private_key_file":
				props["identity"] = toString(v)
			case "ansible_ssh_common_args":
				// Parse for ProxyJump if present
				args := toString(v)
				if jump := extractProxyJump(args); jump != "" {
					props["jump"] = jump
				}
			default:
				// Store other ansible_ prefixed vars in properties for reference
				if strings.HasPrefix(k, "ansible_") {
					props[k] = toString(v)
				}
			}
		}
	}

	// If no ansible_host specified, use the inventory hostname as the host
	if props["host"] == "" {
		// Check if the hostname looks like an IP or FQDN
		if isHostnameOrIP(hostName) {
			props["host"] = hostName
		}
	}

	// Skip hosts without a valid host property
	if props["host"] == "" {
		return nil
	}

	// Store the group as a tag in properties
	if groupName != "" && groupName != "all" {
		props["ansible_group"] = groupName
	}

	return &HostEntry{
		Name:       hostName,
		Type:       "ssh",
		Properties: props,
	}
}

// IsWritable returns false - Ansible inventories are read-only
func (s *AnsibleSource) IsWritable() bool {
	return false
}

// DefaultPaths returns the default Ansible inventory paths
func (s *AnsibleSource) DefaultPaths() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	return []string{
		filepath.Join(home, ".ansible", "inventory.yml"),
		filepath.Join(home, ".ansible", "hosts.yml"),
		"/etc/ansible/hosts",
	}
}

// toString converts an interface{} to string
func toString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case int:
		return strconv.Itoa(val)
	case int64:
		return strconv.FormatInt(val, 10)
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(val)
	default:
		return ""
	}
}

// extractProxyJump extracts the ProxyJump host from SSH args
func extractProxyJump(args string) string {
	// Look for -J or -o ProxyJump= patterns
	parts := strings.Fields(args)
	for i, part := range parts {
		if part == "-J" && i+1 < len(parts) {
			return parts[i+1]
		}
		if strings.HasPrefix(part, "-J") {
			return strings.TrimPrefix(part, "-J")
		}
		if strings.HasPrefix(part, "ProxyJump=") {
			return strings.TrimPrefix(part, "ProxyJump=")
		}
	}
	return ""
}

// isHostnameOrIP checks if a string looks like a hostname or IP address
func isHostnameOrIP(s string) bool {
	// Simple heuristic: contains a dot or is an IP-like pattern
	return strings.Contains(s, ".") || strings.Contains(s, ":")
}
