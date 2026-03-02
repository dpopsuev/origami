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
