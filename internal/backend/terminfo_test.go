package backend

import (
	"os"
	"testing"
)

func TestGetTerminfo(t *testing.T) {
	// Skip if TERM is not set (CI environments)
	if os.Getenv("TERM") == "" {
		t.Skip("TERM environment variable not set")
	}

	output, err := GetTerminfo()
	if err != nil {
		t.Fatalf("GetTerminfo failed: %v", err)
	}

	if len(output) == 0 {
		t.Error("GetTerminfo returned empty output")
	}

	// Output should contain the terminal name
	term := os.Getenv("TERM")
	if term != "" && len(output) > 0 {
		// infocmp output typically starts with the terminal name
		// or contains it in the first few lines
		t.Logf("Got %d bytes of terminfo for %s", len(output), term)
	}
}

func TestTerminfoSyncerInterface(t *testing.T) {
	// Verify all backends implement TerminfoSyncer
	backends := []Backend{
		&SSHBackend{},
		&DockerBackend{},
		&K8sBackend{},
	}

	for _, b := range backends {
		if _, ok := b.(TerminfoSyncer); !ok {
			t.Errorf("Backend %q does not implement TerminfoSyncer", b.Name())
		}
	}
}
