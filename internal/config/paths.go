package config

import (
	"os"
	"path/filepath"
)

// PathMode determines which config file to use
type PathMode int

const (
	// ModeDefault uses the XDG config path (~/.config/hop/hosts.ini)
	ModeDefault PathMode = iota
	// ModeLocal uses a config file in the current directory (./hosts.ini)
	ModeLocal
	// ModeUser uses the XDG config path explicitly (~/.config/hop/hosts.ini)
	ModeUser
	// ModeExplicit uses a user-specified path
	ModeExplicit
)

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
	default:
		return DefaultConfigPath()
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
