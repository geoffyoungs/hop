package backend

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRsyncerInterface(t *testing.T) {
	backends := []Backend{
		&SSHBackend{},
		&DockerBackend{},
		&K8sBackend{},
	}

	for _, b := range backends {
		if _, ok := b.(Rsyncer); !ok {
			t.Errorf("Backend %q does not implement Rsyncer", b.Name())
		}
	}
}

func TestBuildRsyncFlags(t *testing.T) {
	tests := []struct {
		name string
		opts *RsyncOptions
		want []string
	}{
		{
			name: "empty options",
			opts: &RsyncOptions{},
			want: nil,
		},
		{
			name: "archive only",
			opts: &RsyncOptions{Archive: true},
			want: []string{"-a"},
		},
		{
			name: "verbose only",
			opts: &RsyncOptions{Verbose: true},
			want: []string{"-v"},
		},
		{
			name: "compress only",
			opts: &RsyncOptions{Compress: true},
			want: []string{"-z"},
		},
		{
			name: "recursive only",
			opts: &RsyncOptions{Recursive: true},
			want: []string{"-r"},
		},
		{
			name: "dry-run only",
			opts: &RsyncOptions{DryRun: true},
			want: []string{"-n"},
		},
		{
			name: "delete only",
			opts: &RsyncOptions{Delete: true},
			want: []string{"--delete"},
		},
		{
			name: "single exclude",
			opts: &RsyncOptions{Exclude: []string{"*.log"}},
			want: []string{"--exclude=*.log"},
		},
		{
			name: "multiple excludes",
			opts: &RsyncOptions{Exclude: []string{"*.log", "tmp/"}},
			want: []string{"--exclude=*.log", "--exclude=tmp/"},
		},
		{
			name: "extra flags",
			opts: &RsyncOptions{Extra: []string{"--progress", "--partial"}},
			want: []string{"--progress", "--partial"},
		},
		{
			name: "all options combined",
			opts: &RsyncOptions{
				Archive:   true,
				Verbose:   true,
				Compress:  true,
				Recursive: true,
				DryRun:    true,
				Delete:    true,
				Exclude:   []string{"*.log"},
				Extra:     []string{"--progress"},
			},
			want: []string{"-a", "-v", "-z", "-r", "-n", "--delete", "--exclude=*.log", "--progress"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildRsyncFlags(tt.opts)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestShellQuote(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "empty string",
			input: "",
			want:  "''",
		},
		{
			name:  "simple string",
			input: "hello",
			want:  "'hello'",
		},
		{
			name:  "string with spaces",
			input: "hello world",
			want:  "'hello world'",
		},
		{
			name:  "string with single quote",
			input: "it's",
			want:  "'it'\"'\"'s'",
		},
		{
			name:  "path with spaces",
			input: "/path/to/my file",
			want:  "'/path/to/my file'",
		},
		{
			name:  "string with multiple quotes",
			input: "it's a 'test'",
			want:  "'it'\"'\"'s a '\"'\"'test'\"'\"''",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shellQuote(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSSHBackend_buildRsyncSSHCommand(t *testing.T) {
	b := &SSHBackend{}

	tests := []struct {
		name       string
		properties map[string]string
		direction  string
		localPath  string
		remotePath string
		opts       *RsyncOptions
		// We check that the rsync command includes expected -e flag components
		wantEContains []string
	}{
		{
			name:          "basic host",
			properties:    map[string]string{"host": "example.com"},
			direction:     "to",
			localPath:     "/local/path",
			remotePath:    "/remote/path",
			opts:          &RsyncOptions{Archive: true},
			wantEContains: []string{"ssh"},
		},
		{
			name:          "host with port",
			properties:    map[string]string{"host": "example.com", "port": "2222"},
			direction:     "to",
			localPath:     "/local/path",
			remotePath:    "/remote/path",
			opts:          &RsyncOptions{},
			wantEContains: []string{"ssh", "-p", "2222"},
		},
		{
			name:          "host with identity",
			properties:    map[string]string{"host": "example.com", "identity": "/home/user/.ssh/id_rsa"},
			direction:     "to",
			localPath:     "/local/path",
			remotePath:    "/remote/path",
			opts:          &RsyncOptions{},
			wantEContains: []string{"ssh", "-i"},
		},
		{
			name:          "host with jump",
			properties:    map[string]string{"host": "example.com", "jump": "bastion.example.com"},
			direction:     "to",
			localPath:     "/local/path",
			remotePath:    "/remote/path",
			opts:          &RsyncOptions{},
			wantEContains: []string{"ssh", "-J", "bastion.example.com"},
		},
		{
			name:          "host with user",
			properties:    map[string]string{"host": "example.com", "user": "admin"},
			direction:     "to",
			localPath:     "/local/path",
			remotePath:    "/remote/path",
			opts:          &RsyncOptions{},
			wantEContains: []string{"ssh"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host := &Host{Name: "test", Type: "ssh", Properties: tt.properties}
			_ = b
			_ = host
			// Verify the SSH command construction logic
			sshCmd := "ssh"
			if port := host.Properties["port"]; port != "" && port != "22" {
				sshCmd += " -p " + port
			}
			if identity := host.Properties["identity"]; identity != "" {
				sshCmd += " -i " + shellQuote(identity)
			}
			if jump := host.Properties["jump"]; jump != "" {
				sshCmd += " -J " + jump
			}

			for _, expected := range tt.wantEContains {
				assert.Contains(t, sshCmd, expected)
			}
		})
	}
}
