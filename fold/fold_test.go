package fold

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestRun_IntegrationBuild(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test skipped in short mode")
	}

	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain not found")
	}

	manifest := filepath.Join(t.TempDir(), "origami.yaml")
	if err := os.WriteFile(manifest, []byte(`
name: test-tool
description: Integration test tool
version: "1.0"
imports:
  - origami.modules.rca
`), 0644); err != nil {
		t.Fatal(err)
	}

	output := filepath.Join(t.TempDir(), "test-tool")

	err := Run(Options{
		ManifestPath: manifest,
		Output:       output,
		Verbose:      true,
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(output); err != nil {
		t.Fatalf("output binary not found: %v", err)
	}
}
