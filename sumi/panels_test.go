package sumi

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dpopsuev/origami/view"
)

type stubPanel struct {
	id        string
	title     string
	focusable bool
	minW      int
	minH      int
}

func (p *stubPanel) ID() string                    { return p.id }
func (p *stubPanel) Title() string                 { return p.title }
func (p *stubPanel) Update(_ tea.Msg) tea.Cmd      { return nil }
func (p *stubPanel) View(_ Rect) string            { return p.id + "-content" }
func (p *stubPanel) Focusable() bool               { return p.focusable }
func (p *stubPanel) PreferredSize() (int, int)     { return p.minW, p.minH }

func TestRect_Contains(t *testing.T) {
	r := Rect{10, 20, 30, 15}

	tests := []struct {
		x, y int
		want bool
	}{
		{10, 20, true},   // top-left corner
		{39, 34, true},   // bottom-right edge
		{25, 27, true},   // center
		{9, 20, false},   // left of rect
		{40, 20, false},  // right of rect
		{10, 19, false},  // above rect
		{10, 35, false},  // below rect
	}
	for _, tt := range tests {
		got := r.Contains(tt.x, tt.y)
		if got != tt.want {
			t.Errorf("Rect(%v).Contains(%d, %d) = %v, want %v", r, tt.x, tt.y, got, tt.want)
		}
	}
}

func TestRect_Inner(t *testing.T) {
	r := Rect{5, 5, 20, 10}
	inner := r.Inner()
	if inner.X != 6 || inner.Y != 6 || inner.W != 18 || inner.H != 8 {
		t.Errorf("Inner() = %v, want {6, 6, 18, 8}", inner)
	}

	tiny := Rect{0, 0, 1, 1}
	tInner := tiny.Inner()
	if tInner.W != 0 || tInner.H != 0 {
		t.Errorf("Inner() of tiny rect should have 0 usable area, got %v", tInner)
	}
}

func TestPanelRegistry_FocusCycle(t *testing.T) {
	reg := NewPanelRegistry()
	reg.Register(&stubPanel{id: "graph", focusable: true}, SlotCenter)
	reg.Register(&stubPanel{id: "status", focusable: false}, SlotTopBar)
	reg.Register(&stubPanel{id: "inspector", focusable: true}, SlotRightSidebar)
	reg.Register(&stubPanel{id: "timeline", focusable: true}, SlotBottom)

	if reg.FocusedID() != "graph" {
		t.Fatalf("initial focus = %q, want graph", reg.FocusedID())
	}

	reg.CycleFocus()
	if reg.FocusedID() != "inspector" {
		t.Errorf("after 1 cycle = %q, want inspector", reg.FocusedID())
	}

	reg.CycleFocus()
	if reg.FocusedID() != "timeline" {
		t.Errorf("after 2 cycles = %q, want timeline", reg.FocusedID())
	}

	reg.CycleFocus()
	if reg.FocusedID() != "graph" {
		t.Errorf("after 3 cycles should wrap to graph, got %q", reg.FocusedID())
	}
}

func TestPanelRegistry_CycleFocusBack(t *testing.T) {
	reg := NewPanelRegistry()
	reg.Register(&stubPanel{id: "a", focusable: true}, SlotCenter)
	reg.Register(&stubPanel{id: "b", focusable: true}, SlotRightSidebar)

	reg.CycleFocusBack()
	if reg.FocusedID() != "b" {
		t.Errorf("back from a should be b, got %q", reg.FocusedID())
	}
}

func TestPanelRegistry_SetFocusByID(t *testing.T) {
	reg := NewPanelRegistry()
	reg.Register(&stubPanel{id: "graph", focusable: true}, SlotCenter)
	reg.Register(&stubPanel{id: "inspector", focusable: true}, SlotRightSidebar)

	ok := reg.SetFocusByID("inspector")
	if !ok {
		t.Fatal("SetFocusByID should succeed for registered focusable panel")
	}
	if reg.FocusedID() != "inspector" {
		t.Errorf("focus = %q, want inspector", reg.FocusedID())
	}

	ok = reg.SetFocusByID("nonexistent")
	if ok {
		t.Error("SetFocusByID should fail for nonexistent panel")
	}
}

func TestPanelRegistry_BySlot(t *testing.T) {
	reg := NewPanelRegistry()
	p := &stubPanel{id: "graph", focusable: true}
	reg.Register(p, SlotCenter)

	got := reg.BySlot(SlotCenter)
	if got == nil || got.ID() != "graph" {
		t.Errorf("BySlot(Center) = %v, want graph", got)
	}

	got = reg.BySlot(SlotLeftSidebar)
	if got != nil {
		t.Errorf("BySlot(LeftSidebar) should be nil, got %v", got)
	}
}

func TestPanelRegistry_ByID(t *testing.T) {
	reg := NewPanelRegistry()
	reg.Register(&stubPanel{id: "x"}, SlotCenter)

	if reg.ByID("x") == nil {
		t.Error("ByID should find registered panel")
	}
	if reg.ByID("y") != nil {
		t.Error("ByID should return nil for unregistered ID")
	}
}

func TestPanelRegistry_PanelAtPoint(t *testing.T) {
	reg := NewPanelRegistry()
	reg.Register(&stubPanel{id: "graph", focusable: true}, SlotCenter)
	reg.Register(&stubPanel{id: "inspector", focusable: true}, SlotRightSidebar)

	layout := WarRoomLayout{
		Center:       Rect{0, 1, 80, 30},
		RightSidebar: Rect{80, 1, 30, 30},
		Width:        110,
		Height:       40,
		Tier:         TierStandard,
	}

	p := reg.PanelAtPoint(40, 15, layout)
	if p == nil || p.ID() != "graph" {
		t.Errorf("click in center should hit graph, got %v", p)
	}

	p = reg.PanelAtPoint(90, 15, layout)
	if p == nil || p.ID() != "inspector" {
		t.Errorf("click in right sidebar should hit inspector, got %v", p)
	}

	p = reg.PanelAtPoint(0, 0, layout)
	if p != nil {
		t.Errorf("click in top bar (no panel registered) should be nil, got %v", p)
	}
}

func TestComputeLayout_Tiers(t *testing.T) {
	tests := []struct {
		w, h int
		tier LayoutTier
	}{
		{80, 24, TierMinimal},
		{100, 24, TierCompact},
		{120, 30, TierStandard},
		{140, 40, TierFull},
		{200, 60, TierFull},
		{60, 20, TierMinimal},
	}
	for _, tt := range tests {
		l := ComputeLayout(tt.w, tt.h)
		if l.Tier != tt.tier {
			t.Errorf("ComputeLayout(%d, %d).Tier = %d, want %d", tt.w, tt.h, l.Tier, tt.tier)
		}
	}
}

func TestComputeLayout_MinimalHasGraphOnly(t *testing.T) {
	l := ComputeLayout(80, 24)
	if l.TopBar.H != 1 {
		t.Errorf("top bar height = %d, want 1", l.TopBar.H)
	}
	if l.Center.W != 80 || l.Center.H != 23 {
		t.Errorf("center = %dx%d, want 80x23", l.Center.W, l.Center.H)
	}
	if l.LeftSidebar.W != 0 {
		t.Error("minimal should have no left sidebar")
	}
	if l.RightSidebar.W != 0 {
		t.Error("minimal should have no right sidebar")
	}
	if l.Bottom.H != 0 {
		t.Error("minimal should have no bottom panel")
	}
}

func TestComputeLayout_FullHasAllPanels(t *testing.T) {
	l := ComputeLayout(160, 50)
	if l.LeftSidebar.W == 0 {
		t.Error("full should have left sidebar")
	}
	if l.RightSidebar.W == 0 {
		t.Error("full should have right sidebar")
	}
	if l.Bottom.H == 0 {
		t.Error("full should have bottom panel")
	}
	if l.Center.W == 0 {
		t.Error("full should have center panel")
	}

	totalW := l.LeftSidebar.W + l.Center.W + l.RightSidebar.W
	if totalW != 160 {
		t.Errorf("panel widths sum to %d, want 160", totalW)
	}
}

func TestComputeLayout_NoPanelOverlaps(t *testing.T) {
	for _, size := range [][2]int{{80, 24}, {100, 24}, {120, 30}, {140, 40}, {200, 60}} {
		l := ComputeLayout(size[0], size[1])
		rects := []Rect{l.TopBar, l.LeftSidebar, l.Center, l.RightSidebar, l.Bottom, l.BottomTabs}
		for i := 0; i < len(rects); i++ {
			a := rects[i]
			if a.W == 0 || a.H == 0 {
				continue
			}
			for j := i + 1; j < len(rects); j++ {
				b := rects[j]
				if b.W == 0 || b.H == 0 {
					continue
				}
				if rectsOverlap(a, b) {
					t.Errorf("at %dx%d: slot %d (%v) overlaps slot %d (%v)",
						size[0], size[1], i, a, j, b)
				}
			}
		}
	}
}

func rectsOverlap(a, b Rect) bool {
	return a.X < b.X+b.W && b.X < a.X+a.W && a.Y < b.Y+b.H && b.Y < a.Y+a.H
}

func TestComputeLayout_RectFor(t *testing.T) {
	l := ComputeLayout(140, 40)
	if l.RectFor(SlotCenter) != l.Center {
		t.Error("RectFor(SlotCenter) should return Center")
	}
	if l.RectFor(SlotTopBar) != l.TopBar {
		t.Error("RectFor(SlotTopBar) should return TopBar")
	}
}

func TestHitMap_GraphNodes(t *testing.T) {
	def := testCircuit()
	store := view.NewCircuitStore(def)
	defer store.Close()

	engine := &view.GridLayout{}
	layout, _ := engine.Layout(def)

	_, hitMap := RenderGraphWithHitMap(def, layout, store.Snapshot(), RenderOpts{NoColor: true})

	if hitMap == nil {
		t.Fatal("hitMap should not be nil")
	}

	for _, nd := range def.Nodes {
		gc := layout.Grid[nd.Name]
		x, y := cellOrigin(gc)
		name, ok := hitMap[[2]int{x + 1, y + 1}]
		if !ok || name != nd.Name {
			t.Errorf("hitMap[%d,%d] = %q, want %q", x+1, y+1, name, nd.Name)
		}
	}

	_, exists := hitMap[[2]int{0, 0}]
	if exists {
		t.Error("position (0,0) should not map to any node")
	}
}

func TestDispatchMouse_HitsGraphNode(t *testing.T) {
	graphRect := Rect{10, 2, 80, 30}
	hitMap := map[[2]int]string{
		{5, 3}: "recall",
	}

	msg := tea.MouseMsg{X: 15, Y: 5, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress}
	target := DispatchMouse(msg, WarRoomLayout{Center: graphRect}, NewPanelRegistry(), hitMap, graphRect)
	if target.Node != "recall" {
		t.Errorf("target.Node = %q, want recall", target.Node)
	}
}

func TestDispatchMouse_MissesOutsideGraph(t *testing.T) {
	graphRect := Rect{10, 2, 80, 30}
	hitMap := map[[2]int]string{
		{5, 3}: "recall",
	}

	msg := tea.MouseMsg{X: 5, Y: 5, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress}
	target := DispatchMouse(msg, WarRoomLayout{}, NewPanelRegistry(), hitMap, graphRect)
	if target.Node != "" {
		t.Errorf("click outside graph should not hit node, got %q", target.Node)
	}
}
