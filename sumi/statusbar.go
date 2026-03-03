package sumi

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/dpopsuev/origami/view"
)

var (
	styleStatusLabel = lipgloss.NewStyle().Bold(true)
	styleStatusValue = lipgloss.NewStyle()
	styleStatusSep   = lipgloss.NewStyle().Faint(true)
	styleStatusBg    = lipgloss.NewStyle().Background(lipgloss.Color("236")).Padding(0, 1)
	styleStatusOK    = lipgloss.NewStyle().Foreground(lipgloss.Color("34"))
	styleStatusErr   = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	styleStatusWarn  = lipgloss.NewStyle().Foreground(lipgloss.Color("226"))
)

// StatusBarData holds the information needed to render the status bar.
type StatusBarData struct {
	CircuitName  string
	WorkerCount  int
	EventCount   int
	KamiStatus   KamiStatus
	Paused       bool
	Completed    bool
	Error        string
	Elapsed      time.Duration
	SelectedNode string
	NoColor      bool
	Width        int
	Flash        string
}

// StatusBarDataFromModel extracts status bar data from the current model state.
func StatusBarDataFromModel(m *Model) StatusBarData {
	return StatusBarData{
		CircuitName:  m.def.Circuit,
		WorkerCount:  len(m.snap.Walkers),
		EventCount:   m.eventCount,
		KamiStatus:   m.kamiStatus,
		Paused:       m.snap.Paused,
		Completed:    m.snap.Completed,
		Error:        m.snap.Error,
		SelectedNode: selectedNodeName(m),
		NoColor:      m.opts.NoColor,
		Width:        m.width,
		Flash:        m.statusFlash,
	}
}

func selectedNodeName(m *Model) string {
	if m.selected < len(m.nodeOrder) {
		return m.nodeOrder[m.selected]
	}
	return ""
}

// RenderWarRoomStatusBar renders the enhanced status bar for War Room mode.
func RenderWarRoomStatusBar(d StatusBarData) string {
	sep := " │ "
	if d.NoColor {
		return renderStatusBarPlain(d, sep)
	}

	parts := []string{
		styleStatusLabel.Render("⬡ " + d.CircuitName),
	}

	parts = append(parts, styleStatusValue.Render(fmt.Sprintf("Workers: %d", d.WorkerCount)))
	parts = append(parts, styleStatusValue.Render(fmt.Sprintf("Events: %d", d.EventCount)))

	switch d.KamiStatus {
	case KamiConnected:
		parts = append(parts, styleStatusOK.Render("● SSE"))
	default:
		parts = append(parts, lipgloss.NewStyle().Faint(true).Render("○ SSE"))
	}

	if d.Paused {
		parts = append(parts, styleStatusWarn.Render("⏸ PAUSED"))
	}
	if d.Completed {
		parts = append(parts, styleStatusOK.Render("✓ DONE"))
	}
	if d.Error != "" {
		errStr := d.Error
		if len(errStr) > 30 {
			errStr = errStr[:30] + "…"
		}
		parts = append(parts, styleStatusErr.Render("✗ "+errStr))
	}

	if d.SelectedNode != "" {
		parts = append(parts, styleStatusValue.Render("→ "+d.SelectedNode))
	}

	if d.Flash != "" {
		parts = append(parts, styleStatusWarn.Render("⚡ "+d.Flash))
	}

	content := strings.Join(parts, styleStatusSep.Render(sep))
	return styleStatusBg.Width(d.Width).Render(content)
}

func renderStatusBarPlain(d StatusBarData, sep string) string {
	parts := []string{d.CircuitName}
	parts = append(parts, fmt.Sprintf("Workers: %d", d.WorkerCount))
	parts = append(parts, fmt.Sprintf("Events: %d", d.EventCount))

	if d.KamiStatus == KamiConnected {
		parts = append(parts, "SSE: on")
	} else {
		parts = append(parts, "SSE: off")
	}

	if d.Paused {
		parts = append(parts, "PAUSED")
	}
	if d.Completed {
		parts = append(parts, "DONE")
	}
	if d.Error != "" {
		parts = append(parts, "ERR: "+d.Error)
	}
	if d.SelectedNode != "" {
		parts = append(parts, "-> "+d.SelectedNode)
	}
	if d.Flash != "" {
		parts = append(parts, d.Flash)
	}
	return strings.Join(parts, sep)
}

// RenderWalkerProgress renders a progress summary for walkers in the snapshot.
func RenderWalkerProgress(snap view.CircuitSnapshot, noColor bool) string {
	if len(snap.Walkers) == 0 {
		return ""
	}

	completed := 0
	for _, ns := range snap.Nodes {
		if ns.State == view.NodeCompleted {
			completed++
		}
	}
	total := len(snap.Nodes)

	var parts []string
	for _, wp := range snap.Walkers {
		bar := progressBar(completed, total, 10)
		entry := fmt.Sprintf("%s @ %s %s %d/%d", wp.WalkerID, wp.Node, bar, completed, total)
		if !noColor {
			entry = ElementFg(wp.Element).Render(entry)
		}
		parts = append(parts, entry)
	}
	return strings.Join(parts, "  ")
}
