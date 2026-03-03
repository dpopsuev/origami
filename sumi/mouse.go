package sumi

import tea "github.com/charmbracelet/bubbletea"

// MouseTarget identifies what the mouse hit.
type MouseTarget struct {
	PanelID string
	Node    string // non-empty if a graph node was clicked
}

// DispatchMouse determines which panel and graph element (if any) a mouse
// event targets, given the current layout and graph hit-map.
func DispatchMouse(msg tea.MouseMsg, layout WarRoomLayout, registry *PanelRegistry, graphHitMap map[[2]int]string, graphRect Rect) MouseTarget {
	x, y := msg.X, msg.Y
	target := MouseTarget{}

	if p := registry.PanelAtPoint(x, y, layout); p != nil {
		target.PanelID = p.ID()
	}

	if graphRect.Contains(x, y) && graphHitMap != nil {
		localX := x - graphRect.X
		localY := y - graphRect.Y
		if name, ok := graphHitMap[[2]int{localX, localY}]; ok {
			target.Node = name
		}
	}

	return target
}

// IsClick returns true for left button press events.
func IsClick(msg tea.MouseMsg) bool {
	return msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress
}
