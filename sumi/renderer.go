package sumi

import (
	"github.com/dpopsuev/origami/view"
)

// SumiRenderer implements view.CircuitRenderer for the terminal.
// It translates topology and state diffs into Bubble Tea model operations.
type SumiRenderer struct {
	model *Model
}

// NewSumiRenderer creates a renderer bound to a Sumi Model.
func NewSumiRenderer(m *Model) *SumiRenderer {
	return &SumiRenderer{model: m}
}

// RenderTopology is called once when the circuit topology is first available.
// Sumi uses it to store the initial snapshot; actual rendering happens in View().
func (r *SumiRenderer) RenderTopology(snapshot view.CircuitSnapshot, layout view.CircuitLayout) error {
	r.model.snap = snapshot
	r.model.layout = layout
	return nil
}

// ApplyDiff incrementally updates the model for a single state change.
func (r *SumiRenderer) ApplyDiff(diff view.StateDiff) error {
	r.model.applyDiff(diff)
	return nil
}

// HandleInput processes user interaction. In Sumi, input is handled
// by Bubble Tea's Update loop directly, so this is a no-op.
func (r *SumiRenderer) HandleInput(_ any) error {
	return nil
}

// Verify interface compliance at compile time.
var _ view.CircuitRenderer = (*SumiRenderer)(nil)
