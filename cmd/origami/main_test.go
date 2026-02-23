package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func buildBinary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "origami")
	cmd := exec.Command("go", "build", "-o", bin, ".")
	cmd.Dir = filepath.Join(getModuleRoot(t), "cmd", "origami")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build origami binary: %v\n%s", err, out)
	}
	return bin
}

func getModuleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for dir != "/" {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		dir = filepath.Dir(dir)
	}
	t.Fatal("could not find go.mod")
	return ""
}

const integrationPipeline = `
pipeline: cli-integration
vars:
  greeting: hello
nodes:
  - name: start
    element: fire
    transformer: echo
  - name: finish
    element: water
    transformer: echo
edges:
  - id: E1
    name: go
    from: start
    to: finish
    when: "true"
  - id: E2
    name: done
    from: finish
    to: _done
    when: "true"
start: start
done: _done
`

func TestCLI_Validate(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	pipelinePath := filepath.Join(dir, "pipeline.yaml")
	if err := os.WriteFile(pipelinePath, []byte(integrationPipeline), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(bin, "validate", pipelinePath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("origami validate failed: %v\n%s", err, out)
	}
	if len(out) == 0 {
		t.Error("expected output from validate")
	}
}

func TestCLI_Validate_Invalid(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	pipelinePath := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(pipelinePath, []byte("pipeline: bad\nnodes: []\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(bin, "validate", pipelinePath)
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected validation to fail for invalid pipeline")
	}
}

func TestCLI_Version(t *testing.T) {
	bin := buildBinary(t)
	cmd := exec.Command(bin, "version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("origami version failed: %v\n%s", err, out)
	}
	if len(out) == 0 {
		t.Error("expected version output")
	}
}

func TestCLI_UnknownCommand(t *testing.T) {
	bin := buildBinary(t)
	cmd := exec.Command(bin, "nonexistent")
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected error for unknown command")
	}
}

func TestCLI_Run(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	dataPath := filepath.Join(dir, "data.json")
	if err := os.WriteFile(dataPath, []byte(`{"result":"hello"}`), 0644); err != nil {
		t.Fatal(err)
	}

	pipelineYAML := `
pipeline: cli-run-integration
vars:
  mode: fast
nodes:
  - name: load
    element: fire
    transformer: file
    prompt: data.json
  - name: classify
    element: water
    transformer: file
    input: "${load.output}"
    prompt: data.json
edges:
  - id: E1
    name: load-to-classify
    from: load
    to: classify
    when: "true"
  - id: E2
    name: done
    from: classify
    to: _done
    when: "true"
start: load
done: _done
`
	pipelinePath := filepath.Join(dir, "pipeline.yaml")
	if err := os.WriteFile(pipelinePath, []byte(pipelineYAML), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(bin, "run", pipelinePath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("origami run failed: %v\n%s", err, out)
	}
}
