package sumi

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dpopsuev/origami/view"
)

func testSnapshotWithWalkers() *view.CircuitSnapshot {
	return &view.CircuitSnapshot{
		CircuitName: "test",
		Walkers: map[string]view.WalkerPosition{
			"w-01": {WalkerID: "w-01", Node: "recall", Element: "fire"},
			"w-02": {WalkerID: "w-02", Node: "triage", Element: "water"},
		},
		Nodes: map[string]view.NodeState{
			"recall":  {Name: "recall", State: view.NodeActive},
			"triage":  {Name: "triage", State: view.NodeActive},
			"report":  {Name: "report", State: view.NodeIdle},
		},
	}
}

func TestWorkersPanel_ID(t *testing.T) {
	snap := testSnapshotWithWalkers()
	p := NewWorkersPanel(snap, true, nil)
	if p.ID() != "workers" {
		t.Errorf("ID = %q, want workers", p.ID())
	}
}

func TestWorkersPanel_Focusable(t *testing.T) {
	snap := testSnapshotWithWalkers()
	p := NewWorkersPanel(snap, true, nil)
	if !p.Focusable() {
		t.Error("workers panel should be focusable")
	}
}

func TestWorkersPanel_DefaultIsAll(t *testing.T) {
	snap := testSnapshotWithWalkers()
	p := NewWorkersPanel(snap, true, nil)
	if p.SelectedWorker() != "" {
		t.Errorf("default selection should be All (empty), got %q", p.SelectedWorker())
	}
}

func TestWorkersPanel_KeyDownSelectsFirstWorker(t *testing.T) {
	snap := testSnapshotWithWalkers()
	var selected string
	p := NewWorkersPanel(snap, true, func(wid string) { selected = wid })
	p.Refresh()

	p.Update(tea.KeyMsg{Type: tea.KeyDown})

	if p.SelectedWorker() == "" {
		t.Error("after down arrow, should select a worker, not All")
	}
	if selected == "" {
		t.Error("onSelect callback should fire")
	}
}

func TestWorkersPanel_KeyUpFromAllWraps(t *testing.T) {
	snap := testSnapshotWithWalkers()
	p := NewWorkersPanel(snap, true, nil)
	p.Refresh()

	p.Update(tea.KeyMsg{Type: tea.KeyUp})

	if p.selected != len(p.workers)-1 {
		t.Errorf("up from All should wrap to last worker, selected=%d, workers=%d", p.selected, len(p.workers))
	}
}

func TestWorkersPanel_KeyDownPastEndWrapsToAll(t *testing.T) {
	snap := testSnapshotWithWalkers()
	p := NewWorkersPanel(snap, true, nil)
	p.Refresh()

	for i := 0; i <= len(p.workers); i++ {
		p.Update(tea.KeyMsg{Type: tea.KeyDown})
	}

	if p.SelectedWorker() != "" {
		t.Errorf("after cycling past all workers, should be All, got %q", p.SelectedWorker())
	}
}

func TestWorkersPanel_ViewContainsWorkerIDs(t *testing.T) {
	snap := testSnapshotWithWalkers()
	p := NewWorkersPanel(snap, true, nil)

	content := p.View(Rect{0, 0, 30, 10})

	if !strings.Contains(content, "w-01") {
		t.Error("view should contain worker w-01")
	}
	if !strings.Contains(content, "w-02") {
		t.Error("view should contain worker w-02")
	}
	if !strings.Contains(content, "All") {
		t.Error("view should contain All option")
	}
}

func TestWorkersPanel_ViewContainsNodePositions(t *testing.T) {
	snap := testSnapshotWithWalkers()
	p := NewWorkersPanel(snap, true, nil)

	content := p.View(Rect{0, 0, 40, 10})

	if !strings.Contains(content, "recall") {
		t.Error("view should contain node position 'recall'")
	}
}

func TestWorkersPanel_SelectByClick(t *testing.T) {
	snap := testSnapshotWithWalkers()
	var selected string
	p := NewWorkersPanel(snap, true, func(wid string) { selected = wid })
	p.Refresh()

	got := p.SelectByClick(0)
	if got != "" {
		t.Errorf("click Y=0 should select All, got %q", got)
	}
	if selected != "" {
		t.Errorf("onSelect should fire with All, got %q", selected)
	}

	got = p.SelectByClick(1)
	if got == "" {
		t.Error("click Y=1 should select first worker")
	}
}

func TestWorkersPanel_EmptySnapshot(t *testing.T) {
	snap := &view.CircuitSnapshot{Walkers: map[string]view.WalkerPosition{}}
	p := NewWorkersPanel(snap, true, nil)

	content := p.View(Rect{0, 0, 30, 10})
	if content == "" {
		t.Error("view should render something even with no workers")
	}
}
