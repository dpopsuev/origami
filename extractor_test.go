package framework

import (
	"context"
	"fmt"
	"testing"
)

// stubExtractor is a minimal Extractor for testing.
type stubExtractor struct {
	name string
	fn   func(ctx context.Context, input any) (any, error)
}

func (s *stubExtractor) Name() string {
	return s.name
}

func (s *stubExtractor) Extract(ctx context.Context, input any) (any, error) {
	return s.fn(ctx, input)
}

func TestExtractorRegistry_RegisterAndGet(t *testing.T) {
	reg := make(ExtractorRegistry)
	ext := &stubExtractor{name: "test-ext", fn: func(_ context.Context, in any) (any, error) {
		return in, nil
	}}

	reg.Register(ext)

	got, err := reg.Get("test-ext")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name() != "test-ext" {
		t.Errorf("Name() = %q, want %q", got.Name(), "test-ext")
	}
}

func TestExtractorRegistry_GetUnknown(t *testing.T) {
	reg := make(ExtractorRegistry)
	_, err := reg.Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown extractor")
	}
}

func TestExtractorRegistry_DuplicatePanics(t *testing.T) {
	reg := make(ExtractorRegistry)
	ext := &stubExtractor{name: "dup"}
	reg.Register(ext)

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on duplicate registration")
		}
	}()
	reg.Register(ext)
}

func TestExtractorNode_Process(t *testing.T) {
	called := false
	ext := &stubExtractor{
		name: "echo",
		fn: func(_ context.Context, in any) (any, error) {
			called = true
			return fmt.Sprintf("echoed: %v", in), nil
		},
	}

	node := &extractorNode{name: "parse", element: "earth", ext: ext}

	if node.Name() != "parse" {
		t.Errorf("Name() = %q, want %q", node.Name(), "parse")
	}
	if node.ElementAffinity() != "earth" {
		t.Errorf("ElementAffinity() = %q, want %q", node.ElementAffinity(), "earth")
	}

	prior := &extractorArtifact{typeName: "raw", raw: "hello"}
	nc := NodeContext{PriorArtifact: prior}

	art, err := node.Process(context.Background(), nc)
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if !called {
		t.Fatal("extractor was not called")
	}
	if art.Type() != "echo" {
		t.Errorf("Type() = %q, want %q", art.Type(), "echo")
	}
	if art.Raw() != "echoed: hello" {
		t.Errorf("Raw() = %v, want %q", art.Raw(), "echoed: hello")
	}
}

func TestExtractorNode_NilPriorArtifact(t *testing.T) {
	ext := &stubExtractor{
		name: "null-safe",
		fn: func(_ context.Context, in any) (any, error) {
			if in != nil {
				t.Errorf("expected nil input, got %v", in)
			}
			return "ok", nil
		},
	}
	node := &extractorNode{name: "n", ext: ext}
	nc := NodeContext{PriorArtifact: nil}
	_, err := node.Process(context.Background(), nc)
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
}

func TestExtractorNode_ExtractError(t *testing.T) {
	ext := &stubExtractor{
		name: "fail",
		fn: func(_ context.Context, _ any) (any, error) {
			return nil, fmt.Errorf("parse failed")
		},
	}
	node := &extractorNode{name: "n", ext: ext}
	_, err := node.Process(context.Background(), NodeContext{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestExtractorArtifact(t *testing.T) {
	a := &extractorArtifact{typeName: "json", confidence: 0.95, raw: map[string]string{"k": "v"}}
	if a.Type() != "json" {
		t.Errorf("Type() = %q, want %q", a.Type(), "json")
	}
	if a.Confidence() != 0.95 {
		t.Errorf("Confidence() = %f, want 0.95", a.Confidence())
	}
	m, ok := a.Raw().(map[string]string)
	if !ok {
		t.Fatalf("Raw() type = %T, want map[string]string", a.Raw())
	}
	if m["k"] != "v" {
		t.Errorf("Raw()[k] = %q, want %q", m["k"], "v")
	}
}

func TestBuildGraph_WithExtractorNode(t *testing.T) {
	ext := &stubExtractor{
		name: "my-ext",
		fn: func(_ context.Context, in any) (any, error) {
			return "extracted", nil
		},
	}
	extReg := make(ExtractorRegistry)
	extReg.Register(ext)

	nodeReg := NodeRegistry{
		"finish": func(d NodeDef) Node {
			return &extTestNode{name: d.Name}
		},
	}

	data := []byte(`
pipeline: ext-test
nodes:
  - name: parse
    element: earth
    extractor: my-ext
  - name: done_node
    family: finish
edges:
  - id: E1
    name: parse-to-done
    from: parse
    to: done_node
  - id: E2
    name: to-end
    from: done_node
    to: _done
start: parse
done: _done
`)
	def, err := LoadPipeline(data)
	if err != nil {
		t.Fatalf("LoadPipeline: %v", err)
	}

	g, err := def.BuildGraph(nodeReg, nil, extReg)
	if err != nil {
		t.Fatalf("BuildGraph: %v", err)
	}

	n, ok := g.NodeByName("parse")
	if !ok {
		t.Fatal("node 'parse' not found")
	}
	en, ok := n.(*extractorNode)
	if !ok {
		t.Fatalf("node type = %T, want *extractorNode", n)
	}
	if en.ext.Name() != "my-ext" {
		t.Errorf("extractor = %q, want %q", en.ext.Name(), "my-ext")
	}
}

func TestBuildGraph_ExtractorNotRegistered(t *testing.T) {
	extReg := make(ExtractorRegistry)
	nodeReg := NodeRegistry{}

	data := []byte(`
pipeline: fail-test
nodes:
  - name: parse
    extractor: missing
edges:
  - id: E1
    name: parse-done
    from: parse
    to: _done
start: parse
done: _done
`)
	def, err := LoadPipeline(data)
	if err != nil {
		t.Fatalf("LoadPipeline: %v", err)
	}

	_, err = def.BuildGraph(nodeReg, nil, extReg)
	if err == nil {
		t.Fatal("expected error for unregistered extractor")
	}
}

func TestLoadPipeline_ExtractorField_RoundTrip(t *testing.T) {
	original := &PipelineDef{
		Pipeline: "ext-roundtrip",
		Nodes: []NodeDef{
			{Name: "parse", Element: "earth", Extractor: "json-v1"},
			{Name: "process", Family: "compute"},
		},
		Edges: []EdgeDef{
			{ID: "E1", Name: "parse-process", From: "parse", To: "process"},
			{ID: "E2", Name: "process-done", From: "process", To: "_done"},
		},
		Start: "parse",
		Done:  "_done",
	}

	data, err := original.MarshalYAML()
	if err != nil {
		t.Fatalf("MarshalYAML: %v", err)
	}

	restored, err := LoadPipeline(data)
	if err != nil {
		t.Fatalf("LoadPipeline: %v", err)
	}

	if restored.Nodes[0].Extractor != "json-v1" {
		t.Errorf("Nodes[0].Extractor = %q, want %q", restored.Nodes[0].Extractor, "json-v1")
	}
	if restored.Nodes[1].Extractor != "" {
		t.Errorf("Nodes[1].Extractor = %q, want empty", restored.Nodes[1].Extractor)
	}
}

// extTestNode is a minimal Node for extractor DSL tests.
type extTestNode struct {
	name string
}

func (n *extTestNode) Name() string            { return n.name }
func (n *extTestNode) ElementAffinity() Element { return "" }
func (n *extTestNode) Process(_ context.Context, _ NodeContext) (Artifact, error) {
	return nil, nil
}
