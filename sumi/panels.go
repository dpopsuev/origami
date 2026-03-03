package sumi

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// Rect describes a panel's position and size in terminal coordinates.
type Rect struct {
	X, Y, W, H int
}

// Contains reports whether the point (px, py) falls inside the rectangle.
func (r Rect) Contains(px, py int) bool {
	return px >= r.X && px < r.X+r.W && py >= r.Y && py < r.Y+r.H
}

// Inner returns the usable area inside a 1-cell border.
func (r Rect) Inner() Rect {
	if r.W < 2 || r.H < 2 {
		return Rect{r.X, r.Y, 0, 0}
	}
	return Rect{r.X + 1, r.Y + 1, r.W - 2, r.H - 2}
}

// Panel is the extension interface for War Room panels.
// Every visual region (graph, inspector, timeline, workers, etc.)
// implements Panel and registers with the PanelRegistry.
type Panel interface {
	ID() string
	Title() string
	Update(tea.Msg) tea.Cmd
	View(area Rect) string
	Focusable() bool
	PreferredSize() (minW, minH int)
}

// Slot identifies a panel's position in the War Room layout.
type Slot int

const (
	SlotTopBar Slot = iota
	SlotLeftSidebar
	SlotCenter
	SlotRightSidebar
	SlotBottom
	SlotBottomTabs
)

// PanelEntry binds a Panel to its layout slot.
type PanelEntry struct {
	Panel Panel
	Slot  Slot
}

// PanelRegistry holds all registered panels and manages focus.
type PanelRegistry struct {
	entries  []PanelEntry
	focusIdx int
}

// NewPanelRegistry creates an empty registry.
func NewPanelRegistry() *PanelRegistry {
	return &PanelRegistry{}
}

// Register adds a panel to the registry at the given slot.
func (r *PanelRegistry) Register(p Panel, slot Slot) {
	r.entries = append(r.entries, PanelEntry{Panel: p, Slot: slot})
}

// BySlot returns the panel at the given slot, or nil.
func (r *PanelRegistry) BySlot(slot Slot) Panel {
	for _, e := range r.entries {
		if e.Slot == slot {
			return e.Panel
		}
	}
	return nil
}

// ByID returns the panel with the given ID, or nil.
func (r *PanelRegistry) ByID(id string) Panel {
	for _, e := range r.entries {
		if e.Panel.ID() == id {
			return e.Panel
		}
	}
	return nil
}

// Focused returns the currently focused panel, or nil if none are focusable.
func (r *PanelRegistry) Focused() Panel {
	focusable := r.focusableEntries()
	if len(focusable) == 0 {
		return nil
	}
	idx := r.focusIdx % len(focusable)
	return focusable[idx].Panel
}

// FocusedID returns the ID of the focused panel, or empty string.
func (r *PanelRegistry) FocusedID() string {
	if p := r.Focused(); p != nil {
		return p.ID()
	}
	return ""
}

// CycleFocus moves focus to the next focusable panel.
func (r *PanelRegistry) CycleFocus() {
	focusable := r.focusableEntries()
	if len(focusable) == 0 {
		return
	}
	r.focusIdx = (r.focusIdx + 1) % len(focusable)
}

// CycleFocusBack moves focus to the previous focusable panel.
func (r *PanelRegistry) CycleFocusBack() {
	focusable := r.focusableEntries()
	if len(focusable) == 0 {
		return
	}
	r.focusIdx = (r.focusIdx - 1 + len(focusable)) % len(focusable)
}

// SetFocusByID sets focus to the panel with the given ID.
// Returns true if the panel was found and is focusable.
func (r *PanelRegistry) SetFocusByID(id string) bool {
	focusable := r.focusableEntries()
	for i, e := range focusable {
		if e.Panel.ID() == id {
			r.focusIdx = i
			return true
		}
	}
	return false
}

// PanelAtPoint returns the panel whose layout rect contains (x, y),
// given the current layout. Returns nil if no panel is hit.
func (r *PanelRegistry) PanelAtPoint(x, y int, layout WarRoomLayout) Panel {
	for _, e := range r.entries {
		rect := layout.RectFor(e.Slot)
		if rect.W > 0 && rect.H > 0 && rect.Contains(x, y) {
			return e.Panel
		}
	}
	return nil
}

func (r *PanelRegistry) focusableEntries() []PanelEntry {
	var out []PanelEntry
	for _, e := range r.entries {
		if e.Panel.Focusable() {
			out = append(out, e)
		}
	}
	return out
}

// All returns all registered entries.
func (r *PanelRegistry) All() []PanelEntry {
	return r.entries
}

// WarRoomLayout holds the computed rectangles for each slot.
type WarRoomLayout struct {
	TopBar       Rect
	LeftSidebar  Rect
	Center       Rect
	RightSidebar Rect
	Bottom       Rect
	BottomTabs   Rect

	Width  int
	Height int
	Tier   LayoutTier
}

// RectFor returns the rect for a given slot.
func (l WarRoomLayout) RectFor(s Slot) Rect {
	switch s {
	case SlotTopBar:
		return l.TopBar
	case SlotLeftSidebar:
		return l.LeftSidebar
	case SlotCenter:
		return l.Center
	case SlotRightSidebar:
		return l.RightSidebar
	case SlotBottom:
		return l.Bottom
	case SlotBottomTabs:
		return l.BottomTabs
	default:
		return Rect{}
	}
}

// LayoutTier describes the responsive breakpoint.
type LayoutTier int

const (
	TierMinimal  LayoutTier = iota // 80x24: graph + status bar only
	TierCompact                    // 100x24: graph + status + bottom timeline
	TierStandard                   // 120x30: graph + inspector + timeline
	TierFull                       // 140x40+: full war room
)

const (
	topBarHeight     = 1
	sidebarMinW      = 20
	sidebarPreferW   = 24
	inspectorMinW    = 28
	inspectorPreferW = 32
	bottomMinH       = 5
	bottomPreferH    = 8
	bottomTabsH      = 3
)

// ComputeLayout computes the War Room panel layout for the given terminal size.
func ComputeLayout(width, height int) WarRoomLayout {
	tier := classifyTier(width, height)
	l := WarRoomLayout{Width: width, Height: height, Tier: tier}

	l.TopBar = Rect{0, 0, width, topBarHeight}
	remaining := height - topBarHeight

	switch tier {
	case TierMinimal:
		l.Center = Rect{0, topBarHeight, width, remaining}

	case TierCompact:
		bh := clamp(remaining/4, bottomMinH, bottomPreferH)
		l.Center = Rect{0, topBarHeight, width, remaining - bh}
		l.Bottom = Rect{0, topBarHeight + remaining - bh, width, bh}

	case TierStandard:
		bh := clamp(remaining/4, bottomMinH, bottomPreferH)
		mainH := remaining - bh
		rw := clamp(width/4, inspectorMinW, inspectorPreferW)
		l.Center = Rect{0, topBarHeight, width - rw, mainH}
		l.RightSidebar = Rect{width - rw, topBarHeight, rw, mainH}
		l.Bottom = Rect{0, topBarHeight + mainH, width, bh}

	case TierFull:
		bh := clamp(remaining/4, bottomMinH, bottomPreferH)
		tabH := bottomTabsH
		mainH := remaining - bh - tabH
		if mainH < 6 {
			mainH = remaining - bh
			tabH = 0
		}
		lw := clamp(width/6, sidebarMinW, sidebarPreferW)
		rw := clamp(width/4, inspectorMinW, inspectorPreferW)
		centerW := width - lw - rw
		if centerW < 30 {
			lw = 0
			centerW = width - rw
		}
		l.LeftSidebar = Rect{0, topBarHeight, lw, mainH}
		l.Center = Rect{lw, topBarHeight, centerW, mainH}
		l.RightSidebar = Rect{lw + centerW, topBarHeight, rw, mainH}
		l.Bottom = Rect{0, topBarHeight + mainH, width, bh}
		if tabH > 0 {
			l.BottomTabs = Rect{0, topBarHeight + mainH + bh, width, tabH}
		}
	}
	return l
}

func classifyTier(w, h int) LayoutTier {
	switch {
	case w >= 140 && h >= 40:
		return TierFull
	case w >= 120 && h >= 30:
		return TierStandard
	case w >= 100 && h >= 24:
		return TierCompact
	default:
		return TierMinimal
	}
}

func clamp(val, min, max int) int {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}

// --- Panel border rendering ---

var (
	styleBorderFocused = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("33"))

	styleBorderUnfocused = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("240"))
)

// RenderPanelFrame wraps content in a bordered box at the given rect.
// Focused panels get a bright border; unfocused get a dim border.
func RenderPanelFrame(title, content string, rect Rect, focused bool, noColor bool) string {
	if rect.W < 2 || rect.H < 2 {
		return ""
	}
	innerW := rect.W - 2
	innerH := rect.H - 2
	if innerW < 0 || innerH < 0 {
		return ""
	}

	lines := strings.Split(content, "\n")
	for len(lines) < innerH {
		lines = append(lines, "")
	}
	if len(lines) > innerH {
		lines = lines[:innerH]
	}
	for i, line := range lines {
		lines[i] = padOrTruncate(line, innerW)
	}

	style := styleBorderUnfocused
	if focused {
		style = styleBorderFocused
	}
	if noColor {
		style = lipgloss.NewStyle().Border(lipgloss.RoundedBorder())
	}

	rendered := style.
		Width(innerW).
		Height(innerH).
		Render(strings.Join(lines, "\n"))

	if title != "" {
		renderedLines := strings.Split(rendered, "\n")
		if len(renderedLines) > 0 {
			titleText := " " + title + " "
			if !noColor {
				titleText = lipgloss.NewStyle().Bold(true).Render(titleText)
			}
			titleW := lipgloss.Width(titleText)
			borderW := rect.W - 2 - titleW
			if borderW < 0 {
				borderW = 0
			}

			borderFg := style.GetBorderTopForeground()
			border := lipgloss.RoundedBorder()
			fgStyle := lipgloss.NewStyle().Foreground(borderFg)

			if noColor {
				renderedLines[0] = border.TopLeft + titleText +
					strings.Repeat(border.Top, borderW) + border.TopRight
			} else {
				renderedLines[0] = fgStyle.Render(border.TopLeft) + titleText +
					fgStyle.Render(strings.Repeat(border.Top, borderW)+border.TopRight)
			}
		}
		rendered = strings.Join(renderedLines, "\n")
	}

	return rendered
}

func padOrTruncate(s string, width int) string {
	w := lipgloss.Width(s)
	if w > width {
		return ansi.Truncate(s, width, "")
	}
	return s + strings.Repeat(" ", width-w)
}
