package view

import (
	"testing"

	framework "github.com/dpopsuev/origami"
)

func TestGridLayout_LinearCircuit(t *testing.T) {
	def := testCircuitDef()
	var gl GridLayout
	layout, err := gl.Layout(def)
	if err != nil {
		t.Fatal(err)
	}

	if len(layout.Grid) != 4 {
		t.Fatalf("grid has %d nodes, want 4", len(layout.Grid))
	}

	recall := layout.Grid["recall"]
	if recall.Col != 0 {
		t.Errorf("recall col = %d, want 0 (start node)", recall.Col)
	}

	triage := layout.Grid["triage"]
	investigate := layout.Grid["investigate"]
	report := layout.Grid["report"]

	if triage.Col <= recall.Col {
		t.Error("triage should be after recall")
	}
	if investigate.Col <= triage.Col {
		t.Error("investigate should be after triage")
	}
	if report.Col <= investigate.Col {
		t.Error("report should be after investigate")
	}
}

func TestGridLayout_ZoneGrouping(t *testing.T) {
	def := testCircuitDef()
	var gl GridLayout
	layout, err := gl.Layout(def)
	if err != nil {
		t.Fatal(err)
	}

	recall := layout.Grid["recall"]
	if recall.Zone != "analysis" {
		t.Errorf("recall zone = %q, want %q", recall.Zone, "analysis")
	}
	report := layout.Grid["report"]
	if report.Zone != "output" {
		t.Errorf("report zone = %q, want %q", report.Zone, "output")
	}
}

func TestGridLayout_EmptyCircuit(t *testing.T) {
	def := &framework.CircuitDef{}
	var gl GridLayout
	layout, err := gl.Layout(def)
	if err != nil {
		t.Fatal(err)
	}
	if len(layout.Grid) != 0 {
		t.Errorf("empty circuit should produce empty grid, got %d", len(layout.Grid))
	}
}

func TestGridLayout_ParallelNodes(t *testing.T) {
	def := &framework.CircuitDef{
		Circuit: "parallel",
		Start:   "start",
		Nodes: []framework.NodeDef{
			{Name: "start"},
			{Name: "a"},
			{Name: "b"},
			{Name: "join"},
		},
		Edges: []framework.EdgeDef{
			{ID: "e1", From: "start", To: "a"},
			{ID: "e2", From: "start", To: "b"},
			{ID: "e3", From: "a", To: "join"},
			{ID: "e4", From: "b", To: "join"},
		},
	}

	var gl GridLayout
	layout, err := gl.Layout(def)
	if err != nil {
		t.Fatal(err)
	}

	startCell := layout.Grid["start"]
	aCell := layout.Grid["a"]
	bCell := layout.Grid["b"]
	joinCell := layout.Grid["join"]

	if startCell.Col != 0 {
		t.Errorf("start col = %d, want 0", startCell.Col)
	}
	if aCell.Col != 1 || bCell.Col != 1 {
		t.Errorf("a col = %d, b col = %d, both should be 1", aCell.Col, bCell.Col)
	}
	if aCell.Row == bCell.Row {
		t.Error("parallel nodes a and b should be in different rows")
	}
	if joinCell.Col != 2 {
		t.Errorf("join col = %d, want 2", joinCell.Col)
	}
}

func TestGridLayout_LoopEdgesIgnored(t *testing.T) {
	def := &framework.CircuitDef{
		Circuit: "loop",
		Start:   "a",
		Nodes: []framework.NodeDef{
			{Name: "a"},
			{Name: "b"},
		},
		Edges: []framework.EdgeDef{
			{ID: "e1", From: "a", To: "b"},
			{ID: "e2", From: "b", To: "a", Loop: true},
		},
	}

	var gl GridLayout
	layout, err := gl.Layout(def)
	if err != nil {
		t.Fatal(err)
	}
	if layout.Grid["a"].Col != 0 {
		t.Errorf("a col = %d, want 0", layout.Grid["a"].Col)
	}
	if layout.Grid["b"].Col != 1 {
		t.Errorf("b col = %d, want 1", layout.Grid["b"].Col)
	}
}

func TestGridLayout_Edges(t *testing.T) {
	def := testCircuitDef()
	var gl GridLayout
	layout, err := gl.Layout(def)
	if err != nil {
		t.Fatal(err)
	}
	if len(layout.Edges) != 3 {
		t.Errorf("edges = %d, want 3", len(layout.Edges))
	}
}

func TestGridLayout_Zones(t *testing.T) {
	def := testCircuitDef()
	var gl GridLayout
	layout, err := gl.Layout(def)
	if err != nil {
		t.Fatal(err)
	}
	if len(layout.Zones) != 2 {
		t.Errorf("zones = %d, want 2", len(layout.Zones))
	}
	zoneMap := make(map[string]string)
	for _, z := range layout.Zones {
		zoneMap[z.Name] = z.Element
	}
	if zoneMap["analysis"] != "fire" {
		t.Errorf("analysis element = %q, want fire", zoneMap["analysis"])
	}
	if zoneMap["output"] != "water" {
		t.Errorf("output element = %q, want water", zoneMap["output"])
	}
}

func TestLogicalLayout_LinearCircuit(t *testing.T) {
	def := testCircuitDef()
	var ll LogicalLayout
	layout, err := ll.Layout(def)
	if err != nil {
		t.Fatal(err)
	}

	if len(layout.Logical) != 4 {
		t.Fatalf("logical has %d nodes, want 4", len(layout.Logical))
	}

	recall := layout.Logical["recall"]
	triage := layout.Logical["triage"]
	investigate := layout.Logical["investigate"]
	report := layout.Logical["report"]

	if recall.X >= triage.X {
		t.Error("recall.X should be < triage.X")
	}
	if triage.X >= investigate.X {
		t.Error("triage.X should be < investigate.X")
	}
	if investigate.X >= report.X {
		t.Error("investigate.X should be < report.X")
	}
}

func TestLogicalLayout_EmptyCircuit(t *testing.T) {
	def := &framework.CircuitDef{}
	var ll LogicalLayout
	layout, err := ll.Layout(def)
	if err != nil {
		t.Fatal(err)
	}
	if len(layout.Logical) != 0 {
		t.Errorf("empty circuit should produce empty logical, got %d", len(layout.Logical))
	}
}

func TestLogicalLayout_ParallelNodes(t *testing.T) {
	def := &framework.CircuitDef{
		Circuit: "parallel",
		Start:   "start",
		Nodes: []framework.NodeDef{
			{Name: "start"},
			{Name: "a"},
			{Name: "b"},
			{Name: "join"},
		},
		Edges: []framework.EdgeDef{
			{ID: "e1", From: "start", To: "a"},
			{ID: "e2", From: "start", To: "b"},
			{ID: "e3", From: "a", To: "join"},
			{ID: "e4", From: "b", To: "join"},
		},
	}

	var ll LogicalLayout
	layout, err := ll.Layout(def)
	if err != nil {
		t.Fatal(err)
	}

	aPos := layout.Logical["a"]
	bPos := layout.Logical["b"]

	if aPos.X != bPos.X {
		t.Errorf("a.X = %f, b.X = %f, should be equal (same rank)", aPos.X, bPos.X)
	}
	if aPos.Y == bPos.Y {
		t.Error("parallel nodes should have different Y positions")
	}
}

func TestLogicalLayout_ZoneAssignment(t *testing.T) {
	def := testCircuitDef()
	var ll LogicalLayout
	layout, err := ll.Layout(def)
	if err != nil {
		t.Fatal(err)
	}

	if layout.Logical["recall"].Zone != "analysis" {
		t.Errorf("recall zone = %q, want analysis", layout.Logical["recall"].Zone)
	}
	if layout.Logical["report"].Zone != "output" {
		t.Errorf("report zone = %q, want output", layout.Logical["report"].Zone)
	}
}

func TestLogicalLayout_Edges(t *testing.T) {
	def := testCircuitDef()
	var ll LogicalLayout
	layout, err := ll.Layout(def)
	if err != nil {
		t.Fatal(err)
	}
	if len(layout.Edges) != 3 {
		t.Errorf("edges = %d, want 3", len(layout.Edges))
	}
}

func TestLogicalLayout_Zones(t *testing.T) {
	def := testCircuitDef()
	var ll LogicalLayout
	layout, err := ll.Layout(def)
	if err != nil {
		t.Fatal(err)
	}
	if len(layout.Zones) != 2 {
		t.Errorf("zones = %d, want 2", len(layout.Zones))
	}
}
