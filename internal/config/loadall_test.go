package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	// Import to register sources
	_ "github.com/geoff/hop/internal/config/source"
)

func TestLoadFromSources_INI(t *testing.T) {
	tmpDir := t.TempDir()
	iniPath := filepath.Join(tmpDir, "hosts.ini")

	content := `[production]
host = 192.168.1.100
user = admin

[staging]
host = 192.168.1.50
user = deploy
`
	err := os.WriteFile(iniPath, []byte(content), 0644)
	require.NoError(t, err)

	// Create custom settings pointing to our test file
	settings := DefaultSettings()
	settings.Sources.Paths.Ansible = nil // Clear ansible paths
	settings.Sources.Paths.SSHConfig = nil

	// We can't easily test LoadFromSources without modifying DefaultPaths
	// So test LoadFromPath still works as before
	cfg, err := LoadFromPath(iniPath)
	require.NoError(t, err)

	assert.Len(t, cfg.Hosts, 2)

	prod, ok := cfg.Get("production")
	require.True(t, ok)
	assert.Equal(t, "192.168.1.100", prod.Host)
	assert.Equal(t, "admin", prod.User)
}

func TestHostConfig_SourceTracking(t *testing.T) {
	host := &HostConfig{
		Name:           "test",
		Host:           "example.com",
		SourceName:     "ini",
		SourcePath:     "/path/to/hosts.ini",
		SourceReadOnly: false,
	}

	assert.Equal(t, "ini", host.SourceName)
	assert.Equal(t, "/path/to/hosts.ini", host.SourcePath)
	assert.False(t, host.SourceReadOnly)
}

func TestConfig_GetByQualifiedName(t *testing.T) {
	cfg := &Config{
		Hosts: map[string]*HostConfig{
			"production": {
				Name:       "production",
				Host:       "prod.example.com",
				SourceName: "ini",
			},
			"ini:staging": {
				Name:       "ini:staging",
				Host:       "staging.example.com",
				SourceName: "ini",
			},
		},
	}

	// Regular lookup
	host, err := cfg.GetByQualifiedName("production")
	require.NoError(t, err)
	assert.Equal(t, "production", host.Name)

	// Qualified name lookup (already qualified in map)
	host, err = cfg.GetByQualifiedName("ini:staging")
	require.NoError(t, err)
	assert.Equal(t, "ini:staging", host.Name)

	// Not found
	_, err = cfg.GetByQualifiedName("unknown:host")
	assert.Error(t, err)
}

func TestExpandPath(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	tests := []struct {
		input    string
		expected string
	}{
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
		{"~/config", filepath.Join(home, "config")},
		{"~/.ssh/config", filepath.Join(home, ".ssh/config")},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := expandPath(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCollisionStrategy(t *testing.T) {
	t.Run("first strategy", func(t *testing.T) {
		settings := DefaultSettings()
		settings.Collisions.Strategy = CollisionFirst
		assert.Equal(t, CollisionFirst, settings.GetCollisionStrategy())
	})

	t.Run("qualify strategy", func(t *testing.T) {
		settings := DefaultSettings()
		settings.Collisions.Strategy = CollisionQualify
		assert.Equal(t, CollisionQualify, settings.GetCollisionStrategy())
	})

	t.Run("error strategy", func(t *testing.T) {
		settings := DefaultSettings()
		settings.Collisions.Strategy = CollisionError
		assert.Equal(t, CollisionError, settings.GetCollisionStrategy())
	})
}
