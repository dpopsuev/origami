package sumi

import (
	"fmt"
	"strings"

	framework "github.com/dpopsuev/origami"
	"github.com/dpopsuev/origami/view"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// DiffMsg wraps a StateDiff for delivery into the Bubble Tea update loop.
type DiffMsg view.StateDiff

// Model is the Bubble Tea model for Sumi.
// It holds the circuit definition, layout, store subscription, and UI state.
type Model struct {
	def     *framework.CircuitDef
	store   *view.CircuitStore
	layout  view.CircuitLayout
	snap    view.CircuitSnapshot
	opts    RenderOpts
	subID   int
	subCh   <-chan view.StateDiff
	width   int
	height  int
	ready   bool

	// Interactive state
	selected   int
	nodeOrder  []string
	inspecting bool
	searching  bool
	searchBuf  string

	// Debug state
	debug      *DebugClient
	debugAvail bool

	// Kami agent state
	kamiStatus  KamiStatus
	chatOpen    bool
	chatMsgs    []ChatMessage
	chatInput   string
}

// KamiStatus represents the Kami MCP connection state.
type KamiStatus int

const (
	KamiOffline   KamiStatus = iota
	KamiConnected
)

// ChatMessage is a single message in the agent chat panel.
type ChatMessage struct {
	Role    string // "user", "agent", "system"
	Content string
}

// Config holds initialization parameters for a Sumi Model.
type Config struct {
	Def    *framework.CircuitDef
	Store  *view.CircuitStore
	Layout view.CircuitLayout
	Opts   RenderOpts
	Debug  *DebugClient
}

// New creates a Sumi Model ready for Bubble Tea.
func New(cfg Config) Model {
	snap := cfg.Store.Snapshot()

	order := make([]string, 0, len(cfg.Def.Nodes))
	for _, nd := range cfg.Def.Nodes {
		order = append(order, nd.Name)
	}

	m := Model{
		def:       cfg.Def,
		store:     cfg.Store,
		layout:    cfg.Layout,
		snap:      snap,
		opts:      cfg.Opts,
		nodeOrder: order,
		debug:     cfg.Debug,
	}
	return m
}

// Init subscribes to the CircuitStore diff channel.
func (m Model) Init() tea.Cmd {
	m.subID, m.subCh = m.store.Subscribe()
	return waitForDiff(m.subCh)
}

func waitForDiff(ch <-chan view.StateDiff) tea.Cmd {
	return func() tea.Msg {
		diff, ok := <-ch
		if !ok {
			return tea.Quit()
		}
		return DiffMsg(diff)
	}
}

// Update processes Bubble Tea messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		return m, nil

	case DiffMsg:
		diff := view.StateDiff(msg)
		m.applyDiff(diff)
		m.snap = m.store.Snapshot()
		return m, waitForDiff(m.subCh)

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m *Model) applyDiff(diff view.StateDiff) {
	// The store already updates the snapshot; we just track
	// extra UI state here if needed (e.g., auto-select active node).
	if diff.Type == view.DiffNodeState && diff.State == view.NodeActive {
		for i, name := range m.nodeOrder {
			if name == diff.Node {
				m.selected = i
				break
			}
		}
	}
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.searching {
		return m.handleSearchKey(msg)
	}
	if m.chatOpen {
		return m.handleChatKey(msg)
	}

	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "tab":
		m.selected = (m.selected + 1) % len(m.nodeOrder)
		return m, nil

	case "shift+tab":
		m.selected = (m.selected - 1 + len(m.nodeOrder)) % len(m.nodeOrder)
		return m, nil

	case "up":
		m.selected = m.findAdjacentNode("up")
		return m, nil
	case "down":
		m.selected = m.findAdjacentNode("down")
		return m, nil
	case "left":
		m.selected = m.findAdjacentNode("left")
		return m, nil
	case "right":
		m.selected = m.findAdjacentNode("right")
		return m, nil

	case "enter":
		m.inspecting = !m.inspecting
		return m, nil

	case "esc":
		if m.inspecting {
			m.inspecting = false
		}
		return m, nil

	case "/":
		m.searching = true
		m.searchBuf = ""
		return m, nil

	case " ":
		return m.toggleBreakpoint()

	case "p":
		return m.pauseWalk()

	case "r":
		return m.resumeWalk()

	case "n":
		return m.stepNode()

	case "c":
		if m.kamiStatus == KamiConnected {
			m.chatOpen = !m.chatOpen
		}
		return m, nil
	}

	return m, nil
}

func (m Model) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.searching = false
		m.searchBuf = ""
		return m, nil
	case "enter":
		m.applySearch()
		m.searching = false
		return m, nil
	case "backspace":
		if len(m.searchBuf) > 0 {
			m.searchBuf = m.searchBuf[:len(m.searchBuf)-1]
		}
		return m, nil
	default:
		if len(msg.String()) == 1 {
			m.searchBuf += msg.String()
		}
		return m, nil
	}
}

func (m *Model) applySearch() {
	if m.searchBuf == "" {
		return
	}
	q := strings.ToLower(m.searchBuf)
	for i, name := range m.nodeOrder {
		if strings.Contains(strings.ToLower(name), q) {
			m.selected = i
			return
		}
	}
	for zoneName, zd := range m.def.Zones {
		if strings.Contains(strings.ToLower(zoneName), q) {
			if len(zd.Nodes) > 0 {
				for i, name := range m.nodeOrder {
					if name == zd.Nodes[0] {
						m.selected = i
						return
					}
				}
			}
		}
	}
}

func (m Model) handleChatKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.chatOpen = false
		return m, nil
	case "enter":
		if m.chatInput != "" {
			m.chatMsgs = append(m.chatMsgs, ChatMessage{Role: "user", Content: m.chatInput})
			m.chatMsgs = append(m.chatMsgs, ChatMessage{Role: "system", Content: "Agent: unavailable (MCP dispatch not connected)"})
			m.chatInput = ""
		}
		return m, nil
	case "backspace":
		if len(m.chatInput) > 0 {
			m.chatInput = m.chatInput[:len(m.chatInput)-1]
		}
		return m, nil
	default:
		if len(msg.String()) == 1 {
			m.chatInput += msg.String()
		}
		return m, nil
	}
}

func (m Model) findAdjacentNode(dir string) int {
	if len(m.nodeOrder) == 0 {
		return 0
	}
	currentName := m.nodeOrder[m.selected]
	currentGC, ok := m.layout.Grid[currentName]
	if !ok {
		return m.selected
	}

	bestIdx := m.selected
	bestDist := 999999

	for i, name := range m.nodeOrder {
		gc, ok := m.layout.Grid[name]
		if !ok || i == m.selected {
			continue
		}

		var match bool
		var dist int
		switch dir {
		case "up":
			match = gc.Row < currentGC.Row
			dist = (currentGC.Row-gc.Row)*10 + abs(gc.Col-currentGC.Col)
		case "down":
			match = gc.Row > currentGC.Row
			dist = (gc.Row-currentGC.Row)*10 + abs(gc.Col-currentGC.Col)
		case "left":
			match = gc.Col < currentGC.Col
			dist = (currentGC.Col-gc.Col)*10 + abs(gc.Row-currentGC.Row)
		case "right":
			match = gc.Col > currentGC.Col
			dist = (gc.Col-currentGC.Col)*10 + abs(gc.Row-currentGC.Row)
		}
		if match && dist < bestDist {
			bestDist = dist
			bestIdx = i
		}
	}
	return bestIdx
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// --- Debug actions ---

func (m Model) toggleBreakpoint() (tea.Model, tea.Cmd) {
	if m.debug == nil {
		return m, nil
	}
	name := m.nodeOrder[m.selected]
	if m.snap.Breakpoints[name] {
		m.debug.ClearBreakpoint(name)
	} else {
		m.debug.SetBreakpoint(name)
	}
	return m, nil
}

func (m Model) pauseWalk() (tea.Model, tea.Cmd) {
	if m.debug != nil {
		m.debug.Pause()
	}
	return m, nil
}

func (m Model) resumeWalk() (tea.Model, tea.Cmd) {
	if m.debug != nil {
		m.debug.Resume()
	}
	return m, nil
}

func (m Model) stepNode() (tea.Model, tea.Cmd) {
	if m.debug != nil {
		m.debug.AdvanceNode()
	}
	return m, nil
}

// View renders the complete Sumi TUI.
func (m Model) View() string {
	if !m.ready {
		return "Loading..."
	}

	// Mark selected node in snapshot (ephemeral, doesn't affect store)
	graph := RenderGraph(m.def, m.layout, m.snap, m.opts)

	var sections []string
	sections = append(sections, graph)

	if m.inspecting && m.selected < len(m.nodeOrder) {
		sections = append(sections, m.renderInspector())
	}

	if m.chatOpen {
		sections = append(sections, m.renderChatPanel())
	}

	sections = append(sections, m.renderStatusBar())

	if m.searching {
		sections = append(sections, m.renderSearchBar())
	}

	return strings.Join(sections, "\n")
}

func (m Model) renderStatusBar() string {
	parts := []string{}

	// Walker progress
	for _, wp := range m.snap.Walkers {
		completed := 0
		for _, ns := range m.snap.Nodes {
			if ns.State == view.NodeCompleted {
				completed++
			}
		}
		total := len(m.snap.Nodes)
		bar := progressBar(completed, total, 10)
		elemStyle := ElementFg(wp.Element)
		if m.opts.NoColor {
			elemStyle = lipgloss.NewStyle()
		}
		parts = append(parts, elemStyle.Render(fmt.Sprintf("Walker: %s @ %s %s %d/%d", wp.WalkerID, wp.Node, bar, completed, total)))
	}

	// Breakpoints
	bps := []string{}
	for name, set := range m.snap.Breakpoints {
		if set {
			bps = append(bps, name)
		}
	}
	if len(bps) > 0 {
		parts = append(parts, fmt.Sprintf("BP: %s", strings.Join(bps, ", ")))
	}

	// Kami status
	switch m.kamiStatus {
	case KamiConnected:
		parts = append(parts, "🟢 Kami: connected")
	default:
		parts = append(parts, "⚫ Kami: offline")
	}

	// Paused / completed / error
	if m.snap.Paused {
		parts = append(parts, "[PAUSED]")
	}
	if m.snap.Completed {
		parts = append(parts, "[DONE]")
	}
	if m.snap.Error != "" {
		parts = append(parts, fmt.Sprintf("[ERROR: %s]", m.snap.Error))
	}

	// Selected node
	if m.selected < len(m.nodeOrder) {
		parts = append(parts, fmt.Sprintf("Selected: %s", m.nodeOrder[m.selected]))
	}

	status := strings.Join(parts, "  │  ")
	if m.opts.NoColor {
		return status
	}
	return StyleStatusBar.Render(status)
}

func (m Model) renderSearchBar() string {
	prompt := fmt.Sprintf("/ %s", m.searchBuf)
	if m.opts.NoColor {
		return prompt
	}
	return StyleSearchBar.Render(prompt)
}

func (m Model) renderInspector() string {
	if m.selected >= len(m.nodeOrder) {
		return ""
	}
	name := m.nodeOrder[m.selected]
	var nd *framework.NodeDef
	for i := range m.def.Nodes {
		if m.def.Nodes[i].Name == name {
			nd = &m.def.Nodes[i]
			break
		}
	}
	if nd == nil {
		return ""
	}

	ns := m.snap.Nodes[name]

	var sb strings.Builder
	sb.WriteString("┌─── Inspector ─────────────────┐\n")
	sb.WriteString(fmt.Sprintf("│ Name:        %-16s │\n", nd.Name))
	sb.WriteString(fmt.Sprintf("│ Element:     %-16s │\n", nd.Element))
	sb.WriteString(fmt.Sprintf("│ State:       %-16s │\n", ns.State))
	if nd.Transformer != "" {
		sb.WriteString(fmt.Sprintf("│ Transformer: %-16s │\n", nd.Transformer))
	}
	if nd.Extractor != "" {
		sb.WriteString(fmt.Sprintf("│ Extractor:   %-16s │\n", nd.Extractor))
	}
	if nd.Family != "" {
		sb.WriteString(fmt.Sprintf("│ Family:      %-16s │\n", nd.Family))
	}
	badge := DSBadge(nd.Transformer)
	if badge != "" {
		sb.WriteString(fmt.Sprintf("│ D/S:         %-16s │\n", badge))
	}
	zone := ns.Zone
	if zone == "" {
		zone = "(none)"
	}
	sb.WriteString(fmt.Sprintf("│ Zone:        %-16s │\n", zone))
	sb.WriteString("└───────────────────────────────┘")
	return sb.String()
}

func (m Model) renderChatPanel() string {
	var sb strings.Builder
	sb.WriteString("┌─── Agent Chat ─────────────────┐\n")
	for _, msg := range m.chatMsgs {
		prefix := ""
		switch msg.Role {
		case "user":
			prefix = "You: "
		case "agent":
			prefix = "Agent: "
		case "system":
			prefix = "  "
		}
		sb.WriteString(fmt.Sprintf("│ %s%s\n", prefix, msg.Content))
	}
	sb.WriteString(fmt.Sprintf("│ > %s\n", m.chatInput))
	sb.WriteString("└────────────────────────────────┘")
	return sb.String()
}

func progressBar(current, total, width int) string {
	if total == 0 {
		return strings.Repeat("░", width)
	}
	filled := (current * width) / total
	if filled > width {
		filled = width
	}
	return "[" + strings.Repeat("█", filled) + strings.Repeat("░", width-filled) + "]"
}
