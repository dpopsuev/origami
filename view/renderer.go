package view

// CircuitRenderer is the abstract contract for rendering surfaces.
// Implementations translate visual state into concrete output — terminal
// characters (Sumi), HTML/SVG (Washi), or JSON (Kami SSE).
type CircuitRenderer interface {
	// RenderTopology draws the full circuit from a snapshot and layout.
	// Called once when the renderer initializes or the topology changes.
	RenderTopology(snapshot CircuitSnapshot, layout CircuitLayout) error

	// ApplyDiff incrementally updates the rendered output for a single
	// state change (node transition, walker movement, breakpoint toggle).
	ApplyDiff(diff StateDiff) error

	// HandleInput processes user interaction (keyboard, mouse, agent command).
	// The concrete type of input is renderer-specific.
	HandleInput(input any) error
}
