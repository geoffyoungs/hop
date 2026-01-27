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

func TestDefaultHost_SingleHost(t *testing.T) {
	cfg := &Config{
		Hosts: map[string]*HostConfig{
			"only": {Name: "only", Host: "example.com"},
		},
	}

	host, ok := cfg.DefaultHost()
	assert.True(t, ok)
	assert.NotNil(t, host)
	assert.Equal(t, "only", host.Name)
}

func TestDefaultHost_MarkedDefault(t *testing.T) {
	cfg := &Config{
		Hosts: map[string]*HostConfig{
			"first":  {Name: "first", Host: "first.example.com"},
			"second": {Name: "second", Host: "second.example.com", Default: true},
			"third":  {Name: "third", Host: "third.example.com"},
		},
	}

	host, ok := cfg.DefaultHost()
	assert.True(t, ok)
	assert.NotNil(t, host)
	assert.Equal(t, "second", host.Name)
}

func TestDefaultHost_MultipleNoDefault(t *testing.T) {
	cfg := &Config{
		Hosts: map[string]*HostConfig{
			"first":  {Name: "first", Host: "first.example.com"},
			"second": {Name: "second", Host: "second.example.com"},
		},
	}

	host, ok := cfg.DefaultHost()
	assert.False(t, ok)
	assert.Nil(t, host)
}

func TestDefaultHost_Empty(t *testing.T) {
	cfg := &Config{
		Hosts: map[string]*HostConfig{},
	}

	host, ok := cfg.DefaultHost()
	assert.False(t, ok)
	assert.Nil(t, host)
}

func TestLoadFromPath_DefaultYes(t *testing.T) {
	tmpDir := t.TempDir()
	iniPath := filepath.Join(tmpDir, "hosts.ini")

	content := `[production]
host = prod.example.com
default = yes

[staging]
host = staging.example.com
`

	err := os.WriteFile(iniPath, []byte(content), 0644)
	require.NoError(t, err)

	cfg, err := LoadFromPath(iniPath)
	require.NoError(t, err)

	prod, ok := cfg.Get("production")
	require.True(t, ok)
	assert.True(t, prod.Default)

	staging, ok := cfg.Get("staging")
	require.True(t, ok)
	assert.False(t, staging.Default)
}

func TestLoadFromPath_DefaultTrue(t *testing.T) {
	tmpDir := t.TempDir()
	iniPath := filepath.Join(tmpDir, "hosts.ini")

	content := `[production]
host = prod.example.com
default = true

[staging]
host = staging.example.com
`

	err := os.WriteFile(iniPath, []byte(content), 0644)
	require.NoError(t, err)

	cfg, err := LoadFromPath(iniPath)
	require.NoError(t, err)

	prod, ok := cfg.Get("production")
	require.True(t, ok)
	assert.True(t, prod.Default)
}

func TestGetByPrefix_ExactMatch(t *testing.T) {
	cfg := &Config{
		Hosts: map[string]*HostConfig{
			"production":  {Name: "production", Host: "prod.example.com"},
			"prod-backup": {Name: "prod-backup", Host: "backup.example.com"},
		},
	}

	// Exact match should return production, even though prod-backup also starts with "prod"
	host, err := cfg.GetByPrefix("production")
	require.NoError(t, err)
	assert.Equal(t, "production", host.Name)
}

func TestGetByPrefix_UniquePrefix(t *testing.T) {
	cfg := &Config{
		Hosts: map[string]*HostConfig{
			"production": {Name: "production", Host: "prod.example.com"},
			"staging":    {Name: "staging", Host: "staging.example.com"},
		},
	}

	// "prod" uniquely matches "production"
	host, err := cfg.GetByPrefix("prod")
	require.NoError(t, err)
	assert.Equal(t, "production", host.Name)

	// "stag" uniquely matches "staging"
	host, err = cfg.GetByPrefix("stag")
	require.NoError(t, err)
	assert.Equal(t, "staging", host.Name)
}

func TestGetByPrefix_AmbiguousPrefix(t *testing.T) {
	cfg := &Config{
		Hosts: map[string]*HostConfig{
			"production":  {Name: "production", Host: "prod.example.com"},
			"prod-backup": {Name: "prod-backup", Host: "backup.example.com"},
		},
	}

	// "prod" matches both hosts
	host, err := cfg.GetByPrefix("prod")
	assert.Nil(t, host)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ambiguous host")
	assert.Contains(t, err.Error(), "prod-backup")
	assert.Contains(t, err.Error(), "production")
}

func TestGetByPrefix_NoMatch(t *testing.T) {
	cfg := &Config{
		Hosts: map[string]*HostConfig{
			"production": {Name: "production", Host: "prod.example.com"},
			"staging":    {Name: "staging", Host: "staging.example.com"},
		},
	}

	host, err := cfg.GetByPrefix("xyz")
	assert.Nil(t, host)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestGetByPrefix_SingleCharPrefix(t *testing.T) {
	cfg := &Config{
		Hosts: map[string]*HostConfig{
			"production": {Name: "production", Host: "prod.example.com"},
			"staging":    {Name: "staging", Host: "staging.example.com"},
		},
	}

	// "p" uniquely matches "production"
	host, err := cfg.GetByPrefix("p")
	require.NoError(t, err)
	assert.Equal(t, "production", host.Name)

	// "s" uniquely matches "staging"
	host, err = cfg.GetByPrefix("s")
	require.NoError(t, err)
	assert.Equal(t, "staging", host.Name)
}

func TestLoadFromPath_EnvironmentExpansion(t *testing.T) {
	tmpDir := t.TempDir()
	iniPath := filepath.Join(tmpDir, "hosts.ini")

	// Set up environment variables for testing
	os.Setenv("TEST_HOP_HOST", "env.example.com")
	os.Setenv("TEST_HOP_USER", "envuser")
	defer os.Unsetenv("TEST_HOP_HOST")
	defer os.Unsetenv("TEST_HOP_USER")

	content := `[envhost]
host = $TEST_HOP_HOST
user = ${TEST_HOP_USER}
port = 22
`

	err := os.WriteFile(iniPath, []byte(content), 0644)
	require.NoError(t, err)

	cfg, err := LoadFromPath(iniPath)
	require.NoError(t, err)

	host, ok := cfg.Get("envhost")
	require.True(t, ok)
	assert.Equal(t, "env.example.com", host.Host, "should expand $VAR syntax")
	assert.Equal(t, "envuser", host.User, "should expand ${VAR} syntax")
}

func TestLoadFromPath_EnvironmentExpansion_Unset(t *testing.T) {
	tmpDir := t.TempDir()
	iniPath := filepath.Join(tmpDir, "hosts.ini")

	// Ensure variable is not set
	os.Unsetenv("TEST_HOP_UNSET_VAR")

	content := `[envhost]
host = $TEST_HOP_UNSET_VAR
`

	err := os.WriteFile(iniPath, []byte(content), 0644)
	require.NoError(t, err)

	cfg, err := LoadFromPath(iniPath)
	require.NoError(t, err)

	host, ok := cfg.Get("envhost")
	require.True(t, ok)
	assert.Equal(t, "", host.Host, "unset variable should expand to empty string")
}

func TestLoadFromPath_Inheritance(t *testing.T) {
	tmpDir := t.TempDir()
	iniPath := filepath.Join(tmpDir, "hosts.ini")

	content := `[base]
type = k8s
context = prod-cluster
namespace = production

[api]
extends = base
pod_grep = api-server
shell = /bin/bash

[worker]
extends = base
pod_grep = worker
`

	err := os.WriteFile(iniPath, []byte(content), 0644)
	require.NoError(t, err)

	cfg, err := LoadFromPath(iniPath)
	require.NoError(t, err)

	// Test api inherits from base
	api, ok := cfg.Get("api")
	require.True(t, ok)
	assert.Equal(t, "k8s", api.Type, "should inherit type from base")
	assert.Equal(t, "prod-cluster", api.Context, "should inherit context from base")
	assert.Equal(t, "production", api.Namespace, "should inherit namespace from base")
	assert.Equal(t, "api-server", api.PodGrep, "should have own pod_grep")
	assert.Equal(t, "/bin/bash", api.Shell, "should have own shell")

	// Test worker inherits from base
	worker, ok := cfg.Get("worker")
	require.True(t, ok)
	assert.Equal(t, "k8s", worker.Type, "should inherit type from base")
	assert.Equal(t, "prod-cluster", worker.Context, "should inherit context from base")
	assert.Equal(t, "production", worker.Namespace, "should inherit namespace from base")
	assert.Equal(t, "worker", worker.PodGrep, "should have own pod_grep")
}

func TestLoadFromPath_InheritanceOverride(t *testing.T) {
	tmpDir := t.TempDir()
	iniPath := filepath.Join(tmpDir, "hosts.ini")

	content := `[base]
type = k8s
context = prod-cluster
namespace = production
shell = /bin/sh

[staging]
extends = base
context = staging-cluster
namespace = staging
pod_grep = app
`

	err := os.WriteFile(iniPath, []byte(content), 0644)
	require.NoError(t, err)

	cfg, err := LoadFromPath(iniPath)
	require.NoError(t, err)

	staging, ok := cfg.Get("staging")
	require.True(t, ok)
	assert.Equal(t, "k8s", staging.Type, "should inherit type")
	assert.Equal(t, "staging-cluster", staging.Context, "should override context")
	assert.Equal(t, "staging", staging.Namespace, "should override namespace")
	assert.Equal(t, "/bin/sh", staging.Shell, "should inherit shell")
}

func TestLoadFromPath_InheritanceChain(t *testing.T) {
	tmpDir := t.TempDir()
	iniPath := filepath.Join(tmpDir, "hosts.ini")

	content := `[grandparent]
type = k8s
context = cluster

[parent]
extends = grandparent
namespace = myns

[child]
extends = parent
pod_grep = app
`

	err := os.WriteFile(iniPath, []byte(content), 0644)
	require.NoError(t, err)

	cfg, err := LoadFromPath(iniPath)
	require.NoError(t, err)

	child, ok := cfg.Get("child")
	require.True(t, ok)
	assert.Equal(t, "k8s", child.Type, "should inherit type from grandparent")
	assert.Equal(t, "cluster", child.Context, "should inherit context from grandparent")
	assert.Equal(t, "myns", child.Namespace, "should inherit namespace from parent")
	assert.Equal(t, "app", child.PodGrep, "should have own pod_grep")
}

func TestLoadFromPath_InheritanceCycleDetection(t *testing.T) {
	tmpDir := t.TempDir()
	iniPath := filepath.Join(tmpDir, "hosts.ini")

	content := `[a]
extends = b
host = a.example.com

[b]
extends = a
host = b.example.com
`

	err := os.WriteFile(iniPath, []byte(content), 0644)
	require.NoError(t, err)

	_, err = LoadFromPath(iniPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "circular inheritance")
}

func TestLoadFromPath_InheritanceMissingParent(t *testing.T) {
	tmpDir := t.TempDir()
	iniPath := filepath.Join(tmpDir, "hosts.ini")

	content := `[child]
extends = nonexistent
host = example.com
`

	err := os.WriteFile(iniPath, []byte(content), 0644)
	require.NoError(t, err)

	_, err = LoadFromPath(iniPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestLoadFromPath_NewFields(t *testing.T) {
	tmpDir := t.TempDir()
	iniPath := filepath.Join(tmpDir, "hosts.ini")

	content := `[ssh-host]
type = ssh
host = example.com
jump = bastion.example.com
agent_forward = yes

[docker-host]
type = docker
label = app=myapp
image = nginx:latest
image_grep = myapp

[k8s-host]
type = k8s
selector = app=web
pod_grep = scheduler
deployment = my-deploy
`

	err := os.WriteFile(iniPath, []byte(content), 0644)
	require.NoError(t, err)

	cfg, err := LoadFromPath(iniPath)
	require.NoError(t, err)

	// Test SSH fields
	ssh, ok := cfg.Get("ssh-host")
	require.True(t, ok)
	assert.Equal(t, "bastion.example.com", ssh.Jump)
	assert.True(t, ssh.AgentForward)

	// Test Docker fields
	docker, ok := cfg.Get("docker-host")
	require.True(t, ok)
	assert.Equal(t, "app=myapp", docker.Label)
	assert.Equal(t, "nginx:latest", docker.Image)
	assert.Equal(t, "myapp", docker.ImageGrep)

	// Test K8s fields
	k8s, ok := cfg.Get("k8s-host")
	require.True(t, ok)
	assert.Equal(t, "app=web", k8s.Selector)
	assert.Equal(t, "scheduler", k8s.PodGrep)
	assert.Equal(t, "my-deploy", k8s.Deployment)
}
