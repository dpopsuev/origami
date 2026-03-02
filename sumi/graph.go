package sumi

import (
	"fmt"
	"sort"
	"strings"

	framework "github.com/dpopsuev/origami"
	"github.com/dpopsuev/origami/view"
	"github.com/charmbracelet/lipgloss"
)

const (
	nodeWidth  = 20
	nodeHeight = 3
	cellGapH   = 4
	cellGapV   = 1
)

// RenderGraph produces a static box-drawing representation of a circuit.
// It uses GridLayout positions for cell placement, draws zone borders,
// node rectangles with D/S badges, element colors, and edge lines.
func RenderGraph(def *framework.CircuitDef, layout view.CircuitLayout, snap view.CircuitSnapshot, opts RenderOpts) string {
	if layout.Grid == nil || len(layout.Grid) == 0 {
		return "(empty circuit)"
	}

	maxRow, maxCol := gridBounds(layout.Grid)
	canvasW := (maxCol+1)*(nodeWidth+cellGapH) + cellGapH
	canvasH := (maxRow+1)*(nodeHeight+cellGapV) + cellGapV

	canvas := newCanvas(canvasW, canvasH)

	drawZones(canvas, def, layout, opts)
	drawEdges(canvas, def, layout, opts)
	drawNodes(canvas, def, layout, snap, opts)

	return canvas.Render(opts)
}

func gridBounds(grid map[string]view.GridCell) (maxRow, maxCol int) {
	for _, gc := range grid {
		if gc.Row > maxRow {
			maxRow = gc.Row
		}
		if gc.Col > maxCol {
			maxCol = gc.Col
		}
	}
	return
}

func cellOrigin(gc view.GridCell) (x, y int) {
	x = gc.Col*(nodeWidth+cellGapH) + cellGapH
	y = gc.Row*(nodeHeight+cellGapV) + cellGapV
	return
}

// canvas is a 2D character buffer for compositing graph elements.
type canvas struct {
	cells  [][]rune
	styles [][]lipgloss.Style
	width  int
	height int
}

func newCanvas(w, h int) *canvas {
	cells := make([][]rune, h)
	styles := make([][]lipgloss.Style, h)
	for i := range cells {
		cells[i] = make([]rune, w)
		styles[i] = make([]lipgloss.Style, w)
		for j := range cells[i] {
			cells[i][j] = ' '
		}
	}
	return &canvas{cells: cells, styles: styles, width: w, height: h}
}

func (c *canvas) set(x, y int, ch rune, style lipgloss.Style) {
	if y >= 0 && y < c.height && x >= 0 && x < c.width {
		c.cells[y][x] = ch
		c.styles[y][x] = style
	}
}

func (c *canvas) putString(x, y int, s string, style lipgloss.Style) {
	col := 0
	for _, ch := range s {
		c.set(x+col, y, ch, style)
		col++
	}
}

func runeLen(s string) int {
	n := 0
	for range s {
		n++
	}
	return n
}

func truncateRunes(s string, max int) string {
	i := 0
	for pos := range s {
		if i >= max {
			return s[:pos]
		}
		i++
	}
	return s
}

func (c *canvas) Render(opts RenderOpts) string {
	var sb strings.Builder
	for y := 0; y < c.height; y++ {
		line := c.renderLine(y, opts)
		sb.WriteString(strings.TrimRight(line, " "))
		sb.WriteByte('\n')
	}
	return sb.String()
}

func (c *canvas) renderLine(y int, opts RenderOpts) string {
	if opts.NoColor {
		return string(c.cells[y])
	}
	var sb strings.Builder
	for x := 0; x < c.width; x++ {
		ch := string(c.cells[y][x])
		st := c.styles[y][x]
		sb.WriteString(st.Render(ch))
	}
	return sb.String()
}

// RenderOpts controls rendering behavior.
type RenderOpts struct {
	NoColor bool
	Compact bool
	Width   int
}

// --- Node drawing ---

func drawNodes(c *canvas, def *framework.CircuitDef, layout view.CircuitLayout, snap view.CircuitSnapshot, opts RenderOpts) {
	nodeMap := make(map[string]*framework.NodeDef, len(def.Nodes))
	for i := range def.Nodes {
		nodeMap[def.Nodes[i].Name] = &def.Nodes[i]
	}

	for name, gc := range layout.Grid {
		nd := nodeMap[name]
		if nd == nil {
			continue
		}
		ns := snap.Nodes[name]
		x, y := cellOrigin(gc)
		drawNode(c, x, y, nd, ns, snap, opts)
	}
}

func drawNode(c *canvas, x, y int, nd *framework.NodeDef, ns view.NodeState, snap view.CircuitSnapshot, opts RenderOpts) {
	style := nodeStyle(ns, opts)
	elemStyle := ElementFg(nd.Element)
	if opts.NoColor {
		elemStyle = lipgloss.NewStyle()
		style = lipgloss.NewStyle()
	}

	badge := DSBadge(nd.Transformer)
	stateIcon := stateIndicator(ns.State)

	// Walker marker
	walkerMark := ""
	for _, wp := range snap.Walkers {
		if wp.Node == nd.Name {
			walkerMark = "●"
			break
		}
	}

	// Breakpoint marker
	bpMark := ""
	if snap.Breakpoints[nd.Name] {
		bpMark = "◉"
	}

	// Top border
	topBorder := "┌" + strings.Repeat("─", nodeWidth-2) + "┐"
	c.putString(x, y, topBorder, style)

	// Label line: "[D] name ● ✓"
	label := nd.Name
	if badge != "" {
		label = badge + " " + label
	}
	prefix := ""
	if bpMark != "" {
		prefix = bpMark + " "
	}
	suffix := ""
	if walkerMark != "" {
		suffix += " " + walkerMark
	}
	if stateIcon != "" {
		suffix += " " + stateIcon
	}

	content := prefix + label + suffix
	inner := nodeWidth - 2
	contentLen := runeLen(content)
	if contentLen > inner {
		content = truncateRunes(content, inner)
		contentLen = inner
	}
	padded := content + strings.Repeat(" ", inner-contentLen)

	c.set(x, y+1, '│', style)
	if !opts.NoColor {
		c.putString(x+1, y+1, padded, elemStyle)
	} else {
		c.putString(x+1, y+1, padded, lipgloss.NewStyle())
	}
	c.set(x+nodeWidth-1, y+1, '│', style)

	// Bottom border
	bottomBorder := "└" + strings.Repeat("─", nodeWidth-2) + "┘"
	c.putString(x, y+2, bottomBorder, style)
}

func nodeStyle(ns view.NodeState, opts RenderOpts) lipgloss.Style {
	if opts.NoColor {
		return lipgloss.NewStyle()
	}
	switch ns.State {
	case view.NodeActive:
		return StyleActive
	case view.NodeCompleted:
		return StyleCompleted
	case view.NodeError:
		return StyleError
	default:
		return StyleIdle
	}
}

func stateIndicator(state view.NodeVisualState) string {
	switch state {
	case view.NodeActive:
		return "▶"
	case view.NodeCompleted:
		return "✓"
	case view.NodeError:
		return "✗"
	default:
		return ""
	}
}

// --- Edge drawing ---

func drawEdges(c *canvas, def *framework.CircuitDef, layout view.CircuitLayout, opts RenderOpts) {
	style := lipgloss.NewStyle().Faint(true)
	if opts.NoColor {
		style = lipgloss.NewStyle()
	}

	for _, edge := range def.Edges {
		fromGC, fromOK := layout.Grid[edge.From]
		toGC, toOK := layout.Grid[edge.To]
		if !fromOK || !toOK {
			continue
		}

		fromX, fromY := cellOrigin(fromGC)
		toX, toY := cellOrigin(toGC)

		edgeStyle := style
		if edge.Shortcut {
			edgeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
		}
		if edge.Loop {
			edgeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("208"))
		}
		if opts.NoColor {
			edgeStyle = lipgloss.NewStyle()
		}

		drawEdgeLine(c, fromX, fromY, toX, toY, edge, edgeStyle, opts)
	}
}

func drawEdgeLine(c *canvas, fromX, fromY, toX, toY int, edge framework.EdgeDef, style lipgloss.Style, opts RenderOpts) {
	startX := fromX + nodeWidth
	startY := fromY + nodeHeight/2
	endX := toX
	endY := toY + nodeHeight/2

	if edge.Loop && endX <= startX {
		drawLoopEdge(c, startX, startY, fromX, fromY, style)
		return
	}

	ch := '─'
	if edge.Shortcut {
		ch = '╌'
	}

	if startY == endY {
		for x := startX; x < endX; x++ {
			c.set(x, startY, ch, style)
		}
		c.set(endX-1, startY, '▸', style)
	} else {
		midX := (startX + endX) / 2
		for x := startX; x <= midX; x++ {
			c.set(x, startY, ch, style)
		}
		if endY > startY {
			c.set(midX, startY, '┐', style)
			for y := startY + 1; y < endY; y++ {
				c.set(midX, y, '│', style)
			}
			c.set(midX, endY, '└', style)
		} else {
			c.set(midX, startY, '┘', style)
			for y := endY + 1; y < startY; y++ {
				c.set(midX, y, '│', style)
			}
			c.set(midX, endY, '┌', style)
		}
		for x := midX + 1; x < endX; x++ {
			c.set(x, endY, ch, style)
		}
		c.set(endX-1, endY, '▸', style)
	}
}

func drawLoopEdge(c *canvas, startX, startY, nodeX, nodeY int, style lipgloss.Style) {
	loopY := nodeY + nodeHeight + 1
	if loopY >= c.height {
		loopY = c.height - 1
	}
	c.set(startX, startY, '┐', style)
	for y := startY + 1; y <= loopY; y++ {
		c.set(startX, y, '│', style)
	}
	c.set(startX, loopY, '┘', style)
	for x := nodeX; x < startX; x++ {
		c.set(x, loopY, '─', style)
	}
	c.set(nodeX, loopY, '◀', style)
}

// --- Zone drawing ---

type zoneBounds struct {
	name    string
	element string
	minRow  int
	maxRow  int
	minCol  int
	maxCol  int
}

func drawZones(c *canvas, def *framework.CircuitDef, layout view.CircuitLayout, opts RenderOpts) {
	zones := make(map[string]*zoneBounds)

	sortedZoneNames := make([]string, 0, len(def.Zones))
	for name := range def.Zones {
		sortedZoneNames = append(sortedZoneNames, name)
	}
	sort.Strings(sortedZoneNames)

	for _, zoneName := range sortedZoneNames {
		zd := def.Zones[zoneName]
		for _, nodeName := range zd.Nodes {
			gc, ok := layout.Grid[nodeName]
			if !ok {
				continue
			}
			zb, exists := zones[zoneName]
			if !exists {
				zb = &zoneBounds{
					name: zoneName, element: zd.Element,
					minRow: gc.Row, maxRow: gc.Row,
					minCol: gc.Col, maxCol: gc.Col,
				}
				zones[zoneName] = zb
			}
			if gc.Row < zb.minRow {
				zb.minRow = gc.Row
			}
			if gc.Row > zb.maxRow {
				zb.maxRow = gc.Row
			}
			if gc.Col < zb.minCol {
				zb.minCol = gc.Col
			}
			if gc.Col > zb.maxCol {
				zb.maxCol = gc.Col
			}
		}
	}

	for _, zb := range zones {
		drawZoneBorder(c, zb, opts)
	}
}

func drawZoneBorder(c *canvas, zb *zoneBounds, opts RenderOpts) {
	x1, y1 := cellOrigin(view.GridCell{Row: zb.minRow, Col: zb.minCol})
	x2, y2 := cellOrigin(view.GridCell{Row: zb.maxRow, Col: zb.maxCol})

	x1 -= 1
	y1 -= 1
	x2 += nodeWidth
	y2 += nodeHeight

	style := StyleZoneBorder
	if !opts.NoColor {
		if ec, ok := ElementColor[zb.element]; ok {
			style = lipgloss.NewStyle().Faint(true).Foreground(ec)
		}
	} else {
		style = lipgloss.NewStyle()
	}

	// Top + bottom
	for x := x1; x <= x2; x++ {
		c.set(x, y1, '─', style)
		c.set(x, y2, '─', style)
	}
	// Left + right
	for y := y1; y <= y2; y++ {
		c.set(x1, y, '│', style)
		c.set(x2, y, '│', style)
	}
	// Corners
	c.set(x1, y1, '┌', style)
	c.set(x2, y1, '┐', style)
	c.set(x1, y2, '└', style)
	c.set(x2, y2, '┘', style)

	// Zone label
	label := fmt.Sprintf(" %s ", zb.name)
	c.putString(x1+2, y1, label, style)
}
