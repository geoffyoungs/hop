package source

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestINISource_Load(t *testing.T) {
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
`

	err := os.WriteFile(iniPath, []byte(content), 0644)
	require.NoError(t, err)

	src := &INISource{}

	assert.True(t, src.CanLoad(iniPath))
	assert.Equal(t, "ini", src.Name())
	assert.True(t, src.IsWritable())

	entries, err := src.Load(iniPath)
	require.NoError(t, err)
	assert.Len(t, entries, 3)

	// Find production entry
	var prod *HostEntry
	for _, e := range entries {
		if e.Name == "production" {
			prod = e
			break
		}
	}
	require.NotNil(t, prod)
	assert.Equal(t, "ssh", prod.Type)
	assert.Equal(t, "192.168.1.100", prod.Properties["host"])
	assert.Equal(t, "admin", prod.Properties["user"])
}

func TestINISource_CanLoad(t *testing.T) {
	src := &INISource{}

	tests := []struct {
		path     string
		expected bool
	}{
		{"hosts.ini", true},
		{"hosts.conf", true},
		{"/path/to/hosts.ini", true},
		{"/path/to/config.ini", true},
		{"/path/to/file.yaml", false},
		{"/path/to/file.yml", false},
		{"/path/to/inventory.yml", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			assert.Equal(t, tt.expected, src.CanLoad(tt.path))
		})
	}
}

func TestINISource_Inheritance(t *testing.T) {
	tmpDir := t.TempDir()
	iniPath := filepath.Join(tmpDir, "hosts.ini")

	content := `[base]
type = k8s
context = prod-cluster
namespace = production

[api]
extends = base
pod_grep = api-server
`

	err := os.WriteFile(iniPath, []byte(content), 0644)
	require.NoError(t, err)

	src := &INISource{}
	entries, err := src.Load(iniPath)
	require.NoError(t, err)

	// Find api entry
	var api *HostEntry
	for _, e := range entries {
		if e.Name == "api" {
			api = e
			break
		}
	}
	require.NotNil(t, api)
	assert.Equal(t, "k8s", api.Type)
	assert.Equal(t, "prod-cluster", api.Properties["context"])
	assert.Equal(t, "production", api.Properties["namespace"])
	assert.Equal(t, "api-server", api.Properties["pod_grep"])
}

func TestAnsibleSource_Load(t *testing.T) {
	tmpDir := t.TempDir()
	inventoryPath := filepath.Join(tmpDir, "inventory.yml")

	content := `all:
  hosts:
    webserver:
      ansible_host: 192.168.1.10
      ansible_user: deploy
      ansible_port: 22
    dbserver:
      ansible_host: 192.168.1.20
      ansible_user: admin
      ansible_ssh_private_key_file: ~/.ssh/db_key
  children:
    production:
      hosts:
        prodweb:
          ansible_host: 10.0.0.100
          ansible_user: www-data
`

	err := os.WriteFile(inventoryPath, []byte(content), 0644)
	require.NoError(t, err)

	src := &AnsibleSource{}

	assert.True(t, src.CanLoad(inventoryPath))
	assert.Equal(t, "ansible", src.Name())
	assert.False(t, src.IsWritable())

	entries, err := src.Load(inventoryPath)
	require.NoError(t, err)
	assert.Len(t, entries, 3)

	// Find webserver entry
	var webserver *HostEntry
	for _, e := range entries {
		if e.Name == "webserver" {
			webserver = e
			break
		}
	}
	require.NotNil(t, webserver)
	assert.Equal(t, "ssh", webserver.Type)
	assert.Equal(t, "192.168.1.10", webserver.Properties["host"])
	assert.Equal(t, "deploy", webserver.Properties["user"])
}

func TestAnsibleSource_CanLoad(t *testing.T) {
	src := &AnsibleSource{}

	tests := []struct {
		path     string
		expected bool
	}{
		{"inventory.yml", true},
		{"inventory.yaml", true},
		{"hosts.yml", true},
		{"hosts.yaml", true},
		{"/path/to/inventory.yml", true},
		{"/path/to/hosts.ini", false},
		{"/path/to/config.yml", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			assert.Equal(t, tt.expected, src.CanLoad(tt.path))
		})
	}
}

func TestSSHConfigSource_Load(t *testing.T) {
	tmpDir := t.TempDir()
	sshDir := filepath.Join(tmpDir, ".ssh")
	err := os.MkdirAll(sshDir, 0755)
	require.NoError(t, err)

	configPath := filepath.Join(sshDir, "config")

	content := `Host myserver
    HostName 192.168.1.100
    User admin
    Port 2222
    IdentityFile ~/.ssh/mykey

Host jumpbox
    HostName jump.example.com
    User jump
    ForwardAgent yes

Host *
    ServerAliveInterval 60
`

	err = os.WriteFile(configPath, []byte(content), 0644)
	require.NoError(t, err)

	src := &SSHConfigSource{}

	assert.True(t, src.CanLoad(configPath))
	assert.Equal(t, "sshconfig", src.Name())
	assert.False(t, src.IsWritable())

	entries, err := src.Load(configPath)
	require.NoError(t, err)
	assert.Len(t, entries, 2) // * pattern should be skipped

	// Find myserver entry
	var myserver *HostEntry
	for _, e := range entries {
		if e.Name == "myserver" {
			myserver = e
			break
		}
	}
	require.NotNil(t, myserver)
	assert.Equal(t, "ssh", myserver.Type)
	assert.Equal(t, "192.168.1.100", myserver.Properties["host"])
	assert.Equal(t, "admin", myserver.Properties["user"])
	assert.Equal(t, "2222", myserver.Properties["port"])

	// Find jumpbox entry
	var jumpbox *HostEntry
	for _, e := range entries {
		if e.Name == "jumpbox" {
			jumpbox = e
			break
		}
	}
	require.NotNil(t, jumpbox)
	assert.Equal(t, "yes", jumpbox.Properties["agent_forward"])
}

func TestSSHConfigSource_CanLoad(t *testing.T) {
	src := &SSHConfigSource{}

	tests := []struct {
		path     string
		expected bool
	}{
		{"/home/user/.ssh/config", true},
		{"/etc/ssh/ssh_config", true},
		{"/etc/ssh/config", true},
		{"/path/to/config", false},
		{"/path/to/hosts.ini", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			assert.Equal(t, tt.expected, src.CanLoad(tt.path))
		})
	}
}

func TestRegistry(t *testing.T) {
	// All sources should be registered via init()
	sources := All()
	assert.Contains(t, sources, "ini")
	assert.Contains(t, sources, "ansible")
	assert.Contains(t, sources, "sshconfig")
	assert.Contains(t, sources, "vagrant")

	// Get should return registered sources
	ini, err := Get("ini")
	require.NoError(t, err)
	assert.Equal(t, "ini", ini.Name())

	// Get should return error for unknown source
	_, err = Get("unknown")
	assert.ErrorIs(t, err, ErrSourceNotFound)
}

func TestVagrantSource_CanLoad(t *testing.T) {
	src := &VagrantSource{}

	tests := []struct {
		path     string
		expected bool
	}{
		{"Vagrantfile", true},
		{"/path/to/Vagrantfile", true},
		{"/path/to/hosts.ini", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			assert.Equal(t, tt.expected, src.CanLoad(tt.path))
		})
	}
}

func TestParseVagrantSSHConfig(t *testing.T) {
	output := `Host default
  HostName 127.0.0.1
  User vagrant
  Port 2222
  IdentityFile /path/to/.vagrant/machines/default/virtualbox/private_key

Host web
  HostName 127.0.0.1
  User vagrant
  Port 2200
  IdentityFile /path/to/.vagrant/machines/web/virtualbox/private_key
`

	entries, err := parseVagrantSSHConfig(output, "/path/to/project")
	require.NoError(t, err)
	assert.Len(t, entries, 2)

	// Find default entry
	var defaultVM *HostEntry
	for _, e := range entries {
		if e.Name == "vagrant-default" {
			defaultVM = e
			break
		}
	}
	require.NotNil(t, defaultVM)
	assert.Equal(t, "ssh", defaultVM.Type)
	assert.Equal(t, "127.0.0.1", defaultVM.Properties["host"])
	assert.Equal(t, "vagrant", defaultVM.Properties["user"])
	assert.Equal(t, "2222", defaultVM.Properties["port"])
}
