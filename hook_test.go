package framework

import (
	"context"
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
	def := &PipelineDef{
		Pipeline: "test",
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
	def := &PipelineDef{
		Pipeline: "test",
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

func TestHookingWalker_NoHooksNoWrap(t *testing.T) {
	trans := &echoTransformer{}
	def := &PipelineDef{
		Pipeline: "test",
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
