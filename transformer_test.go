package framework

import (
	"context"
	"testing"
)

type echoTransformer struct{}

func (t *echoTransformer) Name() string { return "echo" }
func (t *echoTransformer) Transform(_ context.Context, tc *TransformerContext) (any, error) {
	return map[string]any{"echoed": tc.Input, "node": tc.NodeName}, nil
}

func TestTransformerNode_Process(t *testing.T) {
	trans := &echoTransformer{}
	node := &transformerNode{
		name:    "test-node",
		element: ElementFire,
		trans:   trans,
		config:  map[string]any{"key": "val"},
	}

	if node.Name() != "test-node" {
		t.Errorf("Name() = %q", node.Name())
	}
	if node.ElementAffinity() != ElementFire {
		t.Errorf("Element = %q", node.ElementAffinity())
	}

	nc := NodeContext{
		PriorArtifact: &testArtifact{raw: map[string]any{"data": true}},
	}
	artifact, err := node.Process(context.Background(), nc)
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	m, ok := artifact.Raw().(map[string]any)
	if !ok {
		t.Fatalf("Raw() type = %T", artifact.Raw())
	}
	if m["node"] != "test-node" {
		t.Errorf("node = %v", m["node"])
	}
}

func TestTransformerNode_NilInput(t *testing.T) {
	trans := &echoTransformer{}
	node := &transformerNode{name: "test", trans: trans}

	artifact, err := node.Process(context.Background(), NodeContext{})
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	m := artifact.Raw().(map[string]any)
	if m["echoed"] != nil {
		t.Errorf("expected nil echoed, got %v", m["echoed"])
	}
}

func TestBuildGraphWith_TransformerNode(t *testing.T) {
	trans := &echoTransformer{}
	def := &PipelineDef{
		Pipeline: "test",
		Nodes: []NodeDef{
			{Name: "a", Element: "fire", Transformer: "echo"},
			{Name: "b", Element: "water", Transformer: "echo"},
		},
		Edges: []EdgeDef{
			{ID: "E1", Name: "a-to-b", From: "a", To: "b", When: "true"},
			{ID: "E2", Name: "b-to-done", From: "b", To: "_done", When: "true"},
		},
		Start: "a",
		Done:  "_done",
	}

	reg := GraphRegistries{
		Transformers: TransformerRegistry{"echo": trans},
	}

	graph, err := def.BuildGraph(reg)
	if err != nil {
		t.Fatalf("BuildGraphWith: %v", err)
	}

	nodes := graph.Nodes()
	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(nodes))
	}
	for _, n := range nodes {
		if !IsTransformerNode(n) {
			t.Errorf("node %q should be a transformer node", n.Name())
		}
	}
}

func TestBuildGraphWith_MixedTransformerAndWalker(t *testing.T) {
	trans := &echoTransformer{}
	def := &PipelineDef{
		Pipeline: "test",
		Nodes: []NodeDef{
			{Name: "a", Element: "fire", Transformer: "echo"},
			{Name: "b", Element: "water", Family: "legacy"},
		},
		Edges: []EdgeDef{
			{ID: "E1", Name: "a-to-b", From: "a", To: "b", When: "true"},
			{ID: "E2", Name: "b-done", From: "b", To: "_done", When: "true"},
		},
		Start: "a",
		Done:  "_done",
	}

	nodeFactory := func(nd NodeDef) Node {
		return &testNode{name: nd.Name}
	}

	reg := GraphRegistries{
		Transformers: TransformerRegistry{"echo": trans},
		Nodes:        NodeRegistry{"legacy": nodeFactory},
	}

	graph, err := def.BuildGraph(reg)
	if err != nil {
		t.Fatalf("BuildGraphWith: %v", err)
	}

	nodes := graph.Nodes()
	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(nodes))
	}
	if !IsTransformerNode(nodes[0]) {
		t.Error("node a should be transformer")
	}
	if IsTransformerNode(nodes[1]) {
		t.Error("node b should NOT be transformer")
	}
}

type testNode struct {
	name string
}

func (n *testNode) Name() string            { return n.name }
func (n *testNode) ElementAffinity() Element { return ElementFire }
func (n *testNode) Process(ctx context.Context, nc NodeContext) (Artifact, error) {
	return &testArtifact{typeName: n.name, confidence: 1.0, raw: map[string]any{"processed": true}}, nil
}

func TestTransformerNode_ResolveInput(t *testing.T) {
	trans := &echoTransformer{}
	node := &transformerNode{
		name:    "triage",
		element: ElementFire,
		trans:   trans,
		input:   "${recall.output}",
		config:  map[string]any{"key": "val"},
	}

	recallArtifact := &testArtifact{
		typeName:   "recall",
		confidence: 0.9,
		raw:        map[string]any{"match": true, "data": "recall-data"},
	}

	state := NewWalkerState("test")
	state.Outputs["recall"] = recallArtifact

	nc := NodeContext{
		WalkerState:   state,
		PriorArtifact: &testArtifact{raw: map[string]any{"prior": "should-be-ignored"}},
	}

	artifact, err := node.Process(context.Background(), nc)
	if err != nil {
		t.Fatalf("Process: %v", err)
	}

	m := artifact.Raw().(map[string]any)
	echoed, ok := m["echoed"].(map[string]any)
	if !ok {
		t.Fatalf("echoed type = %T, want map[string]any", m["echoed"])
	}
	if echoed["match"] != true {
		t.Errorf("expected recall data, got %v", echoed)
	}
}

func TestTransformerNode_RenderPrompt(t *testing.T) {
	captureNode := &transformerNode{
		name:    "triage",
		element: ElementFire,
		trans: TransformerFunc("capture", func(_ context.Context, tc *TransformerContext) (any, error) {
			return map[string]any{"prompt": tc.Prompt}, nil
		}),
		prompt: "Analyze {{.Node}} with threshold {{.Config.threshold}}",
		config: map[string]any{"threshold": 0.85},
	}

	state := NewWalkerState("test")
	nc := NodeContext{WalkerState: state}

	artifact, err := captureNode.Process(context.Background(), nc)
	if err != nil {
		t.Fatalf("Process: %v", err)
	}

	m := artifact.Raw().(map[string]any)
	prompt := m["prompt"].(string)
	expected := "Analyze triage with threshold 0.85"
	if prompt != expected {
		t.Errorf("rendered prompt = %q, want %q", prompt, expected)
	}
}

func TestTransformerNode_EmptyInput_FallsBackToPrior(t *testing.T) {
	trans := &echoTransformer{}
	node := &transformerNode{
		name:  "test",
		trans: trans,
	}

	state := NewWalkerState("test")
	nc := NodeContext{
		WalkerState:   state,
		PriorArtifact: &testArtifact{raw: map[string]any{"prior": true}},
	}

	artifact, err := node.Process(context.Background(), nc)
	if err != nil {
		t.Fatalf("Process: %v", err)
	}

	m := artifact.Raw().(map[string]any)
	echoed, ok := m["echoed"].(map[string]any)
	if !ok {
		t.Fatalf("echoed type = %T, want map[string]any", m["echoed"])
	}
	if echoed["prior"] != true {
		t.Errorf("expected prior artifact data, got %v", echoed)
	}
}

func TestIsTransformerNode(t *testing.T) {
	trans := &transformerNode{name: "t", trans: &echoTransformer{}}
	plain := &testNode{name: "p"}

	if !IsTransformerNode(trans) {
		t.Error("expected true for transformerNode")
	}
	if IsTransformerNode(plain) {
		t.Error("expected false for testNode")
	}
}
