package sumi

import (
	"strings"
	"testing"
	"time"

	framework "github.com/dpopsuev/origami"
	"github.com/dpopsuev/origami/view"

	tea "github.com/charmbracelet/bubbletea"
)

func testCircuit() *framework.CircuitDef {
	return &framework.CircuitDef{
		Circuit: "test",
		Nodes: []framework.NodeDef{
			{Name: "recall", Element: "fire"},
			{Name: "triage", Element: "water"},
			{Name: "investigate", Element: "earth"},
			{Name: "report", Element: "diamond"},
		},
		Edges: []framework.EdgeDef{
			{From: "recall", To: "triage"},
			{From: "triage", To: "investigate"},
			{From: "investigate", To: "report"},
		},
		Start: "recall",
		Done:  "report",
	}
}

func TestModel_Init(t *testing.T) {
	def := testCircuit()
	store := view.NewCircuitStore(def)
	defer store.Close()

	engine := &view.GridLayout{}
	layout, _ := engine.Layout(def)

	m := New(Config{
		Def:    def,
		Store:  store,
		Layout: layout,
		Opts:   RenderOpts{NoColor: true},
	})

	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init() should return a non-nil Cmd for store subscription")
	}
}

func TestModel_DiffMsg_AutoSelectsActiveNode(t *testing.T) {
	def := testCircuit()
	store := view.NewCircuitStore(def)
	defer store.Close()

	engine := &view.GridLayout{}
	layout, _ := engine.Layout(def)

	m := New(Config{
		Def:    def,
		Store:  store,
		Layout: layout,
		Opts:   RenderOpts{NoColor: true},
	})

	if m.selected != 0 {
		t.Fatalf("initial selected = %d, want 0", m.selected)
	}

	store.OnEvent(framework.WalkEvent{
		Type: framework.EventNodeEnter, Node: "triage", Walker: "w1",
	})

	diff := view.StateDiff{
		Type:      view.DiffNodeState,
		Node:      "triage",
		State:     view.NodeActive,
		Timestamp: time.Now(),
	}

	updated, _ := m.Update(DiffMsg(diff))
	um := updated.(Model)

	if um.nodeOrder[um.selected] != "triage" {
		t.Errorf("selected = %q, want triage", um.nodeOrder[um.selected])
	}
}

func TestModel_KeyQ_Quits(t *testing.T) {
	def := testCircuit()
	store := view.NewCircuitStore(def)
	defer store.Close()

	engine := &view.GridLayout{}
	layout, _ := engine.Layout(def)

	m := New(Config{
		Def:    def,
		Store:  store,
		Layout: layout,
		Opts:   RenderOpts{NoColor: true},
	})

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("pressing 'q' should return a quit command")
	}
}

func TestModel_TabCyclesNodes(t *testing.T) {
	def := testCircuit()
	store := view.NewCircuitStore(def)
	defer store.Close()

	engine := &view.GridLayout{}
	layout, _ := engine.Layout(def)

	m := New(Config{
		Def:    def,
		Store:  store,
		Layout: layout,
		Opts:   RenderOpts{NoColor: true},
	})

	for i := 0; i < len(def.Nodes)+1; i++ {
		expected := (i + 1) % len(def.Nodes)
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
		m = updated.(Model)
		if m.selected != expected {
			t.Errorf("after tab %d: selected = %d, want %d", i+1, m.selected, expected)
		}
	}
}

func TestModel_EnterTogglesInspector(t *testing.T) {
	def := testCircuit()
	store := view.NewCircuitStore(def)
	defer store.Close()

	engine := &view.GridLayout{}
	layout, _ := engine.Layout(def)

	m := New(Config{
		Def:    def,
		Store:  store,
		Layout: layout,
		Opts:   RenderOpts{NoColor: true},
	})

	if m.inspecting {
		t.Fatal("should not be inspecting initially")
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if !m.inspecting {
		t.Error("Enter should toggle inspecting on")
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if m.inspecting {
		t.Error("Enter again should toggle inspecting off")
	}
}

func TestModel_SearchSelectsNode(t *testing.T) {
	def := testCircuit()
	store := view.NewCircuitStore(def)
	defer store.Close()

	engine := &view.GridLayout{}
	layout, _ := engine.Layout(def)

	m := New(Config{
		Def:    def,
		Store:  store,
		Layout: layout,
		Opts:   RenderOpts{NoColor: true},
	})

	// Enter search mode
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = updated.(Model)
	if !m.searching {
		t.Fatal("'/' should enter search mode")
	}

	for _, ch := range "inv" {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		m = updated.(Model)
	}

	// Press Enter to apply search
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if m.nodeOrder[m.selected] != "investigate" {
		t.Errorf("search for 'inv' selected %q, want investigate", m.nodeOrder[m.selected])
	}
	if m.searching {
		t.Error("search mode should be off after Enter")
	}
}

func TestModel_View_IncludesStatusBar(t *testing.T) {
	def := testCircuit()
	store := view.NewCircuitStore(def)
	defer store.Close()

	engine := &view.GridLayout{}
	layout, _ := engine.Layout(def)

	m := New(Config{
		Def:    def,
		Store:  store,
		Layout: layout,
		Opts:   RenderOpts{NoColor: true},
	})

	// Simulate window size
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated.(Model)

	v := m.View()
	if v == "" {
		t.Fatal("View returned empty")
	}
	hasAny := strings.Contains(v, "Kami:") || strings.Contains(v, "Selected:") || strings.Contains(v, "recall")
	if !hasAny {
		t.Error("View missing status bar elements")
	}
}

func TestSumiRenderer_ImplementsInterface(t *testing.T) {
	var _ view.CircuitRenderer = (*SumiRenderer)(nil)
}

// --- DiffReset tests ---
// These tests verify that the Model correctly rebuilds its rendering
// state (def, layout, nodeOrder) when the store emits a DiffReset event,
// which happens on SSE reconnect after a session swap.

// TestModel_DiffReset_EmptyToPopulated verifies the core bug:
// Model starts with an empty def (Sumi launched before any session),
// then receives DiffReset from store.Reset with a real circuit def.
// The Model should rebuild def, layout, nodeOrder so the circuit renders.
func TestModel_DiffReset_EmptyToPopulated(t *testing.T) {
	emptyDef := &framework.CircuitDef{Circuit: "watch"}
	store := view.NewCircuitStore(emptyDef)
	defer store.Close()

	emptyLayout := view.CircuitLayout{}

	m := New(Config{
		Def:    emptyDef,
		Store:  store,
		Layout: emptyLayout,
		Opts:   RenderOpts{NoColor: true},
	})
	m.Init()

	// Verify initial state: empty circuit.
	if len(m.nodeOrder) != 0 {
		t.Fatalf("initial nodeOrder should be empty, got %d", len(m.nodeOrder))
	}
	if len(m.layout.Grid) != 0 {
		t.Fatalf("initial layout.Grid should be empty, got %d", len(m.layout.Grid))
	}

	// Simulate what rebootstrapStore does: Reset the store with a real def.
	realDef := testCircuit()
	store.Reset(realDef)

	// Feed DiffReset through the Model's Update loop.
	resetDiff := view.StateDiff{
		Type:      view.DiffReset,
		Timestamp: time.Now(),
	}
	updated, cmd := m.Update(DiffMsg(resetDiff))
	m = updated.(Model)

	if cmd == nil {
		t.Fatal("Update should return a cmd to wait for next diff")
	}

	// After DiffReset, the Model should have rebuilt its rendering state.
	if len(m.nodeOrder) == 0 {
		t.Error("BUG: nodeOrder is still empty after DiffReset — Model does not rebuild on reset")
	}
	if len(m.layout.Grid) == 0 {
		t.Error("BUG: layout.Grid is still empty after DiffReset — Model does not rebuild on reset")
	}
	if len(m.def.Nodes) == 0 {
		t.Error("BUG: def.Nodes is still empty after DiffReset — Model does not rebuild on reset")
	}

	// The circuit should render something other than "(empty circuit)".
	updated, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated.(Model)
	v := m.View()
	if strings.Contains(v, "(empty circuit)") {
		t.Error("BUG: View still shows '(empty circuit)' after DiffReset with populated store")
	}

	t.Logf("after DiffReset: nodeOrder=%d, layout.Grid=%d, def.Nodes=%d",
		len(m.nodeOrder), len(m.layout.Grid), len(m.def.Nodes))
}

// TestModel_DiffReset_PreservesEventCount verifies that DiffReset doesn't
// zero the event counter — it should increment like any other diff.
func TestModel_DiffReset_PreservesEventCount(t *testing.T) {
	def := testCircuit()
	store := view.NewCircuitStore(def)
	defer store.Close()

	engine := &view.GridLayout{}
	layout, _ := engine.Layout(def)

	m := New(Config{
		Def:    def,
		Store:  store,
		Layout: layout,
		Opts:   RenderOpts{NoColor: true},
	})
	m.Init()

	// Send a few normal diffs.
	for i := 0; i < 5; i++ {
		store.OnEvent(framework.WalkEvent{
			Type: framework.EventNodeEnter, Node: "recall", Walker: "w1",
		})
		diff := view.StateDiff{Type: view.DiffNodeState, Node: "recall", State: view.NodeActive, Timestamp: time.Now()}
		updated, _ := m.Update(DiffMsg(diff))
		m = updated.(Model)
	}

	countBefore := m.eventCount

	// Now a DiffReset.
	resetDiff := view.StateDiff{Type: view.DiffReset, Timestamp: time.Now()}
	updated, _ := m.Update(DiffMsg(resetDiff))
	m = updated.(Model)

	if m.eventCount != countBefore+1 {
		t.Errorf("eventCount = %d, want %d (DiffReset should increment, not reset)", m.eventCount, countBefore+1)
	}
}

// TestModel_SessionStartAfterWatch simulates the real-world scenario:
// Sumi starts watching before any session → empty circuit. Then a session
// starts, the store gets populated via SSE events. Verifies the Model
// eventually shows the circuit.
func TestModel_SessionStartAfterWatch(t *testing.T) {
	emptyDef := &framework.CircuitDef{Circuit: "watch"}
	store := view.NewCircuitStore(emptyDef)
	defer store.Close()

	m := New(Config{
		Def:    emptyDef,
		Store:  store,
		Layout: view.CircuitLayout{},
		Opts:   RenderOpts{NoColor: true},
	})
	m.Init()

	// Set window size so View() renders.
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated.(Model)

	// Verify: "(empty circuit)" initially.
	v := m.View()
	if !strings.Contains(v, "(empty circuit)") {
		t.Fatal("expected '(empty circuit)' before session start")
	}

	// Now simulate session start: store.Reset with a real circuit def,
	// followed by node/walker events (mimicking rebootstrapStore + SSE).
	realDef := testCircuit()
	store.Reset(realDef)

	// Feed the DiffReset.
	resetDiff := view.StateDiff{Type: view.DiffReset, Timestamp: time.Now()}
	updated, _ = m.Update(DiffMsg(resetDiff))
	m = updated.(Model)

	// Feed walker events through the store (as real SSE flow does).
	walkEvents := []framework.WalkEvent{
		{Type: framework.EventNodeEnter, Node: "recall", Walker: "C01"},
		{Type: framework.EventNodeEnter, Node: "triage", Walker: "C02"},
	}
	for _, we := range walkEvents {
		store.OnEvent(we)
	}

	// Now feed the resulting DiffMsgs to the Model (Bubble Tea Update loop).
	diffs := []view.StateDiff{
		{Type: view.DiffNodeState, Node: "recall", State: view.NodeActive, Timestamp: time.Now()},
		{Type: view.DiffWalkerAdded, Node: "recall", Walker: "C01", Timestamp: time.Now()},
		{Type: view.DiffNodeState, Node: "triage", State: view.NodeActive, Timestamp: time.Now()},
		{Type: view.DiffWalkerAdded, Node: "triage", Walker: "C02", Timestamp: time.Now()},
	}
	for _, diff := range diffs {
		updated, _ = m.Update(DiffMsg(diff))
		m = updated.(Model)
	}

	// Now the Model should show the circuit, not "(empty circuit)".
	v = m.View()
	if strings.Contains(v, "(empty circuit)") {
		t.Error("BUG: View still shows '(empty circuit)' after session start + DiffReset + events")
	}

	// Should have nodes in nodeOrder.
	if len(m.nodeOrder) == 0 {
		t.Error("BUG: nodeOrder still empty after session start")
	}

	// Snapshot should reflect the store state.
	if len(m.snap.Walkers) < 2 {
		t.Errorf("snap.Walkers = %d, want >= 2 (C01, C02)", len(m.snap.Walkers))
	}

	t.Logf("after session start: nodeOrder=%d, walkers=%d, view contains circuit=%v",
		len(m.nodeOrder), len(m.snap.Walkers), !strings.Contains(m.View(), "(empty circuit)"))
}
