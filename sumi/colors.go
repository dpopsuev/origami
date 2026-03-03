// Package sumi provides a terminal-based circuit viewer and debugger (TUI).
//
// Sumi (墨) — "ink" for calligraphy. The terminal is text — ink on paper.
// Washi is the paper surface (GUI); Sumi is the ink on it (TUI).
package sumi

import "github.com/charmbracelet/lipgloss"

// ElementColor maps Origami element names to ANSI 256-color values.
// These are the canonical terminal colors for circuit visualization.
var ElementColor = map[string]lipgloss.Color{
	"fire":      lipgloss.Color("196"), // red
	"water":     lipgloss.Color("33"),  // blue
	"earth":     lipgloss.Color("34"),  // green
	"lightning": lipgloss.Color("226"), // yellow
	"air":       lipgloss.Color("51"),  // cyan
	"void":      lipgloss.Color("129"), // magenta
	"diamond":   lipgloss.Color("255"), // white
}

// ElementFg returns a lipgloss style with the foreground set to the
// element's color. Falls back to default terminal foreground for unknown elements.
func ElementFg(element string) lipgloss.Style {
	if c, ok := ElementColor[element]; ok {
		return lipgloss.NewStyle().Foreground(c)
	}
	return lipgloss.NewStyle()
}

// Style constants for node visual states.
var (
	StyleIdle      = lipgloss.NewStyle().Faint(true)
	StyleActive    = lipgloss.NewStyle().Bold(true)
	StyleCompleted = lipgloss.NewStyle().Foreground(lipgloss.Color("34"))  // green
	StyleError     = lipgloss.NewStyle().Foreground(lipgloss.Color("196")) // red

	StyleSelected   = lipgloss.NewStyle().Foreground(lipgloss.Color("33")).Bold(true)
	StyleZoneBorder = lipgloss.NewStyle().Faint(true)

	StyleBreakpoint = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	StyleWalker     = lipgloss.NewStyle().Bold(true)

	StyleStatusBar = lipgloss.NewStyle().Background(lipgloss.Color("236")).Padding(0, 1)
	StyleSearchBar = lipgloss.NewStyle().Foreground(lipgloss.Color("226"))

	// War Room panel styles
	StylePanelFocused   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("33"))
	StylePanelUnfocused = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240"))
	StylePanelTitle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("33"))
	StyleWorkerActive   = lipgloss.NewStyle().Bold(true)
	StyleWorkerIdle     = lipgloss.NewStyle().Faint(true)
)

// DSBadge returns the OSS unicode symbol for a node's transformer type.
// ⚙ = deterministic, ✦ = stochastic, Δ = dialectic.
func DSBadge(transformer string) string {
	switch {
	case transformer == "":
		return ""
	case isDeterministicTransformer(transformer):
		return "⚙"
	case transformer == "core.dialectic":
		return "Δ"
	default:
		return "✦"
	}
}

func isDeterministicTransformer(t string) bool {
	switch t {
	case "core.jq", "core.file", "core.template", "core.noop", "core.passthrough":
		return true
	default:
		return false
	}
}
