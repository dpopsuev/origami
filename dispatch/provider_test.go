package dispatch

import (
	"strings"
	"testing"
)

var _ Dispatcher = (*ProviderRouter)(nil)

type mockDispatcher struct {
	name    string
	called  bool
	lastCtx DispatchContext
}

func (m *mockDispatcher) Dispatch(ctx DispatchContext) ([]byte, error) {
	m.called = true
	m.lastCtx = ctx
	return []byte(m.name + "-output"), nil
}

func TestProviderRouter_DefaultRoute(t *testing.T) {
	def := &mockDispatcher{name: "default"}
	router := NewProviderRouter(def, nil)

	result, err := router.Dispatch(DispatchContext{
		CaseID: "C1", Step: "F0",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !def.called {
		t.Error("default dispatcher not called")
	}
	if string(result) != "default-output" {
		t.Errorf("result = %q", result)
	}
}

func TestProviderRouter_NamedRoute(t *testing.T) {
	def := &mockDispatcher{name: "default"}
	codex := &mockDispatcher{name: "codex"}
	claude := &mockDispatcher{name: "claude"}

	router := NewProviderRouter(def, map[string]Dispatcher{
		"codex":  codex,
		"claude": claude,
	})

	result, err := router.Dispatch(DispatchContext{
		CaseID: "C1", Step: "F1", Provider: "codex",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !codex.called {
		t.Error("codex dispatcher not called")
	}
	if def.called {
		t.Error("default dispatcher should not be called")
	}
	if string(result) != "codex-output" {
		t.Errorf("result = %q", result)
	}
}

func TestProviderRouter_UnknownProvider(t *testing.T) {
	def := &mockDispatcher{name: "default"}
	router := NewProviderRouter(def, nil)

	_, err := router.Dispatch(DispatchContext{
		CaseID: "C1", Step: "F0", Provider: "nonexistent",
	})
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
	if !strings.Contains(err.Error(), "unknown provider") {
		t.Errorf("error = %v", err)
	}
}

func TestProviderRouter_Register(t *testing.T) {
	def := &mockDispatcher{name: "default"}
	router := NewProviderRouter(def, nil)

	openai := &mockDispatcher{name: "openai"}
	router.Register("openai", openai)

	result, err := router.Dispatch(DispatchContext{
		CaseID: "C1", Step: "F3", Provider: "openai",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !openai.called {
		t.Error("openai dispatcher not called")
	}
	if string(result) != "openai-output" {
		t.Errorf("result = %q", result)
	}
}

func TestProviderRouter_EmptyProviderUsesDefault(t *testing.T) {
	def := &mockDispatcher{name: "default"}
	codex := &mockDispatcher{name: "codex"}
	router := NewProviderRouter(def, map[string]Dispatcher{"codex": codex})

	_, err := router.Dispatch(DispatchContext{
		CaseID: "C1", Step: "F0", Provider: "",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !def.called {
		t.Error("default should be called for empty provider")
	}
	if codex.called {
		t.Error("codex should not be called for empty provider")
	}
}
