package sumi

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dpopsuev/origami/view"
)

// WorkersPanel implements Panel for the worker sidebar.
type WorkersPanel struct {
	snap     *view.CircuitSnapshot
	selected int // -1 = "All", 0..N = worker index
	workers  []string
	noColor  bool

	onSelect func(workerID string)
}

// NewWorkersPanel creates a worker sidebar panel.
// The onSelect callback fires when a worker is selected (empty string = "All").
func NewWorkersPanel(snap *view.CircuitSnapshot, noColor bool, onSelect func(string)) *WorkersPanel {
	return &WorkersPanel{
		snap:     snap,
		selected: -1,
		noColor:  noColor,
		onSelect: onSelect,
	}
}

func (p *WorkersPanel) ID() string                { return "workers" }
func (p *WorkersPanel) Title() string             { return "Workers" }
func (p *WorkersPanel) Focusable() bool           { return true }
func (p *WorkersPanel) PreferredSize() (int, int) { return sidebarMinW, 6 }

// Refresh updates the worker list from the current snapshot.
func (p *WorkersPanel) Refresh() {
	p.workers = p.workers[:0]
	for id := range p.snap.Walkers {
		p.workers = append(p.workers, id)
	}
	sort.Strings(p.workers)
}

// SelectedWorker returns the currently selected worker ID, or "" for "All".
func (p *WorkersPanel) SelectedWorker() string {
	if p.selected < 0 || p.selected >= len(p.workers) {
		return ""
	}
	return p.workers[p.selected]
}

func (p *WorkersPanel) Update(msg tea.Msg) tea.Cmd {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return nil
	}
	switch km.String() {
	case "up":
		p.selected--
		if p.selected < -1 {
			p.selected = len(p.workers) - 1
		}
		p.fireSelect()
	case "down":
		p.selected++
		if p.selected >= len(p.workers) {
			p.selected = -1
		}
		p.fireSelect()
	case "enter":
		p.fireSelect()
	}
	return nil
}

func (p *WorkersPanel) fireSelect() {
	if p.onSelect != nil {
		p.onSelect(p.SelectedWorker())
	}
}

func (p *WorkersPanel) View(area Rect) string {
	p.Refresh()
	inner := area.Inner()
	if inner.W <= 0 || inner.H <= 0 {
		return ""
	}

	var sb strings.Builder

	allStyle := lipgloss.NewStyle()
	if p.selected == -1 {
		allStyle = allStyle.Bold(true).Reverse(true)
	}
	if p.noColor {
		allStyle = lipgloss.NewStyle()
		if p.selected == -1 {
			allStyle = allStyle.Reverse(true)
		}
	}
	sb.WriteString(allStyle.Render(padOrTruncate("  All", inner.W)))
	sb.WriteByte('\n')

	for i, wid := range p.workers {
		wp := p.snap.Walkers[wid]
		indicator := "●"
		label := fmt.Sprintf(" %s %s", indicator, wid)
		if wp.Node != "" {
			label = fmt.Sprintf(" %s %s @ %s", indicator, wid, wp.Node)
		}

		style := lipgloss.NewStyle()
		if !p.noColor {
			style = ElementFg(wp.Element)
		}
		if i == p.selected {
			style = style.Bold(true).Reverse(true)
		}

		sb.WriteString(style.Render(padOrTruncate(label, inner.W)))
		if i < len(p.workers)-1 {
			sb.WriteByte('\n')
		}
	}

	return sb.String()
}

// SelectByClick selects the worker entry at the given local Y offset within the panel content.
// Returns the selected worker ID ("" for All).
func (p *WorkersPanel) SelectByClick(localY int) string {
	p.Refresh()
	if localY <= 0 {
		p.selected = -1
	} else {
		idx := localY - 1
		if idx < len(p.workers) {
			p.selected = idx
		}
	}
	p.fireSelect()
	return p.SelectedWorker()
}
