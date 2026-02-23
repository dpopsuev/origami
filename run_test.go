package framework

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

const testPipelineYAML = `
pipeline: test-run
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

func writeTempPipeline(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "pipeline.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRun_BasicPipeline(t *testing.T) {
	path := writeTempPipeline(t, testPipelineYAML)
	trans := &echoTransformer{}

	err := Run(context.Background(), path, map[string]any{"data": true},
		WithTransformers(TransformerRegistry{"echo": trans}),
	)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
}

func TestRun_WithOverrides(t *testing.T) {
	yaml := `
pipeline: test-vars
vars:
  threshold: 0.5
nodes:
  - name: a
    element: fire
    transformer: echo
edges:
  - id: E1
    from: a
    to: _done
    when: "config.threshold > 0.8"
start: a
done: _done
`
	path := writeTempPipeline(t, yaml)
	trans := &echoTransformer{}

	err := Run(context.Background(), path, nil,
		WithTransformers(TransformerRegistry{"echo": trans}),
		WithOverrides(map[string]any{"threshold": 0.9}),
	)
	if err != nil {
		t.Fatalf("Run with overrides: %v", err)
	}
}

func TestRun_WithHooks(t *testing.T) {
	yaml := `
pipeline: test-hooks
nodes:
  - name: a
    element: fire
    transformer: echo
    after: [my-hook]
edges:
  - id: E1
    from: a
    to: _done
    when: "true"
start: a
done: _done
`
	path := writeTempPipeline(t, yaml)
	trans := &echoTransformer{}
	called := false
	hooks := HookRegistry{}
	hooks.Register(NewHookFunc("my-hook", func(_ context.Context, _ string, _ Artifact) error {
		called = true
		return nil
	}))

	err := Run(context.Background(), path, nil,
		WithTransformers(TransformerRegistry{"echo": trans}),
		WithHooks(hooks),
	)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !called {
		t.Error("hook was not called")
	}
}

func TestRun_MissingFile(t *testing.T) {
	err := Run(context.Background(), "/nonexistent/pipeline.yaml", nil)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestRun_InvalidYAML(t *testing.T) {
	path := writeTempPipeline(t, "{{invalid yaml")
	err := Run(context.Background(), path, nil)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestValidate_ValidPipeline(t *testing.T) {
	path := writeTempPipeline(t, testPipelineYAML)
	err := Validate(path,
		WithTransformers(TransformerRegistry{"echo": &echoTransformer{}}),
	)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

func TestValidate_InvalidExpression(t *testing.T) {
	yaml := `
pipeline: bad
nodes:
  - name: a
    element: fire
    transformer: echo
edges:
  - id: E1
    from: a
    to: _done
    when: ">>> invalid"
start: a
done: _done
`
	path := writeTempPipeline(t, yaml)
	err := Validate(path, WithTransformers(TransformerRegistry{"echo": &echoTransformer{}}))
	if err == nil {
		t.Fatal("expected validation error for invalid expression")
	}
}

func TestValidate_MissingTransformer(t *testing.T) {
	path := writeTempPipeline(t, testPipelineYAML)
	err := Validate(path)
	if err == nil {
		t.Fatal("expected error for missing transformer")
	}
}
