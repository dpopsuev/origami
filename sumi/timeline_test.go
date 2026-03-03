package sumi

import (
	"strings"
	"testing"
	"time"

	"github.com/dpopsuev/origami/view"

	tea "github.com/charmbracelet/bubbletea"
)

func TestTimelineRingBuffer_PushAndAll(t *testing.T) {
	rb := NewTimelineRingBuffer(3)
	now := time.Now()

	rb.Push(TimelineEntry{Timestamp: now, Walker: "w1", Type: view.DiffNodeState, Node: "a"})
	rb.Push(TimelineEntry{Timestamp: now, Walker: "w2", Type: view.DiffNodeState, Node: "b"})

	all := rb.All()
	if len(all) != 2 {
		t.Fatalf("Len = %d, want 2", len(all))
	}
	if all[0].Node != "a" || all[1].Node != "b" {
		t.Errorf("entries = [%s, %s], want [a, b]", all[0].Node, all[1].Node)
	}
}

func TestTimelineRingBuffer_Overflow(t *testing.T) {
	rb := NewTimelineRingBuffer(3)
	now := time.Now()

	for i := 0; i < 5; i++ {
		rb.Push(TimelineEntry{Timestamp: now, Node: string(rune('a' + i))})
	}

	if rb.Len() != 3 {
		t.Fatalf("Len = %d, want 3 (capacity)", rb.Len())
	}

	all := rb.All()
	if all[0].Node != "c" || all[1].Node != "d" || all[2].Node != "e" {
		t.Errorf("after overflow: [%s, %s, %s], want [c, d, e]",
			all[0].Node, all[1].Node, all[2].Node)
	}
}

func TestTimelineRingBuffer_Filtered(t *testing.T) {
	rb := NewTimelineRingBuffer(10)
	now := time.Now()

	rb.Push(TimelineEntry{Timestamp: now, Walker: "w1", Node: "a"})
	rb.Push(TimelineEntry{Timestamp: now, Walker: "w2", Node: "b"})
	rb.Push(TimelineEntry{Timestamp: now, Walker: "w1", Node: "c"})

	filtered := rb.Filtered("w1")
	if len(filtered) != 2 {
		t.Fatalf("filtered len = %d, want 2", len(filtered))
	}
	if filtered[0].Node != "a" || filtered[1].Node != "c" {
		t.Errorf("filtered = [%s, %s], want [a, c]", filtered[0].Node, filtered[1].Node)
	}

	all := rb.Filtered("")
	if len(all) != 3 {
		t.Errorf("empty filter should return all (%d), got %d", rb.Len(), len(all))
	}
}

func TestTimelineEntry_FormatEntry(t *testing.T) {
	e := TimelineEntry{
		Timestamp: time.Date(2026, 3, 1, 14, 2, 3, 0, time.UTC),
		Walker:    "w-01",
		Type:      view.DiffNodeState,
		Node:      "triage",
		Detail:    "active",
	}

	formatted := e.FormatEntry(true)
	if !strings.Contains(formatted, "14:02:03") {
		t.Errorf("should contain timestamp, got: %s", formatted)
	}
	if !strings.Contains(formatted, "w-01") {
		t.Errorf("should contain walker, got: %s", formatted)
	}
	if !strings.Contains(formatted, "triage") {
		t.Errorf("should contain node, got: %s", formatted)
	}
	if !strings.Contains(formatted, "active") {
		t.Errorf("should contain detail, got: %s", formatted)
	}
}

func TestTimelineEntry_FormatEntry_NoWalker(t *testing.T) {
	e := TimelineEntry{
		Timestamp: time.Now(),
		Type:      view.DiffCompleted,
	}
	formatted := e.FormatEntry(true)
	if !strings.Contains(formatted, "---") {
		t.Errorf("missing walker should show ---, got: %s", formatted)
	}
}

func TestDiffToTimelineEntry(t *testing.T) {
	diff := view.StateDiff{
		Type:      view.DiffNodeState,
		Node:      "recall",
		Walker:    "w-01",
		State:     view.NodeActive,
		Timestamp: time.Now(),
	}

	entry := DiffToTimelineEntry(diff)
	if entry.Node != "recall" {
		t.Errorf("Node = %q, want recall", entry.Node)
	}
	if entry.Walker != "w-01" {
		t.Errorf("Walker = %q, want w-01", entry.Walker)
	}
	if entry.Detail != "active" {
		t.Errorf("Detail = %q, want active", entry.Detail)
	}
}

func TestDiffToTimelineEntry_Error(t *testing.T) {
	diff := view.StateDiff{
		Type:      view.DiffError,
		Node:      "triage",
		Error:     "timeout",
		Timestamp: time.Now(),
	}

	entry := DiffToTimelineEntry(diff)
	if entry.Detail != "timeout" {
		t.Errorf("Detail = %q, want timeout", entry.Detail)
	}
}

func TestTimelinePanel_ID(t *testing.T) {
	rb := NewTimelineRingBuffer(10)
	p := NewTimelinePanel(rb, true, nil)
	if p.ID() != "timeline" {
		t.Errorf("ID = %q, want timeline", p.ID())
	}
}

func TestTimelinePanel_EmptyView(t *testing.T) {
	rb := NewTimelineRingBuffer(10)
	p := NewTimelinePanel(rb, true, nil)

	content := p.View(Rect{0, 0, 80, 10})
	if !strings.Contains(content, "Waiting for events") {
		t.Errorf("empty timeline should show waiting message, got: %s", content)
	}
}

func TestTimelinePanel_ShowsEntries(t *testing.T) {
	rb := NewTimelineRingBuffer(10)
	rb.Push(TimelineEntry{
		Timestamp: time.Date(2026, 3, 1, 14, 0, 0, 0, time.UTC),
		Walker:    "w1",
		Type:      view.DiffNodeState,
		Node:      "recall",
	})

	p := NewTimelinePanel(rb, true, nil)
	content := p.View(Rect{0, 0, 80, 10})

	if !strings.Contains(content, "recall") {
		t.Errorf("timeline should show entry, got: %s", content)
	}
}

func TestTimelinePanel_WorkerFilter(t *testing.T) {
	rb := NewTimelineRingBuffer(10)
	rb.Push(TimelineEntry{Timestamp: time.Now(), Walker: "w1", Node: "a"})
	rb.Push(TimelineEntry{Timestamp: time.Now(), Walker: "w2", Node: "b"})
	rb.Push(TimelineEntry{Timestamp: time.Now(), Walker: "w1", Node: "c"})

	p := NewTimelinePanel(rb, true, nil)
	p.SetWorkerFilter("w1")

	content := p.View(Rect{0, 0, 80, 10})
	if strings.Contains(content, " b") {
		t.Errorf("filtered timeline should not show w2's entries, got:\n%s", content)
	}
	if !strings.Contains(content, "a") || !strings.Contains(content, "c") {
		t.Errorf("filtered timeline should show w1's entries, got:\n%s", content)
	}
}

func TestTimelinePanel_AutoScroll(t *testing.T) {
	rb := NewTimelineRingBuffer(100)
	for i := 0; i < 20; i++ {
		rb.Push(TimelineEntry{
			Timestamp: time.Now(),
			Node:      string(rune('a' + i%26)),
			Type:      view.DiffNodeState,
		})
	}

	p := NewTimelinePanel(rb, true, nil)
	content := p.View(Rect{0, 0, 80, 7}) // inner height = 5

	lines := strings.Split(content, "\n")
	lastLine := lines[len(lines)-1]
	if !strings.Contains(lastLine, string(rune('a'+(19%26)))) {
		t.Errorf("auto-scroll should show last entry at bottom, last line: %s", lastLine)
	}
}

func TestTimelinePanel_ScrollUp(t *testing.T) {
	rb := NewTimelineRingBuffer(100)
	for i := 0; i < 20; i++ {
		rb.Push(TimelineEntry{
			Timestamp: time.Now(),
			Node:      string(rune('a' + i%26)),
			Type:      view.DiffNodeState,
		})
	}

	p := NewTimelinePanel(rb, true, nil)
	p.View(Rect{0, 0, 80, 7})

	p.Update(tea.KeyMsg{Type: tea.KeyUp})
	if p.autoScroll {
		t.Error("up arrow should disable auto-scroll")
	}
}

func TestTimelinePanel_EndReenablesAutoScroll(t *testing.T) {
	rb := NewTimelineRingBuffer(10)
	p := NewTimelinePanel(rb, true, nil)
	p.autoScroll = false

	p.Update(tea.KeyMsg{Type: tea.KeyEnd})
	if !p.autoScroll {
		t.Error("End key should re-enable auto-scroll")
	}
}

func TestTimelinePanel_SelectByClick(t *testing.T) {
	rb := NewTimelineRingBuffer(10)
	rb.Push(TimelineEntry{Timestamp: time.Now(), Node: "recall"})
	rb.Push(TimelineEntry{Timestamp: time.Now(), Node: "triage"})

	var selected string
	p := NewTimelinePanel(rb, true, func(node string) { selected = node })

	p.SelectByClick(1)
	if selected != "triage" {
		t.Errorf("click on second entry should select triage, got %q", selected)
	}
}

func TestTimelinePanel_EnterSelectsNode(t *testing.T) {
	rb := NewTimelineRingBuffer(10)
	rb.Push(TimelineEntry{Timestamp: time.Now(), Node: "recall"})

	var selected string
	p := NewTimelinePanel(rb, true, func(node string) { selected = node })
	p.scrollY = 0

	p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if selected != "recall" {
		t.Errorf("Enter should select current entry's node, got %q", selected)
	}
}
