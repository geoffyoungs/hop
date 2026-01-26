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
	modes := []PathMode{ModeDefault, ModeLocal, ModeUser, ModeExplicit, ModeProject}
	seen := make(map[PathMode]bool)

	for _, mode := range modes {
		assert.False(t, seen[mode], "PathMode constant values should be unique")
		seen[mode] = true
	}
}

func TestConfigFileNames(t *testing.T) {
	names := ConfigFileNames()
	assert.Equal(t, []string{"hosts.ini", "hosts.conf"}, names)
}

func TestFindProjectConfig_CurrentDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Resolve symlinks for consistent path comparison (macOS /var -> /private/var)
	tmpDir, err := filepath.EvalSymlinks(tmpDir)
	require.NoError(t, err)

	// Create hosts.ini in the temp dir
	configPath := filepath.Join(tmpDir, "hosts.ini")
	err = os.WriteFile(configPath, []byte("[test]\n"), 0644)
	require.NoError(t, err)

	// Change to the temp dir
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldWd)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	path, found := FindProjectConfig()
	assert.True(t, found)
	assert.Equal(t, configPath, path)
}

func TestFindProjectConfig_ParentDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Resolve symlinks for consistent path comparison (macOS /var -> /private/var)
	tmpDir, err := filepath.EvalSymlinks(tmpDir)
	require.NoError(t, err)

	// Create hosts.ini in the parent dir
	configPath := filepath.Join(tmpDir, "hosts.ini")
	err = os.WriteFile(configPath, []byte("[test]\n"), 0644)
	require.NoError(t, err)

	// Create a subdirectory
	subDir := filepath.Join(tmpDir, "subdir")
	err = os.Mkdir(subDir, 0755)
	require.NoError(t, err)

	// Change to the subdirectory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldWd)

	err = os.Chdir(subDir)
	require.NoError(t, err)

	path, found := FindProjectConfig()
	assert.True(t, found)
	assert.Equal(t, configPath, path)
}

func TestFindProjectConfig_StopsAtGit(t *testing.T) {
	tmpDir := t.TempDir()

	// Resolve symlinks for consistent path comparison (macOS /var -> /private/var)
	tmpDir, err := filepath.EvalSymlinks(tmpDir)
	require.NoError(t, err)

	// Create .git directory (project root)
	gitDir := filepath.Join(tmpDir, ".git")
	err = os.Mkdir(gitDir, 0755)
	require.NoError(t, err)

	// Create hosts.ini in parent of project (should NOT be found)
	parentConfigPath := filepath.Join(filepath.Dir(tmpDir), "hosts.ini")
	// Don't create this - we just want to make sure we stop at .git

	// Create a subdirectory inside project
	subDir := filepath.Join(tmpDir, "subdir")
	err = os.Mkdir(subDir, 0755)
	require.NoError(t, err)

	// Change to the subdirectory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldWd)

	err = os.Chdir(subDir)
	require.NoError(t, err)

	path, found := FindProjectConfig()
	assert.False(t, found)
	assert.Empty(t, path)

	// Now create config in project root
	projectConfigPath := filepath.Join(tmpDir, "hosts.ini")
	err = os.WriteFile(projectConfigPath, []byte("[test]\n"), 0644)
	require.NoError(t, err)

	path, found = FindProjectConfig()
	assert.True(t, found)
	assert.Equal(t, projectConfigPath, path)

	// Clean up the parent config if it was somehow created
	os.Remove(parentConfigPath)
}

func TestFindProjectConfig_NotFound(t *testing.T) {
	tmpDir := t.TempDir()

	// Resolve symlinks for consistent path comparison (macOS /var -> /private/var)
	tmpDir, err := filepath.EvalSymlinks(tmpDir)
	require.NoError(t, err)

	// Create .git directory (project root) but no config
	gitDir := filepath.Join(tmpDir, ".git")
	err = os.Mkdir(gitDir, 0755)
	require.NoError(t, err)

	// Change to the temp dir
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldWd)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	path, found := FindProjectConfig()
	assert.False(t, found)
	assert.Empty(t, path)
}

func TestFindProjectConfig_PrefersIni(t *testing.T) {
	tmpDir := t.TempDir()

	// Resolve symlinks for consistent path comparison (macOS /var -> /private/var)
	tmpDir, err := filepath.EvalSymlinks(tmpDir)
	require.NoError(t, err)

	// Create both hosts.ini and hosts.conf
	iniPath := filepath.Join(tmpDir, "hosts.ini")
	confPath := filepath.Join(tmpDir, "hosts.conf")

	err = os.WriteFile(confPath, []byte("[conf]\n"), 0644)
	require.NoError(t, err)

	err = os.WriteFile(iniPath, []byte("[ini]\n"), 0644)
	require.NoError(t, err)

	// Change to the temp dir
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldWd)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	path, found := FindProjectConfig()
	assert.True(t, found)
	assert.Equal(t, iniPath, path) // Should prefer .ini over .conf
}

func TestFindProjectConfig_FallsBackToConf(t *testing.T) {
	tmpDir := t.TempDir()

	// Resolve symlinks for consistent path comparison (macOS /var -> /private/var)
	tmpDir, err := filepath.EvalSymlinks(tmpDir)
	require.NoError(t, err)

	// Create only hosts.conf
	confPath := filepath.Join(tmpDir, "hosts.conf")
	err = os.WriteFile(confPath, []byte("[conf]\n"), 0644)
	require.NoError(t, err)

	// Change to the temp dir
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldWd)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	path, found := FindProjectConfig()
	assert.True(t, found)
	assert.Equal(t, confPath, path)
}
