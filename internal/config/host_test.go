package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHostConfig_ToHost_SSH(t *testing.T) {
	hc := &HostConfig{
		Name:     "production",
		Type:     "ssh",
		Host:     "192.168.1.100",
		User:     "admin",
		Port:     2222,
		Identity: "~/.ssh/id_rsa",
	}

	host := hc.ToHost()

	assert.Equal(t, "production", host.Name)
	assert.Equal(t, "ssh", host.Type)
	assert.Equal(t, "192.168.1.100", host.Properties["host"])
	assert.Equal(t, "admin", host.Properties["user"])
	assert.Equal(t, "2222", host.Properties["port"])
	assert.Equal(t, "~/.ssh/id_rsa", host.Properties["identity"])
}

func TestHostConfig_ToHost_SSHMinimal(t *testing.T) {
	hc := &HostConfig{
		Name: "minimal",
		Host: "example.com",
	}

	host := hc.ToHost()

	assert.Equal(t, "minimal", host.Name)
	assert.Equal(t, "ssh", host.Type, "should default to ssh")
	assert.Equal(t, "example.com", host.Properties["host"])
	assert.Empty(t, host.Properties["user"])
	assert.Empty(t, host.Properties["port"])
}

func TestHostConfig_ToHost_Docker(t *testing.T) {
	hc := &HostConfig{
		Name:      "mycontainer",
		Type:      "docker",
		Container: "my_app",
		Shell:     "/bin/bash",
	}

	host := hc.ToHost()

	assert.Equal(t, "mycontainer", host.Name)
	assert.Equal(t, "docker", host.Type)
	assert.Equal(t, "my_app", host.Properties["container"])
	assert.Equal(t, "/bin/bash", host.Properties["shell"])
}

func TestHostConfig_ToHost_K8s(t *testing.T) {
	hc := &HostConfig{
		Name:      "mypod",
		Type:      "k8s",
		Namespace: "production",
		Pod:       "my-pod",
		Container: "app",
		Context:   "my-cluster",
		Shell:     "/bin/bash",
	}

	host := hc.ToHost()

	assert.Equal(t, "mypod", host.Name)
	assert.Equal(t, "k8s", host.Type)
	assert.Equal(t, "production", host.Properties["namespace"])
	assert.Equal(t, "my-pod", host.Properties["pod"])
	assert.Equal(t, "app", host.Properties["container"])
	assert.Equal(t, "my-cluster", host.Properties["context"])
	assert.Equal(t, "/bin/bash", host.Properties["shell"])
}

func TestHostConfig_ToHost_PortForwarding(t *testing.T) {
	hc := &HostConfig{
		Name:       "webserver",
		Host:       "example.com",
		LocalPort:  8080,
		RemotePort: 80,
	}

	host := hc.ToHost()

	assert.Equal(t, "8080", host.Properties["local_port"])
	assert.Equal(t, "80", host.Properties["remote_port"])
}

func TestHostConfig_ToHost_EmptyPortNotIncluded(t *testing.T) {
	hc := &HostConfig{
		Name: "minimal",
		Host: "example.com",
		Port: 0, // Not set
	}

	host := hc.ToHost()

	// Port should not be in properties when it's 0
	_, hasPort := host.Properties["port"]
	assert.False(t, hasPort, "port should not be included when 0")
}

func TestHostConfig_ToHost_DefaultType(t *testing.T) {
	tests := []struct {
		name     string
		hostType string
		expected string
	}{
		{
			name:     "empty type defaults to ssh",
			hostType: "",
			expected: "ssh",
		},
		{
			name:     "explicit ssh",
			hostType: "ssh",
			expected: "ssh",
		},
		{
			name:     "docker type",
			hostType: "docker",
			expected: "docker",
		},
		{
			name:     "k8s type",
			hostType: "k8s",
			expected: "k8s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hc := &HostConfig{
				Name: "test",
				Type: tt.hostType,
				Host: "example.com",
			}
			host := hc.ToHost()
			assert.Equal(t, tt.expected, host.Type)
		})
	}
}

func TestItoa(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, ""},
		{1, "1"},
		{22, "22"},
		{2222, "2222"},
		{65535, "65535"},
		{-1, "-1"},
		{-100, "-100"},
	}

	for _, tt := range tests {
		result := itoa(tt.input)
		assert.Equal(t, tt.expected, result, "itoa(%d)", tt.input)
	}
}

func TestUitoa(t *testing.T) {
	tests := []struct {
		input    uint
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{22, "22"},
		{2222, "2222"},
		{65535, "65535"},
	}

	for _, tt := range tests {
		result := uitoa(tt.input)
		assert.Equal(t, tt.expected, result, "uitoa(%d)", tt.input)
	}
}
