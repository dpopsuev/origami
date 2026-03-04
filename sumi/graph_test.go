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

	if !strings.Contains(output, "⚙") {
		t.Error("expected ⚙ badge for deterministic transformer")
	}
	if !strings.Contains(output, "✦") {
		t.Error("expected ✦ badge for stochastic transformer")
	}
	if !strings.Contains(output, "Δ") {
		t.Error("expected Δ badge for dialectic transformer")
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
		{"core.jq", "⚙"},
		{"core.file", "⚙"},
		{"core.template", "⚙"},
		{"core.llm", "✦"},
		{"custom.analyzer", "✦"},
		{"core.dialectic", "Δ"},
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

func TestRenderGraph_ShortcutsVisibleBelow(t *testing.T) {
	def := &framework.CircuitDef{
		Circuit: "shortcuts",
		Nodes: []framework.NodeDef{
			{Name: "a"}, {Name: "b"}, {Name: "c"}, {Name: "d"},
		},
		Edges: []framework.EdgeDef{
			{From: "a", To: "b"},
			{From: "b", To: "c"},
			{From: "c", To: "d"},
			{From: "a", To: "d", Shortcut: true},
		},
		Start: "a",
		Done:  "d",
	}

	store := view.NewCircuitStore(def)
	defer store.Close()
	snap := store.Snapshot()

	engine := &view.GridLayout{}
	layout, _ := engine.Layout(def)

	output := RenderGraph(def, layout, snap, RenderOpts{NoColor: true})

	lines := strings.Split(output, "\n")
	nodeRow := -1
	shortcutRow := -1
	for i, line := range lines {
		if strings.Contains(line, "───▸") {
			nodeRow = i
		}
		if strings.Contains(line, "╌") && strings.Contains(line, "▴") {
			shortcutRow = i
		}
	}

	if nodeRow == -1 {
		t.Fatal("main path not found in output")
	}
	if shortcutRow == -1 {
		t.Fatal("shortcut arc not found below main path (expected ╌ and ▴)")
	}
	if shortcutRow <= nodeRow {
		t.Errorf("shortcut (row %d) should be BELOW main path (row %d)", shortcutRow, nodeRow)
	}
}

func TestRenderGraph_ChannelsSeparated(t *testing.T) {
	def := &framework.CircuitDef{
		Circuit: "multi-shortcut",
		Nodes: []framework.NodeDef{
			{Name: "a"}, {Name: "b"}, {Name: "c"}, {Name: "d"}, {Name: "e"},
		},
		Edges: []framework.EdgeDef{
			{From: "a", To: "b"},
			{From: "b", To: "c"},
			{From: "c", To: "d"},
			{From: "d", To: "e"},
			{From: "a", To: "e", Shortcut: true},
			{From: "b", To: "e", Shortcut: true},
		},
		Start: "a",
		Done:  "e",
	}

	engine := &view.GridLayout{}
	layout, _ := engine.Layout(def)

	routing := computeEdgeRouting(def, layout, def.Done)

	if routing.channels < 2 {
		t.Errorf("expected at least 2 channels for overlapping shortcuts, got %d", routing.channels)
	}

	ch0, ch1 := -1, -1
	for _, re := range routing.below {
		if re.from == "b" && re.to == "e" {
			ch0 = re.channel
		}
		if re.from == "a" && re.to == "e" {
			ch1 = re.channel
		}
	}
	if ch0 == ch1 {
		t.Errorf("overlapping shortcuts a→e and b→e should be on different channels, both on %d", ch0)
	}
}

func TestRenderGraph_LoopRoutesToTarget(t *testing.T) {
	def := &framework.CircuitDef{
		Circuit: "loop-target",
		Nodes: []framework.NodeDef{
			{Name: "a"}, {Name: "b"}, {Name: "c"},
		},
		Edges: []framework.EdgeDef{
			{From: "a", To: "b"},
			{From: "b", To: "c"},
			{From: "c", To: "a", Loop: true},
		},
		Start: "a",
		Done:  "c",
	}

	store := view.NewCircuitStore(def)
	defer store.Close()
	snap := store.Snapshot()

	engine := &view.GridLayout{}
	layout, _ := engine.Layout(def)

	output := RenderGraph(def, layout, snap, RenderOpts{NoColor: true})

	// The ◀ arrowhead should be at node a's left edge (the loop target),
	// not at node c's position.
	lines := strings.Split(output, "\n")
	found := false
	for _, line := range lines {
		idx := strings.Index(line, "◀")
		if idx < 0 {
			continue
		}
		found = true
		// ◀ should be at node a's x position (leftmost). Node a is at col 0,
		// so its origin x = cellGapH = 4. The ◀ should be at x=4.
		if idx != cellGapH {
			t.Errorf("loop ◀ at x=%d, expected x=%d (target node a's left edge)", idx, cellGapH)
		}
	}
	if !found {
		t.Error("loop arrowhead ◀ not found in output")
	}
}

func TestRenderGraph_VirtualDoneFiltered(t *testing.T) {
	def := loadTestCircuit(t, "../testdata/rca-investigation.yaml")
	engine := &view.GridLayout{}
	layout, _ := engine.Layout(def)

	store := view.NewCircuitStore(def)
	defer store.Close()
	snap := store.Snapshot()

	output := RenderGraph(def, layout, snap, RenderOpts{NoColor: true})

	// _done is a virtual terminal node; it should NOT be rendered as a box
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "_done") {
			t.Error("virtual _done node should not appear in rendered output")
		}
	}

	// Real nodes must still be present
	for _, name := range []string{"recall", "triage", "resolve", "investigate", "correlate", "review", "report"} {
		if !strings.Contains(output, name) {
			t.Errorf("real node %q missing from output", name)
		}
	}
}

func TestRenderGraph_RealDoneNotFiltered(t *testing.T) {
	def := &framework.CircuitDef{
		Circuit: "real-done",
		Nodes: []framework.NodeDef{
			{Name: "a"}, {Name: "b"},
		},
		Edges: []framework.EdgeDef{
			{From: "a", To: "b"},
		},
		Start: "a",
		Done:  "b",
	}

	store := view.NewCircuitStore(def)
	defer store.Close()
	snap := store.Snapshot()

	engine := &view.GridLayout{}
	layout, _ := engine.Layout(def)

	output := RenderGraph(def, layout, snap, RenderOpts{NoColor: true})

	if !strings.Contains(output, "b") {
		t.Error("real done node 'b' should still be rendered")
	}
}

func TestComputeEdgeRouting_Deduplication(t *testing.T) {
	def := &framework.CircuitDef{
		Circuit: "dedup",
		Nodes: []framework.NodeDef{
			{Name: "a"}, {Name: "b"},
		},
		Edges: []framework.EdgeDef{
			{From: "a", To: "b"},
			{From: "a", To: "b"},
			{From: "a", To: "b", Shortcut: true},
		},
		Start: "a",
		Done:  "b",
	}

	engine := &view.GridLayout{}
	layout, _ := engine.Layout(def)

	routing := computeEdgeRouting(def, layout, def.Done)

	total := len(routing.inline) + len(routing.below) + len(routing.loops)
	if total != 1 {
		t.Errorf("expected 1 deduplicated edge, got %d", total)
	}
	if len(routing.inline) != 1 {
		t.Errorf("expected 1 inline edge, got %d inline, %d below, %d loops",
			len(routing.inline), len(routing.below), len(routing.loops))
	}
	if routing.inline[0].shortcut {
		t.Error("merged edge should be non-shortcut (normal edge wins)")
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
