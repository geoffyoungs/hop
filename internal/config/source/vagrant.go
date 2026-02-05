package source

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// VagrantSource discovers hosts from Vagrant VMs
type VagrantSource struct{}

func init() {
	Register(&VagrantSource{})
}

// Name returns "vagrant"
func (s *VagrantSource) Name() string {
	return "vagrant"
}

// FilePatterns returns patterns for Vagrantfile detection
func (s *VagrantSource) FilePatterns() []string {
	return []string{"Vagrantfile"}
}

// CanLoad returns true if the path is or contains a Vagrantfile
func (s *VagrantSource) CanLoad(path string) bool {
	base := filepath.Base(path)

	// Direct Vagrantfile
	if base == "Vagrantfile" {
		return true
	}

	// Directory containing a Vagrantfile
	info, err := os.Stat(path)
	if err == nil && info.IsDir() {
		vagrantfile := filepath.Join(path, "Vagrantfile")
		if _, err := os.Stat(vagrantfile); err == nil {
			return true
		}
	}

	return false
}

// Load reads hosts from Vagrant VMs by running vagrant ssh-config
func (s *VagrantSource) Load(path string) ([]*HostEntry, error) {
	// Determine the working directory
	var workDir string
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if info.IsDir() {
		workDir = path
	} else {
		workDir = filepath.Dir(path)
	}

	// Check if vagrant is available
	if _, err := exec.LookPath("vagrant"); err != nil {
		return nil, nil // Vagrant not installed, return empty
	}

	// Run vagrant ssh-config
	cmd := exec.Command("vagrant", "ssh-config")
	cmd.Dir = workDir

	output, err := cmd.Output()
	if err != nil {
		// If vagrant fails (no VMs, not a vagrant project, etc.), return empty
		return nil, nil
	}

	return parseVagrantSSHConfig(string(output), workDir)
}

// IsWritable returns false - Vagrant hosts are discovered dynamically
func (s *VagrantSource) IsWritable() bool {
	return false
}

// DefaultPaths returns empty - Vagrant sources are discovered from working directory
func (s *VagrantSource) DefaultPaths() []string {
	return nil
}

// parseVagrantSSHConfig parses the output of vagrant ssh-config
func parseVagrantSSHConfig(output string, workDir string) ([]*HostEntry, error) {
	entries := make([]*HostEntry, 0)
	var currentHost *vagrantHost

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Check indentation - Host lines are not indented, properties are
		if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			// Save previous host if exists
			if currentHost != nil {
				entry := currentHost.toHostEntry(workDir)
				if entry != nil {
					entries = append(entries, entry)
				}
			}

			// Parse Host line
			fields := strings.Fields(line)
			if len(fields) >= 2 && strings.ToLower(fields[0]) == "host" {
				currentHost = &vagrantHost{
					name:  fields[1],
					props: make(map[string]string),
				}
			}
			continue
		}

		// Parse property line
		if currentHost != nil {
			key, value := parseVagrantConfigLine(line)
			if key != "" {
				currentHost.props[strings.ToLower(key)] = value
			}
		}
	}

	// Don't forget the last host
	if currentHost != nil {
		entry := currentHost.toHostEntry(workDir)
		if entry != nil {
			entries = append(entries, entry)
		}
	}

	return entries, scanner.Err()
}

// vagrantHost represents a parsed VM from vagrant ssh-config
type vagrantHost struct {
	name  string
	props map[string]string
}

// toHostEntry converts a Vagrant VM to a HostEntry
func (h *vagrantHost) toHostEntry(workDir string) *HostEntry {
	if h.name == "" {
		return nil
	}

	props := make(map[string]string)

	// Map vagrant ssh-config options to hop properties
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
		props["identity"] = v
	}
	if v, ok := h.props["userknownhostsfile"]; ok {
		props["vagrant_known_hosts"] = v
	}
	if v, ok := h.props["stricthostkeychecking"]; ok {
		props["vagrant_strict_host_key"] = v
	}

	// Require a host to be valid
	if props["host"] == "" {
		return nil
	}

	// Add a prefix to distinguish vagrant hosts
	name := "vagrant-" + h.name

	// Store the source directory for reference
	props["vagrant_dir"] = workDir

	return &HostEntry{
		Name:       name,
		Type:       "ssh",
		Properties: props,
	}
}

// parseVagrantConfigLine parses an indented line from vagrant ssh-config
func parseVagrantConfigLine(line string) (string, string) {
	line = strings.TrimSpace(line)
	fields := strings.Fields(line)
	if len(fields) >= 2 {
		return fields[0], strings.Join(fields[1:], " ")
	}
	if len(fields) == 1 {
		return fields[0], ""
	}
	return "", ""
}

// FindVagrantDirs searches for directories containing Vagrantfile, starting from current dir
func FindVagrantDirs() []string {
	var dirs []string

	cwd, err := os.Getwd()
	if err != nil {
		return dirs
	}

	// Check current directory
	if _, err := os.Stat(filepath.Join(cwd, "Vagrantfile")); err == nil {
		dirs = append(dirs, cwd)
	}

	// Walk up to find parent vagrantfiles
	dir := cwd
	for {
		parent := filepath.Dir(dir)
		if parent == dir {
			break // Reached root
		}
		dir = parent

		if _, err := os.Stat(filepath.Join(dir, "Vagrantfile")); err == nil {
			dirs = append(dirs, dir)
		}

		// Stop at git root
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			break
		}
	}

	return dirs
}
