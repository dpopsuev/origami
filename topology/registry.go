package topology

import (
	"fmt"
	"sync"
)

// Registry holds named topology definitions.
type Registry struct {
	mu    sync.RWMutex
	topos map[string]*TopologyDef
}

// NewRegistry creates an empty topology registry.
func NewRegistry() *Registry {
	return &Registry{topos: make(map[string]*TopologyDef)}
}

// Register adds a topology definition. Returns an error on duplicate name.
func (r *Registry) Register(def *TopologyDef) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.topos[def.Name]; exists {
		return fmt.Errorf("topology %q already registered", def.Name)
	}
	r.topos[def.Name] = def
	return nil
}

// Lookup returns the topology definition for the given name.
func (r *Registry) Lookup(name string) (*TopologyDef, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	def, ok := r.topos[name]
	return def, ok
}

// List returns all registered topology names in no particular order.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.topos))
	for name := range r.topos {
		names = append(names, name)
	}
	return names
}

// DefaultRegistry returns a registry pre-loaded with the five built-in
// topology primitives: cascade, fan-out, fan-in, feedback-loop, bridge.
func DefaultRegistry() *Registry {
	r := NewRegistry()

	// Cascade: N stages in series. Entry has 0 inputs, exit has 0 downstream
	// outputs. Intermediates have exactly 1 input and 1 output.
	_ = r.Register(&TopologyDef{
		Name:        "cascade",
		Description: "N stages in series, each with exactly 1 input and 1 output",
		MinNodes:    2,
		MaxNodes:    -1,
		Rules: []PositionRule{
			{Position: PositionEntry, MinInputs: 0, MaxInputs: 0, MinOutputs: 1, MaxOutputs: 1},
			{Position: PositionIntermediate, MinInputs: 1, MaxInputs: 1, MinOutputs: 1, MaxOutputs: 1},
			{Position: PositionExit, MinInputs: 1, MaxInputs: 1, MinOutputs: 0, MaxOutputs: 0},
		},
	})

	// Fan-out: 1 source fans to N targets. Source has 1+ outputs,
	// each target has exactly 1 input.
	_ = r.Register(&TopologyDef{
		Name:        "fan-out",
		Description: "1 source fans to N target nodes",
		MinNodes:    2,
		MaxNodes:    -1,
		Rules: []PositionRule{
			{Position: PositionEntry, MinInputs: 0, MaxInputs: 0, MinOutputs: 2, MaxOutputs: -1},
			{Position: PositionExit, MinInputs: 1, MaxInputs: 1, MinOutputs: 0, MaxOutputs: 0},
		},
	})

	// Fan-in: N sources merge to 1 target. Each source has 1 output,
	// target has N inputs.
	_ = r.Register(&TopologyDef{
		Name:        "fan-in",
		Description: "N source nodes merge to 1 target node",
		MinNodes:    2,
		MaxNodes:    -1,
		Rules: []PositionRule{
			{Position: PositionEntry, MinInputs: 0, MaxInputs: 0, MinOutputs: 1, MaxOutputs: 1},
			{Position: PositionExit, MinInputs: 2, MaxInputs: -1, MinOutputs: 0, MaxOutputs: 0},
		},
	})

	// Feedback-loop: cascade with one back-edge. Same as cascade except the
	// feedback source has 1 extra output and the feedback target has 1 extra input.
	_ = r.Register(&TopologyDef{
		Name:        "feedback-loop",
		Description: "Cascade with one back-edge from downstream to upstream",
		MinNodes:    2,
		MaxNodes:    -1,
		Rules: []PositionRule{
			{Position: PositionEntry, MinInputs: 0, MaxInputs: 1, MinOutputs: 1, MaxOutputs: 1},
			{Position: PositionIntermediate, MinInputs: 1, MaxInputs: 2, MinOutputs: 1, MaxOutputs: 2},
			{Position: PositionExit, MinInputs: 1, MaxInputs: 1, MinOutputs: 0, MaxOutputs: 1},
		},
	})

	// Bridge: two parallel paths with a cross-connection.
	_ = r.Register(&TopologyDef{
		Name:        "bridge",
		Description: "Two parallel paths with a cross-connection edge",
		MinNodes:    4,
		MaxNodes:    -1,
		Rules: []PositionRule{
			{Position: PositionEntry, MinInputs: 0, MaxInputs: 0, MinOutputs: 1, MaxOutputs: 2},
			{Position: PositionIntermediate, MinInputs: 1, MaxInputs: 2, MinOutputs: 1, MaxOutputs: 2},
			{Position: PositionExit, MinInputs: 1, MaxInputs: 2, MinOutputs: 0, MaxOutputs: 0},
		},
	})

	// Delegate: exactly 1 input, exactly 1 output. The sub-walk replaces
	// fan-out — a delegate node produces a CircuitDef and walks it internally.
	_ = r.Register(&TopologyDef{
		Name:        "delegate",
		Description: "Single delegate node: 1 input, 1 output, sub-walk replaces fan-out",
		MinNodes:    1,
		MaxNodes:    -1,
		Rules: []PositionRule{
			{Position: PositionEntry, MinInputs: 0, MaxInputs: 0, MinOutputs: 1, MaxOutputs: 1},
			{Position: PositionIntermediate, MinInputs: 1, MaxInputs: 1, MinOutputs: 1, MaxOutputs: 1},
			{Position: PositionExit, MinInputs: 1, MaxInputs: 1, MinOutputs: 0, MaxOutputs: 0},
		},
	})

	return r
}
