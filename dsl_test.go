package framework

import (
	"os"
	"testing"
)

func TestLoadCircuit_ValidYAML(t *testing.T) {
	data := []byte(`
circuit: test-pipe
description: "A test circuit"
nodes:
  - name: a
    element: fire
    family: start
  - name: b
    element: water
    family: finish
edges:
  - id: E1
    name: a-to-b
    from: a
    to: b
    condition: "always"
  - id: E2
    name: b-done
    from: b
    to: _done
    condition: "terminal"
start: a
done: _done
`)
	def, err := LoadCircuit(data)
	if err != nil {
		t.Fatalf("LoadCircuit: %v", err)
	}
	if def.Circuit != "test-pipe" {
		t.Errorf("Circuit = %q, want %q", def.Circuit, "test-pipe")
	}
	if def.Description != "A test circuit" {
		t.Errorf("Description = %q, want %q", def.Description, "A test circuit")
	}
	if len(def.Nodes) != 2 {
		t.Errorf("len(Nodes) = %d, want 2", len(def.Nodes))
	}
	if len(def.Edges) != 2 {
		t.Errorf("len(Edges) = %d, want 2", len(def.Edges))
	}
	if def.Start != "a" {
		t.Errorf("Start = %q, want %q", def.Start, "a")
	}
	if def.Done != "_done" {
		t.Errorf("Done = %q, want %q", def.Done, "_done")
	}
}

func TestLoadCircuit_WithZones(t *testing.T) {
	data := []byte(`
circuit: zoned
nodes:
  - name: x
    family: x
  - name: y
    family: y
zones:
  front:
    nodes: [x]
    approach: rapid
    stickiness: 2
  back:
    nodes: [y]
    approach: analytical
edges:
  - id: E1
    name: x-to-y
    from: x
    to: y
  - id: E2
    name: y-done
    from: y
    to: _done
start: x
done: _done
`)
	def, err := LoadCircuit(data)
	if err != nil {
		t.Fatalf("LoadCircuit: %v", err)
	}
	if len(def.Zones) != 2 {
		t.Fatalf("len(Zones) = %d, want 2", len(def.Zones))
	}
	front := def.Zones["front"]
	if front.Stickiness != 2 {
		t.Errorf("front.Stickiness = %d, want 2", front.Stickiness)
	}
	if front.Approach != "rapid" {
		t.Errorf("front.Approach = %q, want %q", front.Approach, "rapid")
	}
}

func TestLoadCircuit_InvalidYAML(t *testing.T) {
	data := []byte(`{invalid yaml: [`)
	_, err := LoadCircuit(data)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestValidate_Valid(t *testing.T) {
	def := &CircuitDef{
		Circuit: "test",
		Nodes:    []NodeDef{{Name: "a"}, {Name: "b"}},
		Edges:    []EdgeDef{{ID: "E1", From: "a", To: "b"}, {ID: "E2", From: "b", To: "_done"}},
		Start:    "a",
		Done:     "_done",
	}
	if err := def.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

func TestValidate_EmptyCircuitName(t *testing.T) {
	def := &CircuitDef{Nodes: []NodeDef{{Name: "a"}}, Edges: []EdgeDef{{ID: "E1", From: "a", To: "_done"}}, Start: "a", Done: "_done"}
	if err := def.Validate(); err == nil {
		t.Fatal("expected error for empty circuit name")
	}
}

func TestValidate_MissingStartNode(t *testing.T) {
	def := &CircuitDef{
		Circuit: "test",
		Nodes:    []NodeDef{{Name: "a"}},
		Edges:    []EdgeDef{{ID: "E1", From: "a", To: "_done"}},
		Start:    "nonexistent",
		Done:     "_done",
	}
	if err := def.Validate(); err == nil {
		t.Fatal("expected error for missing start node")
	}
}

func TestValidate_BrokenEdgeSource(t *testing.T) {
	def := &CircuitDef{
		Circuit: "test",
		Nodes:    []NodeDef{{Name: "a"}},
		Edges:    []EdgeDef{{ID: "E1", From: "ghost", To: "a"}},
		Start:    "a",
		Done:     "_done",
	}
	err := def.Validate()
	if err == nil {
		t.Fatal("expected error for broken edge source")
	}
	if !contains(err.Error(), "ghost") {
		t.Errorf("error should name the invalid reference: %v", err)
	}
}

func TestValidate_BrokenEdgeTarget(t *testing.T) {
	def := &CircuitDef{
		Circuit: "test",
		Nodes:    []NodeDef{{Name: "a"}},
		Edges:    []EdgeDef{{ID: "E1", From: "a", To: "ghost"}},
		Start:    "a",
		Done:     "_done",
	}
	err := def.Validate()
	if err == nil {
		t.Fatal("expected error for broken edge target")
	}
	if !contains(err.Error(), "ghost") {
		t.Errorf("error should name the invalid reference: %v", err)
	}
}

func TestValidate_BrokenZoneNode(t *testing.T) {
	def := &CircuitDef{
		Circuit: "test",
		Nodes:    []NodeDef{{Name: "a"}},
		Edges:    []EdgeDef{{ID: "E1", From: "a", To: "_done"}},
		Zones:    map[string]ZoneDef{"z": {Nodes: []string{"ghost"}}},
		Start:    "a",
		Done:     "_done",
	}
	err := def.Validate()
	if err == nil {
		t.Fatal("expected error for broken zone node reference")
	}
	if !contains(err.Error(), "ghost") {
		t.Errorf("error should name the invalid reference: %v", err)
	}
}

func TestValidate_DuplicateNodeName(t *testing.T) {
	def := &CircuitDef{
		Circuit: "test",
		Nodes:    []NodeDef{{Name: "a"}, {Name: "a"}},
		Edges:    []EdgeDef{{ID: "E1", From: "a", To: "_done"}},
		Start:    "a",
		Done:     "_done",
	}
	err := def.Validate()
	if err == nil {
		t.Fatal("expected error for duplicate node name")
	}
}

func TestValidate_DuplicateEdgeID(t *testing.T) {
	def := &CircuitDef{
		Circuit: "test",
		Nodes:    []NodeDef{{Name: "a"}, {Name: "b"}},
		Edges:    []EdgeDef{{ID: "E1", From: "a", To: "b"}, {ID: "E1", From: "b", To: "_done"}},
		Start:    "a",
		Done:     "_done",
	}
	err := def.Validate()
	if err == nil {
		t.Fatal("expected error for duplicate edge id")
	}
}

func TestRoundTripFidelity(t *testing.T) {
	original := &CircuitDef{
		Circuit:    "roundtrip",
		Description: "test round trip",
		Nodes:       []NodeDef{{Name: "a", Approach: "rapid", Family: "start"}, {Name: "b", Family: "end"}},
		Edges:       []EdgeDef{{ID: "E1", Name: "a-b", From: "a", To: "b"}, {ID: "E2", Name: "b-done", From: "b", To: "_done"}},
		Start:       "a",
		Done:        "_done",
	}

	data, err := original.MarshalYAML()
	if err != nil {
		t.Fatalf("MarshalYAML: %v", err)
	}

	restored, err := LoadCircuit(data)
	if err != nil {
		t.Fatalf("LoadCircuit round-trip: %v", err)
	}

	if restored.Circuit != original.Circuit {
		t.Errorf("Circuit = %q, want %q", restored.Circuit, original.Circuit)
	}
	if len(restored.Nodes) != len(original.Nodes) {
		t.Errorf("len(Nodes) = %d, want %d", len(restored.Nodes), len(original.Nodes))
	}
	if len(restored.Edges) != len(original.Edges) {
		t.Errorf("len(Edges) = %d, want %d", len(restored.Edges), len(original.Edges))
	}
	if restored.Start != original.Start {
		t.Errorf("Start = %q, want %q", restored.Start, original.Start)
	}
	if restored.Done != original.Done {
		t.Errorf("Done = %q, want %q", restored.Done, original.Done)
	}
	for i, n := range restored.Nodes {
		if n.Name != original.Nodes[i].Name {
			t.Errorf("Node[%d].Name = %q, want %q", i, n.Name, original.Nodes[i].Name)
		}
	}
}

func TestLoadCircuit_RealF0F6(t *testing.T) {
	data, err := os.ReadFile("testdata/rca-investigation.yaml")
	if err != nil {
		t.Fatalf("read rca-investigation.yaml: %v", err)
	}
	def, err := LoadCircuit(data)
	if err != nil {
		t.Fatalf("LoadCircuit: %v", err)
	}
	if err := def.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if def.Circuit != "rca-investigation" {
		t.Errorf("Circuit = %q, want %q", def.Circuit, "rca-investigation")
	}
	if len(def.Nodes) != 7 {
		t.Errorf("len(Nodes) = %d, want 7", len(def.Nodes))
	}
	if len(def.Zones) != 3 {
		t.Errorf("len(Zones) = %d, want 3", len(def.Zones))
	}
}

func TestLoadCircuit_RealDefectDialectic(t *testing.T) {
	data, err := os.ReadFile("testdata/defect-dialectic.yaml")
	if err != nil {
		t.Fatalf("read defect-dialectic.yaml: %v", err)
	}
	def, err := LoadCircuit(data)
	if err != nil {
		t.Fatalf("LoadCircuit: %v", err)
	}
	if err := def.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if def.Circuit != "defect-dialectic" {
		t.Errorf("Circuit = %q, want %q", def.Circuit, "defect-dialectic")
	}
	if len(def.Nodes) != 6 {
		t.Errorf("len(Nodes) = %d, want 6", len(def.Nodes))
	}
}

func TestLoadCircuit_NodeDescription(t *testing.T) {
	data := []byte(`
circuit: desc-test
nodes:
  - name: recall
    description: "Pattern-match against known failures database"
    element: fire
  - name: triage
    element: earth
edges:
  - id: E1
    name: proceed
    from: recall
    to: triage
start: recall
done: _done
`)
	def, err := LoadCircuit(data)
	if err != nil {
		t.Fatalf("LoadCircuit: %v", err)
	}
	if def.Nodes[0].Description != "Pattern-match against known failures database" {
		t.Errorf("Nodes[0].Description = %q, want pattern-match description", def.Nodes[0].Description)
	}
	if def.Nodes[1].Description != "" {
		t.Errorf("Nodes[1].Description = %q, want empty (optional field)", def.Nodes[1].Description)
	}
}

func TestLoadCircuit_NodeDescription_RoundTrip(t *testing.T) {
	data := []byte(`
circuit: roundtrip
nodes:
  - name: a
    description: "First node"
  - name: b
    description: "Second node"
edges:
  - id: E1
    from: a
    to: b
start: a
done: _done
`)
	def, err := LoadCircuit(data)
	if err != nil {
		t.Fatalf("LoadCircuit: %v", err)
	}
	for i, want := range []string{"First node", "Second node"} {
		if def.Nodes[i].Description != want {
			t.Errorf("Nodes[%d].Description = %q, want %q", i, def.Nodes[i].Description, want)
		}
	}
}

func TestLoadCircuit_CompactEdges(t *testing.T) {
	data := []byte(`
circuit: compact
nodes:
  - name: a
    element: fire
    edges:
      - name: go-to-b
        to: b
        when: "output.ready == true"
      - name: skip-to-c
        to: c
        shortcut: true
        when: "output.skip == true"
  - name: b
    element: water
    edges:
      - name: to-c
        to: c
        when: "true"
  - name: c
    element: earth
    edges:
      - name: done
        to: _done
        when: "true"
start: a
done: _done
`)
	def, err := LoadCircuit(data)
	if err != nil {
		t.Fatalf("LoadCircuit: %v", err)
	}
	if err := def.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if len(def.Edges) != 4 {
		t.Fatalf("len(Edges) = %d, want 4", len(def.Edges))
	}

	e0 := def.Edges[0]
	if e0.From != "a" || e0.To != "b" || e0.Name != "go-to-b" {
		t.Errorf("edge[0] = %+v, want from=a to=b name=go-to-b", e0)
	}
	if e0.ID != "a-go-to-b" {
		t.Errorf("edge[0].ID = %q, want %q", e0.ID, "a-go-to-b")
	}

	e1 := def.Edges[1]
	if !e1.Shortcut {
		t.Error("edge[1] should be shortcut")
	}

	e3 := def.Edges[3]
	if e3.From != "c" || e3.To != "_done" {
		t.Errorf("edge[3] = %+v, want from=c to=_done", e3)
	}
}

func TestLoadCircuit_FlowStyleEdges(t *testing.T) {
	data := []byte(`
circuit: linear
nodes:
  - name: setup
    edges: [run]
  - name: run
    edges: [report]
  - name: report
    edges: [_done]
start: setup
done: _done
`)
	def, err := LoadCircuit(data)
	if err != nil {
		t.Fatalf("LoadCircuit: %v", err)
	}
	if err := def.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if len(def.Edges) != 3 {
		t.Fatalf("len(Edges) = %d, want 3", len(def.Edges))
	}
	if def.Edges[0].From != "setup" || def.Edges[0].To != "run" {
		t.Errorf("edge[0] = %+v, want setup -> run", def.Edges[0])
	}
	if def.Edges[0].ID != "setup-run" {
		t.Errorf("edge[0].ID = %q, want %q", def.Edges[0].ID, "setup-run")
	}
}

func TestLoadCircuit_CompactVerboseEquivalence(t *testing.T) {
	compact := []byte(`
circuit: equiv
nodes:
  - name: a
    family: start
    edges:
      - name: proceed
        to: b
        when: "true"
  - name: b
    family: finish
    edges:
      - name: done
        to: _done
        when: "true"
start: a
done: _done
`)
	verbose := []byte(`
circuit: equiv
nodes:
  - name: a
    family: start
  - name: b
    family: finish
edges:
  - id: a-proceed
    name: proceed
    from: a
    to: b
    when: "true"
  - id: b-done
    name: done
    from: b
    to: _done
    when: "true"
start: a
done: _done
`)
	cDef, err := LoadCircuit(compact)
	if err != nil {
		t.Fatalf("LoadCircuit compact: %v", err)
	}
	vDef, err := LoadCircuit(verbose)
	if err != nil {
		t.Fatalf("LoadCircuit verbose: %v", err)
	}

	if len(cDef.Nodes) != len(vDef.Nodes) {
		t.Fatalf("node count: compact=%d verbose=%d", len(cDef.Nodes), len(vDef.Nodes))
	}
	if len(cDef.Edges) != len(vDef.Edges) {
		t.Fatalf("edge count: compact=%d verbose=%d", len(cDef.Edges), len(vDef.Edges))
	}
	for i, ce := range cDef.Edges {
		ve := vDef.Edges[i]
		if ce.ID != ve.ID || ce.From != ve.From || ce.To != ve.To || ce.When != ve.When || ce.Name != ve.Name {
			t.Errorf("edge[%d] mismatch:\n  compact: %+v\n  verbose: %+v", i, ce, ve)
		}
	}
}

func TestLoadCircuit_ImplicitFamily(t *testing.T) {
	data := []byte(`
circuit: fam
nodes:
  - name: recall
    edges: [triage]
  - name: triage
    family: triage-custom
    edges: [_done]
start: recall
done: _done
`)
	def, err := LoadCircuit(data)
	if err != nil {
		t.Fatalf("LoadCircuit: %v", err)
	}
	if def.Nodes[0].Family != "recall" {
		t.Errorf("Nodes[0].Family = %q, want %q (implicit from name)", def.Nodes[0].Family, "recall")
	}
	if def.Nodes[1].Family != "triage-custom" {
		t.Errorf("Nodes[1].Family = %q, want %q (explicit)", def.Nodes[1].Family, "triage-custom")
	}
}

func TestLoadCircuit_AutoGenerateEdgeID(t *testing.T) {
	data := []byte(`
circuit: ids
nodes:
  - name: a
    edges:
      - name: first path
        to: b
        when: "output.x == 1"
      - name: second path
        to: b
        when: "output.x == 2"
      - to: c
start: a
done: _done
`)
	def, err := LoadCircuit(data)
	if err != nil {
		t.Fatalf("LoadCircuit: %v", err)
	}
	wantIDs := []string{"a-first-path", "a-second-path", "a-c"}
	for i, want := range wantIDs {
		if def.Edges[i].ID != want {
			t.Errorf("edge[%d].ID = %q, want %q", i, def.Edges[i].ID, want)
		}
	}
}

func TestLoadCircuit_MixedEdges(t *testing.T) {
	data := []byte(`
circuit: mixed
nodes:
  - name: a
    edges:
      - name: inline
        to: b
        when: "true"
  - name: b
edges:
  - id: EXT1
    name: external
    from: b
    to: _done
    when: "true"
start: a
done: _done
`)
	def, err := LoadCircuit(data)
	if err != nil {
		t.Fatalf("LoadCircuit: %v", err)
	}
	if len(def.Edges) != 2 {
		t.Fatalf("len(Edges) = %d, want 2", len(def.Edges))
	}
	if def.Edges[0].ID != "EXT1" {
		t.Errorf("edge[0].ID = %q, want EXT1 (top-level first)", def.Edges[0].ID)
	}
	if def.Edges[1].ID != "a-inline" {
		t.Errorf("edge[1].ID = %q, want a-inline (inline second)", def.Edges[1].ID)
	}
}

func TestLoadCircuit_CompactEdge_MissingTo(t *testing.T) {
	data := []byte(`
circuit: bad
nodes:
  - name: a
    edges:
      - name: oops
        when: "true"
start: a
done: _done
`)
	_, err := LoadCircuit(data)
	if err == nil {
		t.Fatal("expected error for inline edge missing 'to'")
	}
	if !contains(err.Error(), "missing") {
		t.Errorf("error should mention 'missing': %v", err)
	}
}

func TestLoadCircuit_CompactEdge_LoopFlag(t *testing.T) {
	data := []byte(`
circuit: loopy
nodes:
  - name: a
    edges:
      - name: forward
        to: b
        when: "true"
  - name: b
    edges:
      - name: back
        to: a
        loop: true
        when: "state.loops.b < 3"
      - name: done
        to: _done
        when: "true"
start: a
done: _done
`)
	def, err := LoadCircuit(data)
	if err != nil {
		t.Fatalf("LoadCircuit: %v", err)
	}
	backEdge := def.Edges[1]
	if !backEdge.Loop {
		t.Error("back edge should have loop=true")
	}
	if backEdge.Name != "back" || backEdge.From != "b" || backEdge.To != "a" {
		t.Errorf("back edge = %+v", backEdge)
	}
}

func TestInferTopology_LinearChain(t *testing.T) {
	def := &CircuitDef{
		Circuit: "linear",
		Nodes:   []NodeDef{{Name: "a"}, {Name: "b"}, {Name: "c"}},
		Edges: []EdgeDef{
			{ID: "E1", From: "a", To: "b"},
			{ID: "E2", From: "b", To: "c"},
			{ID: "E3", From: "c", To: "_done"},
		},
		Start: "a",
		Done:  "_done",
	}
	InferTopology(def)
	for _, e := range def.Edges {
		if e.Shortcut {
			t.Errorf("edge %s should not be shortcut in linear chain", e.ID)
		}
		if e.Loop {
			t.Errorf("edge %s should not be loop in linear chain", e.ID)
		}
	}
}

func TestInferTopology_ForwardSkip(t *testing.T) {
	def := &CircuitDef{
		Circuit: "skip",
		Nodes:   []NodeDef{{Name: "a"}, {Name: "b"}, {Name: "c"}},
		Edges: []EdgeDef{
			{ID: "E1", From: "a", To: "b"},
			{ID: "E2", From: "b", To: "c"},
			{ID: "E3", From: "a", To: "c"},
			{ID: "E4", From: "c", To: "_done"},
		},
		Start: "a",
		Done:  "_done",
	}
	InferTopology(def)
	if !def.Edges[2].Shortcut {
		t.Error("edge E3 (a->c) should be inferred as shortcut")
	}
	if def.Edges[0].Shortcut {
		t.Error("edge E1 (a->b) should not be shortcut")
	}
}

func TestInferTopology_BackwardEdge(t *testing.T) {
	def := &CircuitDef{
		Circuit: "loop",
		Nodes:   []NodeDef{{Name: "a"}, {Name: "b"}},
		Edges: []EdgeDef{
			{ID: "E1", From: "a", To: "b"},
			{ID: "E2", From: "b", To: "a"},
			{ID: "E3", From: "b", To: "_done"},
		},
		Start: "a",
		Done:  "_done",
	}
	InferTopology(def)
	if !def.Edges[1].Loop {
		t.Error("edge E2 (b->a) should be inferred as loop")
	}
	if def.Edges[0].Loop {
		t.Error("edge E1 (a->b) should not be loop")
	}
}

func TestInferTopology_DiamondGraph(t *testing.T) {
	def := &CircuitDef{
		Circuit: "diamond",
		Nodes:   []NodeDef{{Name: "a"}, {Name: "b"}, {Name: "c"}, {Name: "d"}},
		Edges: []EdgeDef{
			{ID: "E1", From: "a", To: "b"},
			{ID: "E2", From: "a", To: "c"},
			{ID: "E3", From: "b", To: "d"},
			{ID: "E4", From: "c", To: "d"},
			{ID: "E5", From: "d", To: "_done"},
		},
		Start: "a",
		Done:  "_done",
	}
	InferTopology(def)
	for _, e := range def.Edges {
		if e.Loop {
			t.Errorf("edge %s should not be loop in diamond", e.ID)
		}
	}
	if def.Edges[2].Shortcut || def.Edges[3].Shortcut {
		t.Error("edges to d should not be shortcuts (both are direct)")
	}
}

func TestInferTopology_TerminalEdge(t *testing.T) {
	def := &CircuitDef{
		Circuit: "terminal",
		Nodes:   []NodeDef{{Name: "a"}, {Name: "b"}, {Name: "c"}},
		Edges: []EdgeDef{
			{ID: "E1", From: "a", To: "b"},
			{ID: "E2", From: "b", To: "c"},
			{ID: "E3", From: "a", To: "_done"},
			{ID: "E4", From: "c", To: "_done"},
		},
		Start: "a",
		Done:  "_done",
	}
	InferTopology(def)
	if def.Edges[2].Shortcut {
		t.Error("edge E3 (a->_done) should NOT be shortcut (terminal edges excluded)")
	}
}

func TestInferTopology_RCACircuit(t *testing.T) {
	data, err := os.ReadFile("testdata/rca-investigation.yaml")
	if err != nil {
		t.Fatalf("read rca-investigation.yaml: %v", err)
	}
	def, err := LoadCircuit(data)
	if err != nil {
		t.Fatalf("LoadCircuit: %v", err)
	}

	shortcutsBefore := map[string]bool{}
	loopsBefore := map[string]bool{}
	for _, e := range def.Edges {
		if e.Shortcut {
			shortcutsBefore[e.ID] = true
		}
		if e.Loop {
			loopsBefore[e.ID] = true
		}
	}

	InferTopology(def)

	for _, e := range def.Edges {
		if e.Shortcut && !shortcutsBefore[e.ID] {
			t.Logf("INFO: edge %s (%s->%s) inferred as shortcut (was not declared)", e.ID, e.From, e.To)
		}
		if e.Loop && !loopsBefore[e.ID] {
			t.Logf("INFO: edge %s (%s->%s) inferred as loop (was not declared)", e.ID, e.From, e.To)
		}
	}

	for id := range shortcutsBefore {
		for _, e := range def.Edges {
			if e.ID == id && !e.Shortcut {
				t.Errorf("edge %s was declared shortcut but inference cleared it", id)
			}
		}
	}
	for id := range loopsBefore {
		for _, e := range def.Edges {
			if e.ID == id && !e.Loop {
				t.Errorf("edge %s was declared loop but inference cleared it", id)
			}
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
