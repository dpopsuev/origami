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

func TestRun_InputResolutionAndPromptRendering(t *testing.T) {
	yaml := `
pipeline: test-input-resolve
vars:
  threshold: 0.85
nodes:
  - name: recall
    element: fire
    transformer: echo
  - name: triage
    element: water
    transformer: capture
    input: "${recall.output}"
    prompt: "Node {{.Node}} sees threshold {{.Config.threshold}}"
edges:
  - id: E1
    from: recall
    to: triage
    when: "true"
  - id: E2
    from: triage
    to: _done
    when: "true"
start: recall
done: _done
`
	path := writeTempPipeline(t, yaml)

	var capturedPrompt string
	var capturedInput any

	capture := TransformerFunc("capture", func(_ context.Context, tc *TransformerContext) (any, error) {
		capturedPrompt = tc.Prompt
		capturedInput = tc.Input
		return map[string]any{"captured": true}, nil
	})

	err := Run(context.Background(), path, nil,
		WithTransformers(TransformerRegistry{
			"echo":    &echoTransformer{},
			"capture": capture,
		}),
	)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if capturedPrompt != "Node triage sees threshold 0.85" {
		t.Errorf("rendered prompt = %q", capturedPrompt)
	}

	inputMap, ok := capturedInput.(map[string]any)
	if !ok {
		t.Fatalf("input type = %T, want map from recall echo", capturedInput)
	}
	if inputMap["node"] != "recall" {
		t.Errorf("input should come from recall node, got node=%v", inputMap["node"])
	}
}

func TestRun_WithTeam_TwoWalkers(t *testing.T) {
	yaml := `
pipeline: test-team
nodes:
  - name: classify
    element: fire
    transformer: echo
  - name: investigate
    element: water
    transformer: echo
edges:
  - id: E1
    from: classify
    to: investigate
    when: "true"
  - id: E2
    from: investigate
    to: _done
    when: "true"
start: classify
done: _done
`
	path := writeTempPipeline(t, yaml)

	herald := &stubWalker{
		identity: AgentIdentity{
			PersonaName:  "Herald",
			Element:      ElementFire,
			StepAffinity: map[string]float64{"classify": 0.9, "investigate": 0.1},
		},
		state: NewWalkerState("herald-1"),
	}
	seeker := &stubWalker{
		identity: AgentIdentity{
			PersonaName:  "Seeker",
			Element:      ElementWater,
			StepAffinity: map[string]float64{"classify": 0.1, "investigate": 0.9},
		},
		state: NewWalkerState("seeker-1"),
	}

	team := &Team{
		Walkers:   []Walker{herald, seeker},
		Scheduler: &AffinityScheduler{},
		MaxSteps:  20,
	}

	err := Run(context.Background(), path, nil,
		WithTransformers(TransformerRegistry{"echo": &echoTransformer{}}),
		WithTeam(team),
	)
	if err != nil {
		t.Fatalf("Run with team: %v", err)
	}

	if len(herald.visited) == 0 && len(seeker.visited) == 0 {
		t.Fatal("neither walker visited any nodes")
	}

	allVisited := append(herald.visited, seeker.visited...)
	hasClassify, hasInvestigate := false, false
	for _, v := range allVisited {
		if v == "classify" {
			hasClassify = true
		}
		if v == "investigate" {
			hasInvestigate = true
		}
	}
	if !hasClassify || !hasInvestigate {
		t.Errorf("both nodes should be visited: classify=%v investigate=%v (herald=%v seeker=%v)",
			hasClassify, hasInvestigate, herald.visited, seeker.visited)
	}
}

func TestRun_WithTeam_InputPropagated(t *testing.T) {
	path := writeTempPipeline(t, testPipelineYAML)

	w := &stubWalker{
		identity: AgentIdentity{PersonaName: "Solo"},
		state:    NewWalkerState("solo-1"),
	}

	team := &Team{
		Walkers:   []Walker{w},
		Scheduler: &SingleScheduler{Walker: w},
	}

	err := Run(context.Background(), path, map[string]any{"hello": "world"},
		WithTransformers(TransformerRegistry{"echo": &echoTransformer{}}),
		WithTeam(team),
	)
	if err != nil {
		t.Fatalf("Run with team + input: %v", err)
	}

	got, ok := w.State().Context["input"]
	if !ok {
		t.Fatal("expected input in walker context")
	}
	m, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("input type = %T, want map[string]any", got)
	}
	if m["hello"] != "world" {
		t.Errorf("input = %v, want {hello: world}", m)
	}
}

func TestValidate_MissingTransformer(t *testing.T) {
	path := writeTempPipeline(t, testPipelineYAML)
	err := Validate(path, WithTransformers(TransformerRegistry{}))
	if err == nil {
		t.Fatal("expected error for missing transformer when registry is provided but empty")
	}
}

func TestValidate_NoRegistries_StructuralOnly(t *testing.T) {
	path := writeTempPipeline(t, testPipelineYAML)
	err := Validate(path)
	if err != nil {
		t.Fatalf("structural validation without registries should pass: %v", err)
	}
}
