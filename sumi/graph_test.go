package sumi

import (
	"os"
	"strings"
	"testing"

	framework "github.com/dpopsuev/origami"
	"github.com/dpopsuev/origami/view"
)

func loadTestCircuit(t *testing.T, path string) *framework.CircuitDef {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	def, err := framework.LoadCircuit(data)
	if err != nil {
		t.Fatalf("load %s: %v", path, err)
	}
	return def
}

func TestRenderGraph_DialecticCircuit(t *testing.T) {
	def := loadTestCircuit(t, "../testdata/defect-dialectic.yaml")
	engine := &view.GridLayout{}
	layout, err := engine.Layout(def)
	if err != nil {
		t.Fatalf("layout: %v", err)
	}

	store := view.NewCircuitStore(def)
	defer store.Close()
	snap := store.Snapshot()

	output := RenderGraph(def, layout, snap, RenderOpts{NoColor: true})

	if output == "" {
		t.Fatal("RenderGraph returned empty")
	}

	for _, nd := range def.Nodes {
		if !strings.Contains(output, nd.Name) {
			t.Errorf("output missing node %q", nd.Name)
		}
	}

	for _, zoneName := range []string{"thesis", "discovery", "synthesis"} {
		if !strings.Contains(output, zoneName) {
			t.Errorf("output missing zone %q", zoneName)
		}
	}

	if !strings.Contains(output, "┌") || !strings.Contains(output, "┘") {
		t.Error("output missing box-drawing characters")
	}
}

func TestRenderGraph_RCACircuit(t *testing.T) {
	def := loadTestCircuit(t, "../testdata/rca-investigation.yaml")
	engine := &view.GridLayout{}
	layout, err := engine.Layout(def)
	if err != nil {
		t.Fatalf("layout: %v", err)
	}

	store := view.NewCircuitStore(def)
	defer store.Close()
	snap := store.Snapshot()

	output := RenderGraph(def, layout, snap, RenderOpts{NoColor: true})

	if output == "" {
		t.Fatal("RenderGraph returned empty")
	}

	for _, nd := range def.Nodes {
		if !strings.Contains(output, nd.Name) {
			t.Errorf("output missing node %q", nd.Name)
		}
	}

	if !strings.Contains(output, "▸") && !strings.Contains(output, "─") {
		t.Error("output missing edge characters")
	}
}

func TestRenderGraph_EmptyCircuit(t *testing.T) {
	def := &framework.CircuitDef{Circuit: "empty"}
	layout := view.CircuitLayout{}
	snap := view.CircuitSnapshot{CircuitName: "empty"}

	output := RenderGraph(def, layout, snap, RenderOpts{NoColor: true})
	if output != "(empty circuit)" {
		t.Errorf("expected empty message, got %q", output)
	}
}

func TestRenderGraph_NodeStates(t *testing.T) {
	def := &framework.CircuitDef{
		Circuit: "test",
		Nodes: []framework.NodeDef{
			{Name: "a"},
			{Name: "b"},
			{Name: "c"},
		},
		Edges: []framework.EdgeDef{
			{From: "a", To: "b"},
			{From: "b", To: "c"},
		},
		Start: "a",
		Done:  "c",
	}

	store := view.NewCircuitStore(def)
	defer store.Close()

	store.OnEvent(framework.WalkEvent{Type: framework.EventNodeEnter, Node: "a", Walker: "w1"})
	store.OnEvent(framework.WalkEvent{Type: framework.EventNodeExit, Node: "a"})
	store.OnEvent(framework.WalkEvent{Type: framework.EventNodeEnter, Node: "b", Walker: "w1"})

	snap := store.Snapshot()

	engine := &view.GridLayout{}
	layout, _ := engine.Layout(def)

	output := RenderGraph(def, layout, snap, RenderOpts{NoColor: true})

	if !strings.Contains(output, "✓") {
		t.Error("expected completed indicator (✓) for node a")
	}
	if !strings.Contains(output, "▶") {
		t.Error("expected active indicator (▶) for node b")
	}
	if !strings.Contains(output, "●") {
		t.Error("expected walker marker (●) for active node b")
	}
}

func TestRenderGraph_DSBadges(t *testing.T) {
	def := &framework.CircuitDef{
		Circuit: "badges",
		Nodes: []framework.NodeDef{
			{Name: "det", Transformer: "core.jq"},
			{Name: "stoch", Transformer: "core.llm"},
			{Name: "dial", Transformer: "core.dialectic"},
		},
		Edges: []framework.EdgeDef{
			{From: "det", To: "stoch"},
			{From: "stoch", To: "dial"},
		},
		Start: "det",
		Done:  "dial",
	}

	store := view.NewCircuitStore(def)
	defer store.Close()
	snap := store.Snapshot()

	engine := &view.GridLayout{}
	layout, _ := engine.Layout(def)

	output := RenderGraph(def, layout, snap, RenderOpts{NoColor: true})

	if !strings.Contains(output, "[D]") {
		t.Error("expected [D] badge for deterministic transformer")
	}
	if !strings.Contains(output, "[S]") {
		t.Error("expected [S] badge for stochastic transformer")
	}
	if !strings.Contains(output, "[Δ]") {
		t.Error("expected [Δ] badge for dialectic transformer")
	}
}

func TestRenderGraph_Breakpoints(t *testing.T) {
	def := &framework.CircuitDef{
		Circuit: "bp",
		Nodes: []framework.NodeDef{
			{Name: "a"},
			{Name: "b"},
		},
		Edges: []framework.EdgeDef{
			{From: "a", To: "b"},
		},
		Start: "a",
		Done:  "b",
	}

	store := view.NewCircuitStore(def)
	defer store.Close()
	store.SetBreakpoints([]string{"b"})
	snap := store.Snapshot()

	engine := &view.GridLayout{}
	layout, _ := engine.Layout(def)

	output := RenderGraph(def, layout, snap, RenderOpts{NoColor: true})
	if !strings.Contains(output, "◉") {
		t.Error("expected breakpoint marker (◉)")
	}
}

func TestRenderGraph_ShortcutEdge(t *testing.T) {
	def := loadTestCircuit(t, "../testdata/defect-dialectic.yaml")
	engine := &view.GridLayout{}
	layout, err := engine.Layout(def)
	if err != nil {
		t.Fatalf("layout: %v", err)
	}

	store := view.NewCircuitStore(def)
	defer store.Close()
	snap := store.Snapshot()

	output := RenderGraph(def, layout, snap, RenderOpts{NoColor: true})
	// The defect-dialectic circuit has shortcut edges; verify dashed edge marker
	if !strings.Contains(output, "╌") {
		t.Error("expected shortcut edge marker (╌)")
	}
}

func TestDSBadge(t *testing.T) {
	tests := []struct {
		transformer string
		want        string
	}{
		{"", ""},
		{"core.jq", "[D]"},
		{"core.file", "[D]"},
		{"core.template", "[D]"},
		{"core.llm", "[S]"},
		{"custom.analyzer", "[S]"},
		{"core.dialectic", "[Δ]"},
	}
	for _, tt := range tests {
		got := DSBadge(tt.transformer)
		if got != tt.want {
			t.Errorf("DSBadge(%q) = %q, want %q", tt.transformer, got, tt.want)
		}
	}
}

func TestElementColor_AllElementsCovered(t *testing.T) {
	elements := []string{"fire", "water", "earth", "lightning", "air", "void", "diamond"}
	for _, el := range elements {
		if _, ok := ElementColor[el]; !ok {
			t.Errorf("ElementColor missing %q", el)
		}
	}
}

func TestProgressBar(t *testing.T) {
	tests := []struct {
		current, total, width int
		wantFilled            int
	}{
		{0, 10, 10, 0},
		{5, 10, 10, 5},
		{10, 10, 10, 10},
		{0, 0, 10, 0},
	}
	for _, tt := range tests {
		bar := progressBar(tt.current, tt.total, tt.width)
		if tt.total == 0 {
			if !strings.Contains(bar, "░") {
				t.Errorf("expected empty bar for total=0")
			}
			continue
		}
		filled := strings.Count(bar, "█")
		if filled != tt.wantFilled {
			t.Errorf("progressBar(%d,%d,%d) filled=%d, want %d", tt.current, tt.total, tt.width, filled, tt.wantFilled)
		}
	}
}
