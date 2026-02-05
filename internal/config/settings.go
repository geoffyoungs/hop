package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// CollisionStrategy determines how to handle duplicate host names across sources
type CollisionStrategy string

const (
	// CollisionFirst uses the first source in priority order (default)
	CollisionFirst CollisionStrategy = "first"
	// CollisionQualify keeps both hosts as source:hostname
	CollisionQualify CollisionStrategy = "qualify"
	// CollisionError fails when duplicates are found
	CollisionError CollisionStrategy = "error"
)

// SourceSettings contains settings for a specific source
type SourceSettings struct {
	Enabled bool     `toml:"enabled"`
	Paths   []string `toml:"paths"`
}

// Settings holds all hop configuration settings
type Settings struct {
	Sources    SourcesSettings    `toml:"sources"`
	Collisions CollisionsSettings `toml:"collisions"`
}

// SourcesSettings contains per-source enable/disable and path overrides
type SourcesSettings struct {
	INI       *bool    `toml:"ini"`
	Ansible   *bool    `toml:"ansible"`
	SSHConfig *bool    `toml:"sshconfig"`
	Vagrant   *bool    `toml:"vagrant"`
	Priority  []string `toml:"priority"`

	// Custom paths per source
	Paths SourcePathsSettings `toml:"paths"`
}

// SourcePathsSettings holds custom search paths for each source
type SourcePathsSettings struct {
	Ansible   []string `toml:"ansible"`
	SSHConfig []string `toml:"sshconfig"`
}

// CollisionsSettings controls collision handling
type CollisionsSettings struct {
	Strategy CollisionStrategy `toml:"strategy"`
}

// DefaultSettings returns settings with sensible defaults
func DefaultSettings() *Settings {
	t := true
	f := false
	return &Settings{
		Sources: SourcesSettings{
			INI:       &t,
			Ansible:   &t,
			SSHConfig: &f, // Opt-in (can be noisy)
			Vagrant:   &t,
			Priority:  []string{"ini", "ansible", "vagrant", "sshconfig"},
			Paths: SourcePathsSettings{
				Ansible:   nil, // Use source defaults
				SSHConfig: nil, // Use source defaults
			},
		},
		Collisions: CollisionsSettings{
			Strategy: CollisionFirst,
		},
	}
}

// SettingsPath returns the path to the settings file
func SettingsPath() string {
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "settings.toml"
		}
		configDir = filepath.Join(home, ".config")
	}
	return filepath.Join(configDir, "hop", "settings.toml")
}

// LoadSettings loads settings from the default path
func LoadSettings() (*Settings, error) {
	return LoadSettingsFromPath(SettingsPath())
}

// LoadSettingsFromPath loads settings from a specific path
func LoadSettingsFromPath(path string) (*Settings, error) {
	// If file doesn't exist, return defaults
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return DefaultSettings(), nil
	}

	// Decode into a fresh struct to avoid issues with pointer fields
	var loaded Settings
	if _, err := toml.DecodeFile(path, &loaded); err != nil {
		return nil, err
	}

	// Apply defaults for any nil/empty values
	defaults := DefaultSettings()
	if loaded.Sources.INI == nil {
		loaded.Sources.INI = defaults.Sources.INI
	}
	if loaded.Sources.Ansible == nil {
		loaded.Sources.Ansible = defaults.Sources.Ansible
	}
	if loaded.Sources.SSHConfig == nil {
		loaded.Sources.SSHConfig = defaults.Sources.SSHConfig
	}
	if loaded.Sources.Vagrant == nil {
		loaded.Sources.Vagrant = defaults.Sources.Vagrant
	}
	if len(loaded.Sources.Priority) == 0 {
		loaded.Sources.Priority = defaults.Sources.Priority
	}
	if loaded.Collisions.Strategy == "" {
		loaded.Collisions.Strategy = defaults.Collisions.Strategy
	}

	return &loaded, nil
}

// IsSourceEnabled returns whether a source is enabled in settings
func (s *Settings) IsSourceEnabled(name string) bool {
	switch name {
	case "ini":
		return s.Sources.INI != nil && *s.Sources.INI
	case "ansible":
		return s.Sources.Ansible != nil && *s.Sources.Ansible
	case "sshconfig":
		return s.Sources.SSHConfig != nil && *s.Sources.SSHConfig
	case "vagrant":
		return s.Sources.Vagrant != nil && *s.Sources.Vagrant
	default:
		return false
	}
}

// GetSourcePaths returns custom paths for a source, or nil to use defaults
func (s *Settings) GetSourcePaths(name string) []string {
	switch name {
	case "ansible":
		return s.Sources.Paths.Ansible
	case "sshconfig":
		return s.Sources.Paths.SSHConfig
	default:
		return nil
	}
}

// GetPriority returns the source priority order
func (s *Settings) GetPriority() []string {
	return s.Sources.Priority
}

// GetCollisionStrategy returns the collision handling strategy
func (s *Settings) GetCollisionStrategy() CollisionStrategy {
	return s.Collisions.Strategy
}
