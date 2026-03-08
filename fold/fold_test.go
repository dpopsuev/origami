package fold

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestRun_IntegrationBuild_DomainServe(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test skipped in short mode")
	}

	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain not found")
	}

	tmpDir := t.TempDir()

	circuitDir := filepath.Join(tmpDir, "internal", "circuits")
	if err := os.MkdirAll(circuitDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(circuitDir, "test.yaml"), []byte("topology: cascade\ndescription: test circuit\n"), 0644); err != nil {
		t.Fatal(err)
	}

	manifest := filepath.Join(tmpDir, "origami.yaml")
	if err := os.WriteFile(manifest, []byte(`
name: test-domain
version: "0.1"
domain_serve:
  port: 9300
  embed: internal/
`), 0644); err != nil {
		t.Fatal(err)
	}

	output := filepath.Join(t.TempDir(), "test-domain")

	err := Run(Options{
		ManifestPath: manifest,
		Output:       output,
		Verbose:      true,
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(output); err != nil {
		t.Fatalf("domain-serve binary not found: %v", err)
	}
}

func TestRun_IntegrationBuild_Assets(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test skipped in short mode")
	}

	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain not found")
	}

	tmpDir := t.TempDir()

	writeFile := func(rel, content string) {
		t.Helper()
		p := filepath.Join(tmpDir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	writeFile("circuits/rca.yaml", "topology: cascade\ndescription: RCA circuit\n")
	writeFile("prompts/recall.md", "You are a recall judge.")
	writeFile("vocabulary.yaml", "defects:\n  pb001: product bug\n")

	manifest := filepath.Join(tmpDir, "origami.yaml")
	if err := os.WriteFile(manifest, []byte(`
name: test-assets
version: "0.1"
domain_serve:
  port: 9300
  assets:
    circuits:
      rca: circuits/rca.yaml
    prompts:
      recall: prompts/recall.md
    files:
      vocabulary: vocabulary.yaml
`), 0644); err != nil {
		t.Fatal(err)
	}

	output := filepath.Join(t.TempDir(), "test-assets")

	err := Run(Options{
		ManifestPath: manifest,
		Output:       output,
		Verbose:      true,
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(output); err != nil {
		t.Fatalf("domain-serve binary not found: %v", err)
	}
}

func TestRun_MissingDomainServe(t *testing.T) {
	manifest := filepath.Join(t.TempDir(), "origami.yaml")
	if err := os.WriteFile(manifest, []byte(`
name: test-no-serve
version: "1.0"
`), 0644); err != nil {
		t.Fatal(err)
	}

	err := Run(Options{ManifestPath: manifest})
	if err == nil {
		t.Fatal("expected error for manifest without domain_serve")
	}
}
