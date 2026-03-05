package subprocess_test

import (
	"os/exec"
	"testing"

	"github.com/dpopsuev/origami/subprocess"
)

func requirePodman(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("podman"); err != nil {
		t.Skip("podman not available")
	}
}

func TestContainerManager_NewDefaults(t *testing.T) {
	cm := subprocess.NewContainerManager("")
	if cm.Runtime != "podman" {
		t.Errorf("default runtime = %q, want podman", cm.Runtime)
	}
}

func TestContainerManager_CustomRuntime(t *testing.T) {
	cm := subprocess.NewContainerManager("docker")
	if cm.Runtime != "docker" {
		t.Errorf("runtime = %q, want docker", cm.Runtime)
	}
}

func TestContainerManager_StartUnknownImage(t *testing.T) {
	requirePodman(t)

	cm := subprocess.NewContainerManager("podman")
	// Starting with a non-existent image should fail
	err := cm.Start(t.Context(), "test-nonexistent", "origami-nonexistent-image:latest", 19100)
	if err == nil {
		t.Fatal("expected error for non-existent image")
		// Clean up just in case
		cm.Stop(t.Context(), "test-nonexistent")
	}
}
