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

func TestHostConfig_ToMap_SSH(t *testing.T) {
	hc := &HostConfig{
		Name:         "production",
		Type:         "ssh",
		Host:         "192.168.1.100",
		User:         "admin",
		Port:         2222,
		Identity:     "~/.ssh/id_rsa",
		Jump:         "bastion.example.com",
		AgentForward: true,
	}

	props := hc.ToMap()

	// Type is not included when it's "ssh" (default)
	_, hasType := props["type"]
	assert.False(t, hasType, "ssh type should not be included")
	assert.Equal(t, "192.168.1.100", props["host"])
	assert.Equal(t, "admin", props["user"])
	assert.Equal(t, "2222", props["port"])
	assert.Equal(t, "~/.ssh/id_rsa", props["identity"])
	assert.Equal(t, "bastion.example.com", props["jump"])
	assert.Equal(t, "yes", props["agent_forward"])
}

func TestHostConfig_ToMap_Docker(t *testing.T) {
	hc := &HostConfig{
		Name:      "mycontainer",
		Type:      "docker",
		Container: "my_app",
		Shell:     "/bin/bash",
		Label:     "app=myapp",
		Image:     "nginx:latest",
		ImageGrep: "myapp",
	}

	props := hc.ToMap()

	assert.Equal(t, "docker", props["type"])
	assert.Equal(t, "my_app", props["container"])
	assert.Equal(t, "/bin/bash", props["shell"])
	assert.Equal(t, "app=myapp", props["label"])
	assert.Equal(t, "nginx:latest", props["image"])
	assert.Equal(t, "myapp", props["image_grep"])
}

func TestHostConfig_ToMap_K8s(t *testing.T) {
	hc := &HostConfig{
		Name:       "mypod",
		Type:       "k8s",
		Namespace:  "production",
		Pod:        "my-pod",
		Container:  "app",
		Context:    "my-cluster",
		Selector:   "app=web",
		PodGrep:    "scheduler",
		Deployment: "my-deploy",
	}

	props := hc.ToMap()

	assert.Equal(t, "k8s", props["type"])
	assert.Equal(t, "production", props["namespace"])
	assert.Equal(t, "my-pod", props["pod"])
	assert.Equal(t, "app", props["container"])
	assert.Equal(t, "my-cluster", props["context"])
	assert.Equal(t, "app=web", props["selector"])
	assert.Equal(t, "scheduler", props["pod_grep"])
	assert.Equal(t, "my-deploy", props["deployment"])
}

func TestHostConfig_ToMap_DefaultsNotIncluded(t *testing.T) {
	hc := &HostConfig{
		Name:      "minimal",
		Type:      "ssh",
		Host:      "example.com",
		Shell:     "/bin/sh",
		Namespace: "default",
	}

	props := hc.ToMap()

	// Default values should not be included
	_, hasType := props["type"]
	assert.False(t, hasType, "default type 'ssh' should not be included")

	_, hasShell := props["shell"]
	assert.False(t, hasShell, "default shell '/bin/sh' should not be included")

	_, hasNamespace := props["namespace"]
	assert.False(t, hasNamespace, "default namespace 'default' should not be included")
}

func TestHostConfig_ToMap_Default(t *testing.T) {
	hc := &HostConfig{
		Name:    "production",
		Host:    "example.com",
		Default: true,
	}

	props := hc.ToMap()

	assert.Equal(t, "yes", props["default"])
}

func TestHostConfig_ToMap_Extends(t *testing.T) {
	hc := &HostConfig{
		Name:    "child",
		Host:    "example.com",
		Extends: "parent",
	}

	props := hc.ToMap()

	assert.Equal(t, "parent", props["extends"])
}

func TestHostConfig_ToHost_NewSSHFields(t *testing.T) {
	hc := &HostConfig{
		Name:         "production",
		Type:         "ssh",
		Host:         "example.com",
		Jump:         "bastion",
		AgentForward: true,
	}

	host := hc.ToHost()

	assert.Equal(t, "bastion", host.Properties["jump"])
	assert.Equal(t, "yes", host.Properties["agent_forward"])
}

func TestHostConfig_ToHost_NewDockerFields(t *testing.T) {
	hc := &HostConfig{
		Name:      "mycontainer",
		Type:      "docker",
		Label:     "app=myapp",
		Image:     "nginx",
		ImageGrep: "myapp",
	}

	host := hc.ToHost()

	assert.Equal(t, "app=myapp", host.Properties["label"])
	assert.Equal(t, "nginx", host.Properties["image"])
	assert.Equal(t, "myapp", host.Properties["image_grep"])
}

func TestHostConfig_ToHost_NewK8sFields(t *testing.T) {
	hc := &HostConfig{
		Name:       "mypod",
		Type:       "k8s",
		Selector:   "app=web",
		PodGrep:    "scheduler",
		Deployment: "my-deploy",
	}

	host := hc.ToHost()

	assert.Equal(t, "app=web", host.Properties["selector"])
	assert.Equal(t, "scheduler", host.Properties["pod_grep"])
	assert.Equal(t, "my-deploy", host.Properties["deployment"])
}
