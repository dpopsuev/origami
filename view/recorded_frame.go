package view

import "time"

// RecordedFrame is a captured snapshot of Sumi's rendered TUI output.
// The ViewText is rendered with NoColor=true so it contains no ANSI escapes.
type RecordedFrame struct {
	Timestamp    time.Time `json:"timestamp"`
	Width        int       `json:"width"`
	Height       int       `json:"height"`
	LayoutTier   string    `json:"layout_tier"`
	SelectedNode string    `json:"selected_node,omitempty"`
	FocusedPanel string    `json:"focused_panel,omitempty"`
	WorkerCount  int       `json:"worker_count"`
	EventCount   int       `json:"event_count"`
	ViewText     string    `json:"view_text"`
}
