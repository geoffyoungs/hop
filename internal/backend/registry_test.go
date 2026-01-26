package backend

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegister(t *testing.T) {
	// Note: backends are already registered via init() functions,
	// so we test that the registry works correctly with them

	// Test that Get returns a registered backend
	ssh, err := Get("ssh")
	require.NoError(t, err)
	assert.NotNil(t, ssh)
	assert.Equal(t, "ssh", ssh.Name())
}

func TestGet(t *testing.T) {
	tests := []struct {
		name    string
		backend string
		wantErr error
	}{
		{
			name:    "get ssh backend",
			backend: "ssh",
			wantErr: nil,
		},
		{
			name:    "get docker backend",
			backend: "docker",
			wantErr: nil,
		},
		{
			name:    "get k8s backend",
			backend: "k8s",
			wantErr: nil,
		},
		{
			name:    "get nonexistent backend",
			backend: "nonexistent",
			wantErr: ErrBackendNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := Get(tt.backend)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, b)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, b)
				assert.Equal(t, tt.backend, b.Name())
			}
		})
	}
}

func TestAll(t *testing.T) {
	backends := All()

	// Should have at least the 3 built-in backends
	assert.GreaterOrEqual(t, len(backends), 3)

	// Verify the expected backends are present
	expectedBackends := []string{"ssh", "docker", "k8s"}
	for _, name := range expectedBackends {
		b, ok := backends[name]
		assert.True(t, ok, "backend %s should be registered", name)
		assert.NotNil(t, b)
		assert.Equal(t, name, b.Name())
	}

	// Verify All returns a copy (modifying it doesn't affect the registry)
	delete(backends, "ssh")
	ssh, err := Get("ssh")
	require.NoError(t, err)
	assert.NotNil(t, ssh, "deleting from All() result should not affect registry")
}

func TestErrBackendNotFound(t *testing.T) {
	assert.Equal(t, "backend not found", ErrBackendNotFound.Error())
}

func TestErrNotSupported(t *testing.T) {
	assert.Equal(t, "operation not supported by this backend", ErrNotSupported.Error())
}
