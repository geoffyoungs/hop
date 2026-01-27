package backend

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSSHBackend_Name(t *testing.T) {
	b := &SSHBackend{}
	assert.Equal(t, "ssh", b.Name())
}

func TestSSHBackend_Validate_RequiresHost(t *testing.T) {
	b := &SSHBackend{}

	tests := []struct {
		name       string
		properties map[string]string
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "missing host",
			properties: map[string]string{},
			wantErr:    true,
			errMsg:     "host is required",
		},
		{
			name:       "empty host",
			properties: map[string]string{"host": ""},
			wantErr:    true,
			errMsg:     "host is required",
		},
		{
			name:       "valid host only",
			properties: map[string]string{"host": "example.com"},
			wantErr:    false,
		},
		{
			name:       "valid with user",
			properties: map[string]string{"host": "example.com", "user": "admin"},
			wantErr:    false,
		},
		{
			name:       "valid with all options",
			properties: map[string]string{"host": "example.com", "user": "admin", "port": "2222", "identity": "~/.ssh/id_rsa"},
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host := &Host{Name: "test", Type: "ssh", Properties: tt.properties}
			err := b.Validate(host)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSSHBackend_buildSSHArgs(t *testing.T) {
	b := &SSHBackend{}

	tests := []struct {
		name       string
		properties map[string]string
		want       []string
	}{
		{
			name:       "no options",
			properties: map[string]string{"host": "example.com"},
			want:       nil,
		},
		{
			name:       "default port",
			properties: map[string]string{"host": "example.com", "port": "22"},
			want:       nil,
		},
		{
			name:       "custom port",
			properties: map[string]string{"host": "example.com", "port": "2222"},
			want:       []string{"-p", "2222"},
		},
		{
			name:       "identity file",
			properties: map[string]string{"host": "example.com", "identity": "~/.ssh/id_rsa"},
			want:       []string{"-i", "~/.ssh/id_rsa"},
		},
		{
			name:       "port and identity",
			properties: map[string]string{"host": "example.com", "port": "2222", "identity": "~/.ssh/id_rsa"},
			want:       []string{"-p", "2222", "-i", "~/.ssh/id_rsa"},
		},
		{
			name:       "jump host",
			properties: map[string]string{"host": "example.com", "jump": "bastion.example.com"},
			want:       []string{"-J", "bastion.example.com"},
		},
		{
			name:       "agent forwarding yes",
			properties: map[string]string{"host": "example.com", "agent_forward": "yes"},
			want:       []string{"-A"},
		},
		{
			name:       "agent forwarding true",
			properties: map[string]string{"host": "example.com", "agent_forward": "true"},
			want:       []string{"-A"},
		},
		{
			name:       "agent forwarding 1",
			properties: map[string]string{"host": "example.com", "agent_forward": "1"},
			want:       []string{"-A"},
		},
		{
			name:       "agent forwarding no",
			properties: map[string]string{"host": "example.com", "agent_forward": "no"},
			want:       nil,
		},
		{
			name:       "all options",
			properties: map[string]string{"host": "example.com", "port": "2222", "identity": "~/.ssh/id_rsa", "jump": "bastion", "agent_forward": "yes"},
			want:       []string{"-p", "2222", "-i", "~/.ssh/id_rsa", "-J", "bastion", "-A"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host := &Host{Name: "test", Type: "ssh", Properties: tt.properties}
			args := b.buildSSHArgs(host)
			assert.Equal(t, tt.want, args)
		})
	}
}

func TestSSHBackend_buildSCPArgs(t *testing.T) {
	b := &SSHBackend{}

	tests := []struct {
		name       string
		properties map[string]string
		want       []string
	}{
		{
			name:       "no options",
			properties: map[string]string{"host": "example.com"},
			want:       nil,
		},
		{
			name:       "default port",
			properties: map[string]string{"host": "example.com", "port": "22"},
			want:       nil,
		},
		{
			name:       "custom port uses uppercase P",
			properties: map[string]string{"host": "example.com", "port": "2222"},
			want:       []string{"-P", "2222"},
		},
		{
			name:       "identity file",
			properties: map[string]string{"host": "example.com", "identity": "~/.ssh/id_rsa"},
			want:       []string{"-i", "~/.ssh/id_rsa"},
		},
		{
			name:       "jump host",
			properties: map[string]string{"host": "example.com", "jump": "bastion.example.com"},
			want:       []string{"-J", "bastion.example.com"},
		},
		{
			name:       "all options",
			properties: map[string]string{"host": "example.com", "port": "2222", "identity": "~/.ssh/id_rsa", "jump": "bastion"},
			want:       []string{"-P", "2222", "-i", "~/.ssh/id_rsa", "-J", "bastion"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host := &Host{Name: "test", Type: "ssh", Properties: tt.properties}
			args := b.buildSCPArgs(host)
			assert.Equal(t, tt.want, args)
		})
	}
}

func TestSSHBackend_buildDestination(t *testing.T) {
	b := &SSHBackend{}

	tests := []struct {
		name       string
		properties map[string]string
		want       string
	}{
		{
			name:       "host only",
			properties: map[string]string{"host": "example.com"},
			want:       "example.com",
		},
		{
			name:       "host with user",
			properties: map[string]string{"host": "example.com", "user": "admin"},
			want:       "admin@example.com",
		},
		{
			name:       "ip address",
			properties: map[string]string{"host": "192.168.1.100"},
			want:       "192.168.1.100",
		},
		{
			name:       "ip with user",
			properties: map[string]string{"host": "192.168.1.100", "user": "root"},
			want:       "root@192.168.1.100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host := &Host{Name: "test", Type: "ssh", Properties: tt.properties}
			dest := b.buildDestination(host)
			assert.Equal(t, tt.want, dest)
		})
	}
}

func TestParseSSHVersion(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   string
	}{
		{
			name:   "standard openssh format",
			output: "OpenSSH_9.0p1, LibreSSL 3.3.6",
			want:   "9.0p1",
		},
		{
			name:   "openssh with underscore",
			output: "OpenSSH_8.9p1 Ubuntu-3ubuntu0.1, OpenSSL 3.0.2",
			want:   "8.9p1",
		},
		{
			name:   "unknown format",
			output: "some other ssh client",
			want:   "some other ssh client",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version := parseSSHVersion(tt.output)
			assert.Equal(t, tt.want, version)
		})
	}
}

func TestSSHBackend_BuildConnectCommand(t *testing.T) {
	b := &SSHBackend{}

	tests := []struct {
		name       string
		properties map[string]string
		wantCmd    string
		wantArgs   []string
	}{
		{
			name:       "basic host",
			properties: map[string]string{"host": "example.com"},
			wantCmd:    "ssh",
			wantArgs:   []string{"example.com"},
		},
		{
			name:       "host with user",
			properties: map[string]string{"host": "example.com", "user": "admin"},
			wantCmd:    "ssh",
			wantArgs:   []string{"admin@example.com"},
		},
		{
			name:       "with port and identity",
			properties: map[string]string{"host": "example.com", "user": "admin", "port": "2222", "identity": "~/.ssh/key"},
			wantCmd:    "ssh",
			wantArgs:   []string{"-p", "2222", "-i", "~/.ssh/key", "admin@example.com"},
		},
		{
			name:       "with jump and agent forward",
			properties: map[string]string{"host": "example.com", "jump": "bastion", "agent_forward": "yes"},
			wantCmd:    "ssh",
			wantArgs:   []string{"-J", "bastion", "-A", "example.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host := &Host{Name: "test", Type: "ssh", Properties: tt.properties}
			cmd, args, err := b.BuildConnectCommand(nil, host)
			require.NoError(t, err)
			assert.Equal(t, tt.wantCmd, cmd)
			assert.Equal(t, tt.wantArgs, args)
		})
	}
}

func TestDockerBackend_Name(t *testing.T) {
	b := &DockerBackend{}
	assert.Equal(t, "docker", b.Name())
}

func TestDockerBackend_Validate_RequiresContainer(t *testing.T) {
	b := &DockerBackend{}

	tests := []struct {
		name       string
		properties map[string]string
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "missing container",
			properties: map[string]string{},
			wantErr:    true,
			errMsg:     "one of container, label, image, or image_grep is required",
		},
		{
			name:       "empty container",
			properties: map[string]string{"container": ""},
			wantErr:    true,
			errMsg:     "one of container, label, image, or image_grep is required",
		},
		{
			name:       "valid container",
			properties: map[string]string{"container": "my_app"},
			wantErr:    false,
		},
		{
			name:       "valid with shell",
			properties: map[string]string{"container": "my_app", "shell": "/bin/bash"},
			wantErr:    false,
		},
		{
			name:       "valid label only",
			properties: map[string]string{"label": "app=myapp"},
			wantErr:    false,
		},
		{
			name:       "valid image only",
			properties: map[string]string{"image": "nginx:latest"},
			wantErr:    false,
		},
		{
			name:       "valid image_grep only",
			properties: map[string]string{"image_grep": "myapp"},
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host := &Host{Name: "test", Type: "docker", Properties: tt.properties}
			err := b.Validate(host)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestK8sBackend_Name(t *testing.T) {
	b := &K8sBackend{}
	assert.Equal(t, "k8s", b.Name())
}

func TestK8sBackend_Validate_RequiresPod(t *testing.T) {
	b := &K8sBackend{}

	tests := []struct {
		name       string
		properties map[string]string
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "missing pod selector or grep",
			properties: map[string]string{},
			wantErr:    true,
			errMsg:     "one of pod, selector, pod_grep, or deployment is required",
		},
		{
			name:       "empty pod selector and grep",
			properties: map[string]string{"pod": "", "selector": "", "pod_grep": ""},
			wantErr:    true,
			errMsg:     "one of pod, selector, pod_grep, or deployment is required",
		},
		{
			name:       "valid pod only",
			properties: map[string]string{"pod": "my-pod"},
			wantErr:    false,
		},
		{
			name:       "valid selector only",
			properties: map[string]string{"selector": "app=myapp"},
			wantErr:    false,
		},
		{
			name:       "valid pod_grep only",
			properties: map[string]string{"pod_grep": "myapp-scheduler"},
			wantErr:    false,
		},
		{
			name:       "valid deployment only",
			properties: map[string]string{"deployment": "my-deployment"},
			wantErr:    false,
		},
		{
			name:       "valid with namespace",
			properties: map[string]string{"pod": "my-pod", "namespace": "production"},
			wantErr:    false,
		},
		{
			name:       "valid with all options",
			properties: map[string]string{"pod": "my-pod", "namespace": "production", "container": "app", "context": "my-cluster"},
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host := &Host{Name: "test", Type: "k8s", Properties: tt.properties}
			err := b.Validate(host)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestK8sBackend_buildBaseArgs(t *testing.T) {
	b := &K8sBackend{}

	tests := []struct {
		name       string
		properties map[string]string
		want       []string
	}{
		{
			name:       "no options",
			properties: map[string]string{"pod": "my-pod"},
			want:       nil,
		},
		{
			name:       "with namespace",
			properties: map[string]string{"pod": "my-pod", "namespace": "production"},
			want:       []string{"-n", "production"},
		},
		{
			name:       "with context",
			properties: map[string]string{"pod": "my-pod", "context": "my-cluster"},
			want:       []string{"--context", "my-cluster"},
		},
		{
			name:       "with namespace and context",
			properties: map[string]string{"pod": "my-pod", "namespace": "production", "context": "my-cluster"},
			want:       []string{"-n", "production", "--context", "my-cluster"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host := &Host{Name: "test", Type: "k8s", Properties: tt.properties}
			args := b.buildBaseArgs(host)
			assert.Equal(t, tt.want, args)
		})
	}
}

func TestParseKubectlVersion(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   string
	}{
		{
			name:   "json format",
			output: `{"clientVersion":{"major":"1","minor":"28","gitVersion":"v1.28.0"}}`,
			want:   "1.28.0",
		},
		{
			name:   "short format",
			output: "Client Version: v1.28.0",
			want:   "1.28.0",
		},
		{
			name:   "unknown format",
			output: "kubectl version unknown",
			want:   "kubectl version unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version := parseKubectlVersion(tt.output)
			assert.Equal(t, tt.want, version)
		})
	}
}

func TestK8sBackend_firstPodFromOutput(t *testing.T) {
	b := &K8sBackend{}

	tests := []struct {
		name   string
		output string
		want   string
	}{
		{
			name:   "single pod",
			output: "pod/my-app-7d8f6c9b5-abc12\n",
			want:   "my-app-7d8f6c9b5-abc12",
		},
		{
			name:   "multiple pods returns first",
			output: "pod/my-app-7d8f6c9b5-abc12\npod/my-app-7d8f6c9b5-def34\n",
			want:   "my-app-7d8f6c9b5-abc12",
		},
		{
			name:   "empty output",
			output: "",
			want:   "",
		},
		{
			name:   "whitespace only",
			output: "   \n",
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := b.firstPodFromOutput([]byte(tt.output))
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestK8sBackend_findPodByPattern(t *testing.T) {
	b := &K8sBackend{}

	output := `pod/web-frontend-7d8f6c9b5-abc12
pod/web-kiosk-background-scheduler-8e9f7d6c5-def34
pod/web-prism-background-scheduler-9f0g8e7d6-ghi56
pod/redis-master-0
`

	tests := []struct {
		name    string
		pattern string
		want    string
	}{
		{
			name:    "match kiosk scheduler",
			pattern: "web-kiosk-background-scheduler",
			want:    "web-kiosk-background-scheduler-8e9f7d6c5-def34",
		},
		{
			name:    "match prism scheduler",
			pattern: "web-prism-background-scheduler",
			want:    "web-prism-background-scheduler-9f0g8e7d6-ghi56",
		},
		{
			name:    "match frontend",
			pattern: "web-frontend",
			want:    "web-frontend-7d8f6c9b5-abc12",
		},
		{
			name:    "partial match",
			pattern: "background-scheduler",
			want:    "web-kiosk-background-scheduler-8e9f7d6c5-def34",
		},
		{
			name:    "no match",
			pattern: "nonexistent",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := b.findPodByPattern([]byte(output), tt.pattern)
			assert.Equal(t, tt.want, result)
		})
	}
}
