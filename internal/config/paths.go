package config

import (
	"os"
	"path/filepath"
)

// PathMode determines which config file to use
type PathMode int

const (
	// ModeDefault searches project config first, then user config
	ModeDefault PathMode = iota
	// ModeLocal uses a config file in the current directory only (./hosts.ini)
	ModeLocal
	// ModeUser uses the XDG config path explicitly (~/.config/hop/hosts.ini)
	ModeUser
	// ModeExplicit uses a user-specified path
	ModeExplicit
	// ModeProject walks up directories to .git looking for config
	ModeProject
)

// ConfigFileNames returns the valid config file names in order of preference
func ConfigFileNames() []string {
	return []string{"hosts.ini", "hosts.conf"}
}

// LocalConfigPath returns the local config file path (./hosts.ini)
func LocalConfigPath() string {
	return "hosts.ini"
}

// UserConfigPath returns the user config file path (~/.config/hop/hosts.ini)
// This is the same as DefaultConfigPath but named for clarity with --user flag
func UserConfigPath() string {
	return DefaultConfigPath()
}

// ResolvePath determines the config file path based on mode and explicit path
func ResolvePath(mode PathMode, explicit string) string {
	switch mode {
	case ModeExplicit:
		if explicit != "" {
			return explicit
		}
		return DefaultConfigPath()
	case ModeLocal:
		return LocalConfigPath()
	case ModeUser:
		return UserConfigPath()
	case ModeProject:
		if path, found := FindProjectConfig(); found {
			return path
		}
		return "" // No project config found
	default:
		// ModeDefault: search project first, then user
		return FindConfigFile()
	}
}

// EnsureConfigDir creates the config directory if it doesn't exist
func EnsureConfigDir(path string) error {
	dir := filepath.Dir(path)
	return os.MkdirAll(dir, 0755)
}

// ConfigExists checks if a config file exists at the given path
func ConfigExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// FindProjectConfig walks up directories looking for a config file, stopping at .git
func FindProjectConfig() (string, bool) {
	dir, err := os.Getwd()
	if err != nil {
		return "", false
	}

	for {
		// Check for .git (stop here, this is project root)
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			// Check for config in this dir before stopping
			for _, name := range ConfigFileNames() {
				path := filepath.Join(dir, name)
				if ConfigExists(path) {
					return path, true
				}
			}
			return "", false // Hit .git but no config found
		}

		// Check for config file
		for _, name := range ConfigFileNames() {
			path := filepath.Join(dir, name)
			if ConfigExists(path) {
				return path, true
			}
		}

		// Move to parent
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false // Reached filesystem root
		}
		dir = parent
	}
}

// FindUserConfig finds the user config file if it exists
func FindUserConfig() (string, bool) {
	for _, name := range ConfigFileNames() {
		configDir := os.Getenv("XDG_CONFIG_HOME")
		if configDir == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				continue
			}
			configDir = filepath.Join(home, ".config")
		}
		path := filepath.Join(configDir, "hop", name)
		if ConfigExists(path) {
			return path, true
		}
	}
	return "", false
}

// FindConfigFile searches for config in order: project -> user
// Returns the default user config path if nothing found
func FindConfigFile() string {
	// Try project config first
	if path, found := FindProjectConfig(); found {
		return path
	}

	// Try user config
	if path, found := FindUserConfig(); found {
		return path
	}

	// Fall back to default user config path
	return DefaultConfigPath()
}
