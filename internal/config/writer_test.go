package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddHost(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "hosts.ini")

	// Add first host
	err := AddHost(path, "production", map[string]string{
		"host": "192.168.1.100",
		"user": "admin",
	})
	require.NoError(t, err)

	// Verify the host was added
	cfg, err := LoadFromPath(path)
	require.NoError(t, err)

	host, ok := cfg.Get("production")
	require.True(t, ok)
	assert.Equal(t, "192.168.1.100", host.Host)
	assert.Equal(t, "admin", host.User)
}

func TestAddHost_CreatesDirIfNeeded(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "subdir", "nested", "hosts.ini")

	err := AddHost(path, "test", map[string]string{
		"host": "example.com",
	})
	require.NoError(t, err)

	// Verify file exists
	assert.True(t, ConfigExists(path))
}

func TestAddHost_AppendsToExisting(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "hosts.ini")

	// Add first host
	err := AddHost(path, "host1", map[string]string{"host": "1.1.1.1"})
	require.NoError(t, err)

	// Add second host
	err = AddHost(path, "host2", map[string]string{"host": "2.2.2.2"})
	require.NoError(t, err)

	// Verify both hosts exist
	cfg, err := LoadFromPath(path)
	require.NoError(t, err)

	_, ok1 := cfg.Get("host1")
	_, ok2 := cfg.Get("host2")
	assert.True(t, ok1)
	assert.True(t, ok2)
}

func TestAddHost_DuplicateError(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "hosts.ini")

	// Add first host
	err := AddHost(path, "myhost", map[string]string{"host": "example.com"})
	require.NoError(t, err)

	// Try to add duplicate
	err = AddHost(path, "myhost", map[string]string{"host": "other.com"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestAddHost_EmptyName(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "hosts.ini")

	err := AddHost(path, "", map[string]string{"host": "example.com"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be empty")
}

func TestAddHost_InvalidName(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "hosts.ini")

	tests := []struct {
		name   string
		errMsg string
	}{
		{"DEFAULT", "reserved"},
		{"[invalid]", "cannot contain"},
		{"  spaces  ", "whitespace"},
	}

	for _, tt := range tests {
		err := AddHost(path, tt.name, map[string]string{"host": "example.com"})
		assert.Error(t, err, "name %q should be invalid", tt.name)
		assert.Contains(t, err.Error(), tt.errMsg)
	}
}

func TestAddHost_UnknownProperty(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "hosts.ini")

	err := AddHost(path, "test", map[string]string{
		"host":    "example.com",
		"unknown": "value",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown property")
}

func TestAddHost_MissingRequiredSSH(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "hosts.ini")

	err := AddHost(path, "test", map[string]string{
		"user": "admin",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "'host' is required")
}

func TestAddHost_MissingRequiredDocker(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "hosts.ini")

	err := AddHost(path, "test", map[string]string{
		"type": "docker",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "one of 'container', 'label', 'image', or 'image_grep' is required")
}

func TestAddHost_MissingRequiredK8s(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "hosts.ini")

	err := AddHost(path, "test", map[string]string{
		"type":      "k8s",
		"namespace": "default",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "one of 'pod', 'selector', 'pod_grep', or 'deployment' is required")
}

func TestAddHost_K8sWithSelector(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "hosts.ini")

	err := AddHost(path, "test", map[string]string{
		"type":     "k8s",
		"selector": "app=myapp",
	})
	assert.NoError(t, err)
}

func TestAddHost_K8sWithPodGrep(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "hosts.ini")

	err := AddHost(path, "test", map[string]string{
		"type":     "k8s",
		"pod_grep": "myapp-scheduler",
	})
	assert.NoError(t, err)
}

func TestAddHost_InvalidBackendType(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "hosts.ini")

	err := AddHost(path, "test", map[string]string{
		"type": "invalid",
		"host": "example.com",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown backend type")
}

func TestAddHost_DockerBackend(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "hosts.ini")

	err := AddHost(path, "mycontainer", map[string]string{
		"type":      "docker",
		"container": "my_app",
		"shell":     "/bin/bash",
	})
	require.NoError(t, err)

	cfg, err := LoadFromPath(path)
	require.NoError(t, err)

	host, ok := cfg.Get("mycontainer")
	require.True(t, ok)
	assert.Equal(t, "docker", host.Type)
	assert.Equal(t, "my_app", host.Container)
}

func TestAddHost_K8sBackend(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "hosts.ini")

	err := AddHost(path, "mypod", map[string]string{
		"type":      "k8s",
		"namespace": "production",
		"pod":       "my-pod",
		"context":   "my-cluster",
	})
	require.NoError(t, err)

	cfg, err := LoadFromPath(path)
	require.NoError(t, err)

	host, ok := cfg.Get("mypod")
	require.True(t, ok)
	assert.Equal(t, "k8s", host.Type)
	assert.Equal(t, "production", host.Namespace)
	assert.Equal(t, "my-pod", host.Pod)
}

func TestUpdateHost(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "hosts.ini")

	// Add a host first
	err := AddHost(path, "test", map[string]string{"host": "old.example.com"})
	require.NoError(t, err)

	// Update it
	err = UpdateHost(path, "test", map[string]string{"host": "new.example.com", "user": "admin"})
	require.NoError(t, err)

	// Verify the update
	cfg, err := LoadFromPath(path)
	require.NoError(t, err)

	host, ok := cfg.Get("test")
	require.True(t, ok)
	assert.Equal(t, "new.example.com", host.Host)
	assert.Equal(t, "admin", host.User)
}

func TestUpdateHost_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "hosts.ini")

	// Create empty config
	err := CreateEmptyConfig(path)
	require.NoError(t, err)

	err = UpdateHost(path, "nonexistent", map[string]string{"host": "example.com"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRemoveHost(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "hosts.ini")

	// Add a host
	err := AddHost(path, "test", map[string]string{"host": "example.com"})
	require.NoError(t, err)

	// Remove it
	err = RemoveHost(path, "test")
	require.NoError(t, err)

	// Verify it's gone
	cfg, err := LoadFromPath(path)
	require.NoError(t, err)

	_, ok := cfg.Get("test")
	assert.False(t, ok)
}

func TestRemoveHost_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "hosts.ini")

	err := CreateEmptyConfig(path)
	require.NoError(t, err)

	err = RemoveHost(path, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestParseKeyValue(t *testing.T) {
	tests := []struct {
		input     string
		wantKey   string
		wantValue string
		wantErr   bool
	}{
		{"host=example.com", "host", "example.com", false},
		{"user=admin", "user", "admin", false},
		{"port=2222", "port", "2222", false},
		{"path=/some/path/with=equals", "path", "/some/path/with=equals", false},
		{"key = value with spaces", "key", "value with spaces", false},
		{"invalid", "", "", true},
		{"", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			key, value, err := ParseKeyValue(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantKey, key)
				assert.Equal(t, tt.wantValue, value)
			}
		})
	}
}

func TestParseKeyValuePairs(t *testing.T) {
	args := []string{"host=example.com", "user=admin", "port=2222"}
	props, err := ParseKeyValuePairs(args)
	require.NoError(t, err)

	assert.Equal(t, "example.com", props["host"])
	assert.Equal(t, "admin", props["user"])
	assert.Equal(t, "2222", props["port"])
}

func TestParseKeyValuePairs_Error(t *testing.T) {
	args := []string{"host=example.com", "invalid"}
	_, err := ParseKeyValuePairs(args)
	assert.Error(t, err)
}

func TestFormatHostEntry(t *testing.T) {
	props := map[string]string{
		"host": "example.com",
		"user": "admin",
		"port": "2222",
	}

	output := FormatHostEntry("myhost", props)

	assert.Contains(t, output, "[myhost]")
	assert.Contains(t, output, "host = example.com")
	assert.Contains(t, output, "user = admin")
	assert.Contains(t, output, "port = 2222")
}

func TestHostExists(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "hosts.ini")

	// No config file
	exists, err := HostExists(path, "test")
	require.NoError(t, err)
	assert.False(t, exists)

	// Add a host
	err = AddHost(path, "test", map[string]string{"host": "example.com"})
	require.NoError(t, err)

	// Check existing host
	exists, err = HostExists(path, "test")
	require.NoError(t, err)
	assert.True(t, exists)

	// Check non-existing host
	exists, err = HostExists(path, "other")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestCreateEmptyConfig(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "subdir", "hosts.ini")

	err := CreateEmptyConfig(path)
	require.NoError(t, err)

	// File should exist and be empty
	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, int64(0), info.Size())
}

func TestAddHost_SSHWithJumpAndAgentForward(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "hosts.ini")

	err := AddHost(path, "test", map[string]string{
		"host":          "example.com",
		"jump":          "bastion.example.com",
		"agent_forward": "yes",
	})
	require.NoError(t, err)

	cfg, err := LoadFromPath(path)
	require.NoError(t, err)

	host, ok := cfg.Get("test")
	require.True(t, ok)
	assert.Equal(t, "bastion.example.com", host.Jump)
	assert.True(t, host.AgentForward)
}

func TestAddHost_DockerWithLabel(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "hosts.ini")

	err := AddHost(path, "test", map[string]string{
		"type":  "docker",
		"label": "app=myapp",
	})
	require.NoError(t, err)

	cfg, err := LoadFromPath(path)
	require.NoError(t, err)

	host, ok := cfg.Get("test")
	require.True(t, ok)
	assert.Equal(t, "app=myapp", host.Label)
}

func TestAddHost_DockerWithImage(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "hosts.ini")

	err := AddHost(path, "test", map[string]string{
		"type":  "docker",
		"image": "nginx:latest",
	})
	require.NoError(t, err)

	cfg, err := LoadFromPath(path)
	require.NoError(t, err)

	host, ok := cfg.Get("test")
	require.True(t, ok)
	assert.Equal(t, "nginx:latest", host.Image)
}

func TestAddHost_DockerWithImageGrep(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "hosts.ini")

	err := AddHost(path, "test", map[string]string{
		"type":       "docker",
		"image_grep": "myapp",
	})
	require.NoError(t, err)

	cfg, err := LoadFromPath(path)
	require.NoError(t, err)

	host, ok := cfg.Get("test")
	require.True(t, ok)
	assert.Equal(t, "myapp", host.ImageGrep)
}

func TestAddHost_K8sWithDeployment(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "hosts.ini")

	err := AddHost(path, "test", map[string]string{
		"type":       "k8s",
		"deployment": "my-deploy",
	})
	require.NoError(t, err)

	cfg, err := LoadFromPath(path)
	require.NoError(t, err)

	host, ok := cfg.Get("test")
	require.True(t, ok)
	assert.Equal(t, "my-deploy", host.Deployment)
}

func TestAddHost_WithExtends(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "hosts.ini")

	err := AddHost(path, "test", map[string]string{
		"host":    "example.com",
		"extends": "base",
	})
	require.NoError(t, err)
}

func TestValidateHostProperties_NewProperties(t *testing.T) {
	// Test that all new properties are recognized as valid
	tests := []struct {
		name  string
		props map[string]string
	}{
		{
			name:  "ssh jump",
			props: map[string]string{"host": "example.com", "jump": "bastion"},
		},
		{
			name:  "ssh agent_forward",
			props: map[string]string{"host": "example.com", "agent_forward": "yes"},
		},
		{
			name:  "docker label",
			props: map[string]string{"type": "docker", "label": "app=myapp"},
		},
		{
			name:  "docker image",
			props: map[string]string{"type": "docker", "image": "nginx"},
		},
		{
			name:  "docker image_grep",
			props: map[string]string{"type": "docker", "image_grep": "myapp"},
		},
		{
			name:  "k8s deployment",
			props: map[string]string{"type": "k8s", "deployment": "my-deploy"},
		},
		{
			name:  "extends",
			props: map[string]string{"host": "example.com", "extends": "base"},
		},
		{
			name:  "default",
			props: map[string]string{"host": "example.com", "default": "yes"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			path := filepath.Join(tmpDir, "hosts.ini")

			err := AddHost(path, "test", tt.props)
			assert.NoError(t, err, "property should be valid")
		})
	}
}
