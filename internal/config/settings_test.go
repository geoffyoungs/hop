package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultSettings(t *testing.T) {
	settings := DefaultSettings()

	// Check default enabled sources
	assert.True(t, settings.IsSourceEnabled("ini"))
	assert.True(t, settings.IsSourceEnabled("ansible"))
	assert.False(t, settings.IsSourceEnabled("sshconfig")) // Opt-in
	assert.True(t, settings.IsSourceEnabled("vagrant"))

	// Check default priority
	priority := settings.GetPriority()
	assert.Equal(t, []string{"ini", "ansible", "vagrant", "sshconfig"}, priority)

	// Check default collision strategy
	assert.Equal(t, CollisionFirst, settings.GetCollisionStrategy())
}

func TestLoadSettingsFromPath_Defaults(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.toml")

	// Load from non-existent file should return defaults
	settings, err := LoadSettingsFromPath(settingsPath)
	require.NoError(t, err)

	assert.True(t, settings.IsSourceEnabled("ini"))
	assert.True(t, settings.IsSourceEnabled("ansible"))
	assert.False(t, settings.IsSourceEnabled("sshconfig"))
}

func TestLoadSettingsFromPath_Override(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.toml")

	content := `[sources]
ini = true
ansible = false
sshconfig = true
vagrant = true
priority = ["ini", "sshconfig"]

[collisions]
strategy = "qualify"
`

	err := os.WriteFile(settingsPath, []byte(content), 0644)
	require.NoError(t, err)

	settings, err := LoadSettingsFromPath(settingsPath)
	require.NoError(t, err)

	assert.True(t, settings.IsSourceEnabled("ini"))
	assert.False(t, settings.IsSourceEnabled("ansible"))
	assert.True(t, settings.IsSourceEnabled("sshconfig"))
	assert.True(t, settings.IsSourceEnabled("vagrant"))

	assert.Equal(t, []string{"ini", "sshconfig"}, settings.GetPriority())
	assert.Equal(t, CollisionQualify, settings.GetCollisionStrategy())
}

func TestLoadSettingsFromPath_CustomPaths(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.toml")

	content := `[sources.paths]
ansible = ["~/my-inventory.yml", "/etc/ansible/hosts"]
sshconfig = ["~/.ssh/config", "~/.ssh/work_config"]
`

	err := os.WriteFile(settingsPath, []byte(content), 0644)
	require.NoError(t, err)

	settings, err := LoadSettingsFromPath(settingsPath)
	require.NoError(t, err)

	ansiblePaths := settings.GetSourcePaths("ansible")
	assert.Equal(t, []string{"~/my-inventory.yml", "/etc/ansible/hosts"}, ansiblePaths)

	sshPaths := settings.GetSourcePaths("sshconfig")
	assert.Equal(t, []string{"~/.ssh/config", "~/.ssh/work_config"}, sshPaths)
}

func TestIsSourceEnabled_UnknownSource(t *testing.T) {
	settings := DefaultSettings()
	assert.False(t, settings.IsSourceEnabled("unknown"))
}

func TestGetSourcePaths_UnknownSource(t *testing.T) {
	settings := DefaultSettings()
	paths := settings.GetSourcePaths("unknown")
	assert.Nil(t, paths)
}

func TestCollisionStrategy_Values(t *testing.T) {
	assert.Equal(t, CollisionStrategy("first"), CollisionFirst)
	assert.Equal(t, CollisionStrategy("qualify"), CollisionQualify)
	assert.Equal(t, CollisionStrategy("error"), CollisionError)
}
