package framework

import (
	"context"
	"testing"
)

type echoNode struct {
	name string
	elem Element
	val  string
}

func (n *echoNode) Name() string            { return n.name }
func (n *echoNode) ElementAffinity() Element { return n.elem }
func (n *echoNode) Process(_ context.Context, _ NodeContext) (Artifact, error) {
	return &stubCacheArtifact{val: n.val}, nil
}

func TestAtomicMarble_Walk(t *testing.T) {
	inner := &echoNode{name: "echo", val: "hello"}
	marble := NewAtomicMarble(inner)

	if marble.IsComposite() {
		t.Fatal("atomic marble should not be composite")
	}
	if marble.PipelineDef() != nil {
		t.Fatal("atomic marble should have nil PipelineDef")
	}
	if marble.Name() != "echo" {
		t.Errorf("Name() = %q, want echo", marble.Name())
	}

	art, err := marble.Process(context.Background(), NodeContext{})
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if art.(*stubCacheArtifact).val != "hello" {
		t.Errorf("got %v, want hello", art.Raw())
	}
}

func TestCompositeMarble_SubWalk(t *testing.T) {
	subDef := &PipelineDef{
		Pipeline: "sub",
		Nodes: []NodeDef{
			{Name: "a", Family: "a"},
			{Name: "b", Family: "b"},
		},
		Edges: []EdgeDef{
			{ID: "e1", Name: "a-to-b", From: "a", To: "b", When: "true"},
			{ID: "e2", Name: "b-done", From: "b", To: "DONE", When: "true"},
		},
		Start: "a",
		Done:  "DONE",
	}

	reg := GraphRegistries{
		Nodes: NodeRegistry{
			"a": func(def NodeDef) Node { return &echoNode{name: def.Name, val: "from-a"} },
			"b": func(def NodeDef) Node { return &echoNode{name: def.Name, val: "from-b"} },
		},
	}

	marble := NewCompositeMarble("test-marble", "", subDef, reg)
	if !marble.IsComposite() {
		t.Fatal("composite marble should be composite")
	}
	if marble.PipelineDef() != subDef {
		t.Fatal("PipelineDef should return the sub definition")
	}

	art, err := marble.Process(context.Background(), NodeContext{})
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if art == nil {
		t.Fatal("expected non-nil artifact from sub-walk")
	}
}

func TestCompositeMarble_DepthLimit(t *testing.T) {
	subDef := &PipelineDef{
		Pipeline: "deep",
		Nodes:    []NodeDef{{Name: "a", Family: "a"}},
		Edges:    []EdgeDef{{ID: "e1", From: "a", To: "DONE", When: "true"}},
		Start:    "a",
		Done:     "DONE",
	}

	reg := GraphRegistries{
		Nodes: NodeRegistry{
			"a": func(def NodeDef) Node { return &echoNode{name: def.Name, val: "deep"} },
		},
	}

	marble := NewCompositeMarble("deep", "", subDef, reg)
	marble.depth = maxMarbleDepth

	_, err := marble.Process(context.Background(), NodeContext{})
	if err == nil {
		t.Fatal("expected depth limit error")
	}
}

func TestMarbleRegistry_FQCNLookup(t *testing.T) {
	reg := MarbleRegistry{
		"ns.scorer": func(def NodeDef) Marble {
			return NewAtomicMarble(&echoNode{name: def.Name, val: "scored"})
		},
	}

	nd := NodeDef{Name: "test", Marble: "ns.scorer"}
	node, err := resolveMarble(nd, reg, 0)
	if err != nil {
		t.Fatalf("resolveMarble: %v", err)
	}
	art, err := node.Process(context.Background(), NodeContext{})
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if art.(*stubCacheArtifact).val != "scored" {
		t.Errorf("got %v, want scored", art.Raw())
	}
}

func TestMarbleRegistry_NotFound(t *testing.T) {
	reg := MarbleRegistry{}
	nd := NodeDef{Name: "test", Marble: "nonexistent"}
	_, err := resolveMarble(nd, reg, 0)
	if err == nil {
		t.Fatal("expected error for missing marble")
	}
}

func TestMarbleRegistry_NilRegistry(t *testing.T) {
	nd := NodeDef{Name: "test", Marble: "scorer"}
	_, err := resolveMarble(nd, nil, 0)
	if err == nil {
		t.Fatal("expected error for nil registry")
	}
}

func TestDetectMarbleCycle_NoCycle(t *testing.T) {
	reg := MarbleRegistry{
		"a": func(def NodeDef) Marble {
			return NewAtomicMarble(&echoNode{name: "a", val: "a"})
		},
	}
	if err := DetectMarbleCycle(reg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDetectMarbleCycle_DirectCycle(t *testing.T) {
	subDef := &PipelineDef{
		Pipeline: "cycle",
		Nodes:    []NodeDef{{Name: "inner", Marble: "self"}},
		Edges:    []EdgeDef{{ID: "e1", From: "inner", To: "DONE", When: "true"}},
		Start:    "inner",
		Done:     "DONE",
	}

	reg := MarbleRegistry{
		"self": func(def NodeDef) Marble {
			return NewCompositeMarble("self", "", subDef, GraphRegistries{})
		},
	}
	err := DetectMarbleCycle(reg)
	if err == nil {
		t.Fatal("expected cycle detection error")
	}
}

func TestBuildGraphWith_MarbleNode(t *testing.T) {
	def := &PipelineDef{
		Pipeline: "test",
		Nodes: []NodeDef{
			{Name: "start", Marble: "echo-marble"},
		},
		Edges: []EdgeDef{
			{ID: "e1", Name: "done", From: "start", To: "DONE", When: "true"},
		},
		Start: "start",
		Done:  "DONE",
	}

	reg := GraphRegistries{
		Marbles: MarbleRegistry{
			"echo-marble": func(nd NodeDef) Marble {
				return NewAtomicMarble(&echoNode{name: nd.Name, val: "marble-out"})
			},
		},
	}

	graph, err := def.BuildGraph(reg)
	if err != nil {
		t.Fatalf("BuildGraphWith: %v", err)
	}

	walker := NewProcessWalker("test")
	err = graph.Walk(context.Background(), walker, "start")
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
}
