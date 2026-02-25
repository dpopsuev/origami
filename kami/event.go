// Package kami provides a live agentic pipeline debugger for Origami.
//
// Kami (神) — "divine spirit" in Shinto. The agents walking the pipeline
// graph are the kami inhabiting the nodes.
//
// Kami follows the Demiurge pattern: triple-homed process bridging
// AI (MCP stdio), browser (HTTP/SSE), and pipeline (WalkObserver + SignalBus).
package kami

import "time"

// EventType classifies kami events for routing and rendering.
type EventType string

const (
	// Walk events (from WalkObserver)
	EventNodeEnter    EventType = "node_enter"
	EventNodeExit     EventType = "node_exit"
	EventEdgeEvaluate EventType = "edge_evaluate"
	EventTransition   EventType = "transition"
	EventWalkerSwitch EventType = "walker_switch"
	EventFanOutStart  EventType = "fan_out_start"
	EventFanOutEnd    EventType = "fan_out_end"
	EventWalkComplete EventType = "walk_complete"
	EventWalkError    EventType = "walk_error"

	// Signal events (from SignalBus)
	EventSignal EventType = "signal"

	// Debug events (from Debug API)
	EventBreakpointHit EventType = "breakpoint_hit"
	EventPaused        EventType = "paused"
	EventResumed       EventType = "resumed"
)

// Event is the unified event type for all kami consumers (SSE, WS, Recorder).
// It normalizes both WalkEvents and Signals into a single stream.
type Event struct {
	Type      EventType      `json:"type"`
	Timestamp time.Time      `json:"ts"`
	Agent     string         `json:"agent,omitempty"`
	Node      string         `json:"node,omitempty"`
	Edge      string         `json:"edge,omitempty"`
	Zone      string         `json:"zone,omitempty"`
	CaseID    string         `json:"case_id,omitempty"`
	ElapsedMs int64          `json:"elapsed_ms,omitempty"`
	Error     string         `json:"error,omitempty"`
	Data      map[string]any `json:"data,omitempty"`
}
