package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalConfigPath(t *testing.T) {
	path := LocalConfigPath()
	assert.Equal(t, "hosts.ini", path)
}

func TestUserConfigPath(t *testing.T) {
	// UserConfigPath should return the same as DefaultConfigPath
	path := UserConfigPath()
	assert.Equal(t, DefaultConfigPath(), path)
}

func TestResolvePath(t *testing.T) {
	tests := []struct {
		name     string
		mode     PathMode
		explicit string
		want     string
	}{
		{
			name:     "default mode",
			mode:     ModeDefault,
			explicit: "",
			want:     DefaultConfigPath(),
		},
		{
			name:     "local mode",
			mode:     ModeLocal,
			explicit: "",
			want:     "hosts.ini",
		},
		{
			name:     "user mode",
			mode:     ModeUser,
			explicit: "",
			want:     DefaultConfigPath(),
		},
		{
			name:     "explicit mode with path",
			mode:     ModeExplicit,
			explicit: "/custom/path/hosts.ini",
			want:     "/custom/path/hosts.ini",
		},
		{
			name:     "explicit mode without path falls back to default",
			mode:     ModeExplicit,
			explicit: "",
			want:     DefaultConfigPath(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ResolvePath(tt.mode, tt.explicit)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestEnsureConfigDir(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "subdir", "nested", "hosts.ini")

	err := EnsureConfigDir(path)
	require.NoError(t, err)

	// Check that the directory was created
	dir := filepath.Dir(path)
	info, err := os.Stat(dir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestConfigExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Test non-existent file
	assert.False(t, ConfigExists(filepath.Join(tmpDir, "nonexistent.ini")))

	// Create a file and test
	path := filepath.Join(tmpDir, "hosts.ini")
	err := os.WriteFile(path, []byte("[test]\n"), 0644)
	require.NoError(t, err)

	assert.True(t, ConfigExists(path))
}

func TestPathModeConstants(t *testing.T) {
	// Ensure constants have distinct values
	modes := []PathMode{ModeDefault, ModeLocal, ModeUser, ModeExplicit}
	seen := make(map[PathMode]bool)

	for _, mode := range modes {
		assert.False(t, seen[mode], "PathMode constant values should be unique")
		seen[mode] = true
	}
}
