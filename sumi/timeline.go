package sumi

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/dpopsuev/origami/view"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const timelineMaxEntries = 500

// TimelineEntry is a single formatted event in the timeline.
type TimelineEntry struct {
	Timestamp time.Time
	Walker    string
	Type      view.DiffType
	Node      string
	Detail    string
}

// FormatEntry produces a single-line representation of a timeline entry.
func (e TimelineEntry) FormatEntry(noColor bool) string {
	ts := e.Timestamp.Format("15:04:05")
	walker := e.Walker
	if walker == "" {
		walker = "---"
	}
	evtType := string(e.Type)
	node := e.Node

	line := fmt.Sprintf("%s %-8s %-16s %s", ts, walker, evtType, node)
	if e.Detail != "" {
		line += " " + e.Detail
	}

	if noColor {
		return line
	}
	return styleForDiffType(e.Type).Render(line)
}

func styleForDiffType(dt view.DiffType) lipgloss.Style {
	switch dt {
	case view.DiffNodeState:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("34"))
	case view.DiffWalkerMoved:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("226"))
	case view.DiffWalkerAdded:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("51"))
	case view.DiffWalkerRemoved:
		return lipgloss.NewStyle().Faint(true)
	case view.DiffBreakpointSet, view.DiffBreakpointCleared:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("129"))
	case view.DiffCompleted:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("34")).Bold(true)
	case view.DiffError:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	case view.DiffPaused:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Bold(true)
	case view.DiffResumed:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("33"))
	default:
		return lipgloss.NewStyle()
	}
}

// TimelineRingBuffer is a thread-safe fixed-capacity ring buffer of timeline entries.
type TimelineRingBuffer struct {
	mu      sync.RWMutex
	entries []TimelineEntry
	head    int
	count   int
	cap     int
}

// NewTimelineRingBuffer creates a ring buffer with the given capacity.
func NewTimelineRingBuffer(capacity int) *TimelineRingBuffer {
	return &TimelineRingBuffer{
		entries: make([]TimelineEntry, capacity),
		cap:     capacity,
	}
}

// Push adds an entry, overwriting the oldest if full.
func (rb *TimelineRingBuffer) Push(e TimelineEntry) {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	idx := (rb.head + rb.count) % rb.cap
	if rb.count == rb.cap {
		rb.head = (rb.head + 1) % rb.cap
	} else {
		rb.count++
	}
	rb.entries[idx] = e
}

// Len returns the number of entries.
func (rb *TimelineRingBuffer) Len() int {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	return rb.count
}

// All returns all entries in order (oldest first).
func (rb *TimelineRingBuffer) All() []TimelineEntry {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	out := make([]TimelineEntry, rb.count)
	for i := 0; i < rb.count; i++ {
		out[i] = rb.entries[(rb.head+i)%rb.cap]
	}
	return out
}

// Filtered returns entries matching the given worker ID. Empty string returns all.
func (rb *TimelineRingBuffer) Filtered(workerID string) []TimelineEntry {
	if workerID == "" {
		return rb.All()
	}
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	var out []TimelineEntry
	for i := 0; i < rb.count; i++ {
		e := rb.entries[(rb.head+i)%rb.cap]
		if e.Walker == workerID {
			out = append(out, e)
		}
	}
	return out
}

// DiffToTimelineEntry converts a StateDiff to a TimelineEntry.
func DiffToTimelineEntry(diff view.StateDiff) TimelineEntry {
	e := TimelineEntry{
		Timestamp: diff.Timestamp,
		Walker:    diff.Walker,
		Type:      diff.Type,
		Node:      diff.Node,
	}
	if diff.State != "" {
		e.Detail = string(diff.State)
	}
	if diff.Error != "" {
		e.Detail = diff.Error
	}
	return e
}

// TimelinePanel implements Panel for the event timeline.
type TimelinePanel struct {
	buffer       *TimelineRingBuffer
	noColor      bool
	scrollY      int
	autoScroll   bool
	workerFilter string

	onSelectNode func(node string)
}

// NewTimelinePanel creates a timeline panel.
func NewTimelinePanel(buffer *TimelineRingBuffer, noColor bool, onSelectNode func(string)) *TimelinePanel {
	return &TimelinePanel{
		buffer:       buffer,
		noColor:      noColor,
		autoScroll:   true,
		onSelectNode: onSelectNode,
	}
}

func (p *TimelinePanel) ID() string                { return "timeline" }
func (p *TimelinePanel) Title() string             { return "Events" }
func (p *TimelinePanel) Focusable() bool           { return true }
func (p *TimelinePanel) PreferredSize() (int, int) { return 60, bottomMinH }

// SetWorkerFilter restricts the timeline to a specific worker. Empty = show all.
func (p *TimelinePanel) SetWorkerFilter(wid string) {
	p.workerFilter = wid
}

func (p *TimelinePanel) Update(msg tea.Msg) tea.Cmd {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return nil
	}
	switch km.String() {
	case "up":
		p.autoScroll = false
		if p.scrollY > 0 {
			p.scrollY--
		}
	case "down":
		p.scrollY++
	case "end":
		p.autoScroll = true
	case "home":
		p.autoScroll = false
		p.scrollY = 0
	case "enter":
		entries := p.buffer.Filtered(p.workerFilter)
		if p.scrollY < len(entries) && p.onSelectNode != nil {
			e := entries[p.scrollY]
			if e.Node != "" {
				p.onSelectNode(e.Node)
			}
		}
	}
	return nil
}

func (p *TimelinePanel) View(area Rect) string {
	inner := area.Inner()
	if inner.W <= 0 || inner.H <= 0 {
		return ""
	}

	entries := p.buffer.Filtered(p.workerFilter)
	if len(entries) == 0 {
		return "Waiting for events..."
	}

	if p.autoScroll {
		p.scrollY = len(entries) - inner.H
		if p.scrollY < 0 {
			p.scrollY = 0
		}
	}

	maxScroll := len(entries) - inner.H
	if maxScroll < 0 {
		maxScroll = 0
	}
	if p.scrollY > maxScroll {
		p.scrollY = maxScroll
	}

	start := p.scrollY
	end := start + inner.H
	if end > len(entries) {
		end = len(entries)
	}

	var sb strings.Builder
	for i := start; i < end; i++ {
		line := entries[i].FormatEntry(p.noColor)
		sb.WriteString(padOrTruncate(line, inner.W))
		if i < end-1 {
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

// SelectByClick selects the timeline entry at the given local Y offset
// and fires the onSelectNode callback if the entry has a node.
func (p *TimelinePanel) SelectByClick(localY int) {
	entries := p.buffer.Filtered(p.workerFilter)
	idx := p.scrollY + localY
	if idx >= 0 && idx < len(entries) && p.onSelectNode != nil {
		e := entries[idx]
		if e.Node != "" {
			p.onSelectNode(e.Node)
		}
	}
}
