package framework

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestHookRegistry_RegisterAndGet(t *testing.T) {
	reg := HookRegistry{}
	called := false
	h := NewHookFunc("test-hook", func(_ context.Context, _ string, _ Artifact) error {
		called = true
		return nil
	})
	reg.Register(h)

	got, err := reg.Get("test-hook")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name() != "test-hook" {
		t.Errorf("Name() = %q", got.Name())
	}

	err = got.Run(context.Background(), "node", nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !called {
		t.Error("hook was not called")
	}
}

func TestHookRegistry_NotFound(t *testing.T) {
	reg := HookRegistry{}
	_, err := reg.Get("missing")
	if err == nil {
		t.Fatal("expected error for missing hook")
	}
}

func TestHookRegistry_Nil(t *testing.T) {
	var reg HookRegistry
	_, err := reg.Get("any")
	if err == nil {
		t.Fatal("expected error for nil registry")
	}
}

func TestHookRegistry_DuplicatePanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on duplicate")
		}
	}()
	reg := HookRegistry{}
	h := NewHookFunc("dup", func(_ context.Context, _ string, _ Artifact) error { return nil })
	reg.Register(h)
	reg.Register(h)
}

func TestHookingWalker_FiresHooks(t *testing.T) {
	var hookCalls []string
	hooks := HookRegistry{}
	hooks.Register(NewHookFunc("h1", func(_ context.Context, nodeName string, _ Artifact) error {
		hookCalls = append(hookCalls, "h1:"+nodeName)
		return nil
	}))
	hooks.Register(NewHookFunc("h2", func(_ context.Context, nodeName string, _ Artifact) error {
		hookCalls = append(hookCalls, "h2:"+nodeName)
		return nil
	}))

	trans := &echoTransformer{}
	def := &CircuitDef{
		Circuit: "test",
		Nodes: []NodeDef{
			{Name: "a", Element: "fire", Transformer: "echo", After: []string{"h1", "h2"}},
			{Name: "b", Element: "water", Transformer: "echo", After: []string{"h1"}},
		},
		Edges: []EdgeDef{
			{ID: "E1", From: "a", To: "b", When: "true"},
			{ID: "E2", From: "b", To: "_done", When: "true"},
		},
		Start: "a",
		Done:  "_done",
	}

	runner, err := NewRunnerWith(def, GraphRegistries{
		Transformers: TransformerRegistry{"echo": trans},
		Hooks:        hooks,
	})
	if err != nil {
		t.Fatalf("NewRunnerWith: %v", err)
	}

	err = runner.Walk(context.Background(), nil, "a")
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}

	expected := []string{"h1:a", "h2:a", "h1:b"}
	if len(hookCalls) != len(expected) {
		t.Fatalf("hook calls = %v, want %v", hookCalls, expected)
	}
	for i, exp := range expected {
		if hookCalls[i] != exp {
			t.Errorf("hookCalls[%d] = %q, want %q", i, hookCalls[i], exp)
		}
	}
}

func TestHookingWalker_MissingHookContinues(t *testing.T) {
	hooks := HookRegistry{}
	trans := &echoTransformer{}
	def := &CircuitDef{
		Circuit: "test",
		Nodes: []NodeDef{
			{Name: "a", Element: "fire", Transformer: "echo", After: []string{"nonexistent"}},
		},
		Edges: []EdgeDef{
			{ID: "E1", From: "a", To: "_done", When: "true"},
		},
		Start: "a",
		Done:  "_done",
	}

	runner, err := NewRunnerWith(def, GraphRegistries{
		Transformers: TransformerRegistry{"echo": trans},
		Hooks:        hooks,
	})
	if err != nil {
		t.Fatalf("NewRunnerWith: %v", err)
	}

	err = runner.Walk(context.Background(), nil, "a")
	if err != nil {
		t.Fatalf("Walk should succeed even with missing hook: %v", err)
	}
}

func TestFileWriteHook_WritesArtifact(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "recall.json")

	def := &CircuitDef{
		Circuit: "test",
		Nodes: []NodeDef{
			{
				Name:        "recall",
				Element:     "earth",
				Transformer: "go-template",
				Prompt:      "test data",
				After:       []string{"file-write"},
				Meta:        map[string]any{"output_path": outPath},
			},
		},
		Edges: []EdgeDef{
			{ID: "E1", Name: "done", From: "recall", To: "_done", When: "true"},
		},
		Start: "recall",
		Done:  "_done",
	}

	runner, err := NewRunnerWith(def, GraphRegistries{})
	if err != nil {
		t.Fatalf("NewRunnerWith: %v", err)
	}

	err = runner.Walk(context.Background(), nil, "recall")
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	var result any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if result != "test data" {
		t.Errorf("file content = %v, want %q", result, "test data")
	}
}

func TestFileWriteHook_TemplatedPath(t *testing.T) {
	dir := t.TempDir()
	pathTmpl := filepath.Join(dir, "{{ .NodeName }}.json")

	def := &CircuitDef{
		Circuit: "test",
		Nodes: []NodeDef{
			{
				Name:        "triage",
				Element:     "fire",
				Transformer: "go-template",
				Prompt:      "triage output",
				After:       []string{"file-write"},
				Meta:        map[string]any{"output_path": pathTmpl},
			},
		},
		Edges: []EdgeDef{
			{ID: "E1", Name: "done", From: "triage", To: "_done", When: "true"},
		},
		Start: "triage",
		Done:  "_done",
	}

	runner, err := NewRunnerWith(def, GraphRegistries{})
	if err != nil {
		t.Fatalf("NewRunnerWith: %v", err)
	}

	err = runner.Walk(context.Background(), nil, "triage")
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}

	expectedPath := filepath.Join(dir, "triage.json")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Fatalf("expected file at %s", expectedPath)
	}
}

func TestFileWriteHook_MissingOutputPath(t *testing.T) {
	hook := &FileWriteHook{
		nodeMeta: map[string]map[string]any{
			"node": {},
		},
	}

	art := &transformerArtifact{typeName: "test", raw: "data"}
	err := hook.Run(context.Background(), "node", art)
	if err == nil {
		t.Fatal("expected error for missing output_path")
	}
}

func TestFileWriteHook_AutoRegisteredByRunner(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "auto.json")

	def := &CircuitDef{
		Circuit: "test",
		Nodes: []NodeDef{
			{
				Name:        "a",
				Element:     "fire",
				Transformer: "passthrough",
				After:       []string{"file-write"},
				Meta:        map[string]any{"output_path": outPath},
			},
		},
		Edges: []EdgeDef{
			{ID: "E1", Name: "done", From: "a", To: "_done", When: "true"},
		},
		Start: "a",
		Done:  "_done",
	}

	runner, err := NewRunnerWith(def, GraphRegistries{})
	if err != nil {
		t.Fatalf("runner should auto-register file-write hook: %v", err)
	}

	if runner.Hooks == nil {
		t.Fatal("hooks should not be nil")
	}
	_, err = runner.Hooks.Get("file-write")
	if err != nil {
		t.Fatalf("file-write hook should be registered: %v", err)
	}
}

func TestHookingWalker_NoHooksNoWrap(t *testing.T) {
	trans := &echoTransformer{}
	def := &CircuitDef{
		Circuit: "test",
		Nodes:    []NodeDef{{Name: "a", Element: "fire", Transformer: "echo"}},
		Edges:    []EdgeDef{{ID: "E1", From: "a", To: "_done", When: "true"}},
		Start:    "a",
		Done:     "_done",
	}

	runner, err := NewRunnerWith(def, GraphRegistries{
		Transformers: TransformerRegistry{"echo": trans},
	})
	if err != nil {
		t.Fatalf("NewRunnerWith: %v", err)
	}

	err = runner.Walk(context.Background(), nil, "a")
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
}
