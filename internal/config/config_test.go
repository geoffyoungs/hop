package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadFromPath(t *testing.T) {
	// Create a temporary INI file
	tmpDir := t.TempDir()
	iniPath := filepath.Join(tmpDir, "hosts.ini")

	content := `[production]
host = 192.168.1.100
user = admin
port = 22

[staging]
type = ssh
host = 10.0.0.50
user = deploy
port = 2222
identity = ~/.ssh/staging_key

[mycontainer]
type = docker
container = my_app
shell = /bin/bash

[mypod]
type = k8s
namespace = production
pod = my-pod
container = app
context = my-cluster
`

	err := os.WriteFile(iniPath, []byte(content), 0644)
	require.NoError(t, err)

	cfg, err := LoadFromPath(iniPath)
	require.NoError(t, err)
	assert.NotNil(t, cfg)

	// Test that all hosts were loaded
	assert.Len(t, cfg.Hosts, 4)

	// Test production host
	prod, ok := cfg.Get("production")
	require.True(t, ok)
	assert.Equal(t, "production", prod.Name)
	assert.Equal(t, "ssh", prod.Type)
	assert.Equal(t, "192.168.1.100", prod.Host)
	assert.Equal(t, "admin", prod.User)
	assert.Equal(t, 22, prod.Port)

	// Test staging host
	staging, ok := cfg.Get("staging")
	require.True(t, ok)
	assert.Equal(t, "staging", staging.Name)
	assert.Equal(t, "ssh", staging.Type)
	assert.Equal(t, "10.0.0.50", staging.Host)
	assert.Equal(t, "deploy", staging.User)
	assert.Equal(t, 2222, staging.Port)
	assert.Equal(t, "~/.ssh/staging_key", staging.Identity)

	// Test docker host
	docker, ok := cfg.Get("mycontainer")
	require.True(t, ok)
	assert.Equal(t, "mycontainer", docker.Name)
	assert.Equal(t, "docker", docker.Type)
	assert.Equal(t, "my_app", docker.Container)
	assert.Equal(t, "/bin/bash", docker.Shell)

	// Test k8s host
	k8s, ok := cfg.Get("mypod")
	require.True(t, ok)
	assert.Equal(t, "mypod", k8s.Name)
	assert.Equal(t, "k8s", k8s.Type)
	assert.Equal(t, "production", k8s.Namespace)
	assert.Equal(t, "my-pod", k8s.Pod)
	assert.Equal(t, "app", k8s.Container)
	assert.Equal(t, "my-cluster", k8s.Context)
}

func TestLoadFromPath_Defaults(t *testing.T) {
	tmpDir := t.TempDir()
	iniPath := filepath.Join(tmpDir, "hosts.ini")

	content := `[minimal]
host = example.com
`

	err := os.WriteFile(iniPath, []byte(content), 0644)
	require.NoError(t, err)

	cfg, err := LoadFromPath(iniPath)
	require.NoError(t, err)

	host, ok := cfg.Get("minimal")
	require.True(t, ok)

	// Check defaults
	assert.Equal(t, "ssh", host.Type, "type should default to ssh")
	assert.Equal(t, "/bin/sh", host.Shell, "shell should default to /bin/sh")
	assert.Equal(t, "default", host.Namespace, "namespace should default to default")
	assert.Equal(t, 0, host.Port, "port should be 0 if not specified")
}

func TestLoadFromPath_FileNotFound(t *testing.T) {
	_, err := LoadFromPath("/nonexistent/path/hosts.ini")
	assert.Error(t, err)
}

func TestLoadFromPath_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	iniPath := filepath.Join(tmpDir, "hosts.ini")

	err := os.WriteFile(iniPath, []byte(""), 0644)
	require.NoError(t, err)

	cfg, err := LoadFromPath(iniPath)
	require.NoError(t, err)
	assert.Empty(t, cfg.Hosts)
}

func TestLoadFromPath_PortForwarding(t *testing.T) {
	tmpDir := t.TempDir()
	iniPath := filepath.Join(tmpDir, "hosts.ini")

	content := `[webserver]
host = example.com
local_port = 8080
remote_port = 80
`

	err := os.WriteFile(iniPath, []byte(content), 0644)
	require.NoError(t, err)

	cfg, err := LoadFromPath(iniPath)
	require.NoError(t, err)

	host, ok := cfg.Get("webserver")
	require.True(t, ok)
	assert.Equal(t, 8080, host.LocalPort)
	assert.Equal(t, 80, host.RemotePort)
}

func TestConfig_Get(t *testing.T) {
	cfg := &Config{
		Hosts: map[string]*HostConfig{
			"test": {Name: "test", Host: "example.com"},
		},
	}

	// Existing host
	host, ok := cfg.Get("test")
	assert.True(t, ok)
	assert.NotNil(t, host)
	assert.Equal(t, "test", host.Name)

	// Non-existing host
	host, ok = cfg.Get("nonexistent")
	assert.False(t, ok)
	assert.Nil(t, host)
}

func TestConfig_Names(t *testing.T) {
	cfg := &Config{
		Hosts: map[string]*HostConfig{
			"alpha":   {Name: "alpha"},
			"beta":    {Name: "beta"},
			"charlie": {Name: "charlie"},
		},
	}

	names := cfg.Names()
	assert.Len(t, names, 3)
	assert.Contains(t, names, "alpha")
	assert.Contains(t, names, "beta")
	assert.Contains(t, names, "charlie")
}

func TestConfig_Names_Empty(t *testing.T) {
	cfg := &Config{
		Hosts: map[string]*HostConfig{},
	}

	names := cfg.Names()
	assert.Empty(t, names)
}

func TestDefaultConfigPath(t *testing.T) {
	// Test with XDG_CONFIG_HOME set
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", originalXDG)

	os.Setenv("XDG_CONFIG_HOME", "/custom/config")
	path := DefaultConfigPath()
	assert.Equal(t, "/custom/config/hop/hosts.ini", path)

	// Test without XDG_CONFIG_HOME (uses ~/.config)
	os.Unsetenv("XDG_CONFIG_HOME")
	path = DefaultConfigPath()
	home, err := os.UserHomeDir()
	require.NoError(t, err)
	expected := filepath.Join(home, ".config", "hop", "hosts.ini")
	assert.Equal(t, expected, path)
}
