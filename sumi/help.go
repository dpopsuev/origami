package sumi

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var helpContent = []helpSection{
	{
		Title: "Global",
		Bindings: []helpBinding{
			{"q / Ctrl+C", "Quit"},
			{"?", "Toggle help"},
			{"Tab", "Next panel"},
			{"Shift+Tab", "Previous panel"},
			{"/", "Search nodes"},
		},
	},
	{
		Title: "Graph",
		Bindings: []helpBinding{
			{"Click node", "Select node"},
			{"Arrow keys", "Navigate nodes"},
			{"Enter", "Toggle inspector (legacy)"},
			{"Space", "Toggle breakpoint"},
			{"p", "Pause walk"},
			{"r", "Resume walk"},
			{"n", "Step to next node"},
		},
	},
	{
		Title: "Inspector",
		Bindings: []helpBinding{
			{"Up/Down", "Scroll"},
			{"Home", "Scroll to top"},
		},
	},
	{
		Title: "Timeline",
		Bindings: []helpBinding{
			{"Up/Down", "Scroll"},
			{"Home", "Scroll to top"},
			{"End", "Resume auto-scroll"},
			{"Enter", "Select entry's node"},
			{"Click entry", "Select entry's node"},
		},
	},
	{
		Title: "Workers",
		Bindings: []helpBinding{
			{"Up/Down", "Navigate workers"},
			{"Enter", "Filter by worker"},
			{"Click worker", "Filter by worker"},
		},
	},
	{
		Title: "Mouse",
		Bindings: []helpBinding{
			{"Click panel", "Focus panel"},
			{"Click node", "Select and inspect node"},
			{"Click worker", "Filter events by worker"},
			{"Click event", "Jump to event's node"},
		},
	},
}

type helpSection struct {
	Title    string
	Bindings []helpBinding
}

type helpBinding struct {
	Key  string
	Desc string
}

var (
	styleHelpTitle   = lipgloss.NewStyle().Bold(true).Underline(true)
	styleHelpKey     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("33"))
	styleHelpDesc    = lipgloss.NewStyle().Faint(true)
	styleHelpOverlay = lipgloss.NewStyle().
				Border(lipgloss.DoubleBorder()).
				BorderForeground(lipgloss.Color("33")).
				Padding(1, 2)
)

// RenderHelpOverlay renders the help modal centered in the given area.
func RenderHelpOverlay(width, height int, noColor bool) string {
	var sb strings.Builder

	sb.WriteString("Sumi War Room — Key Bindings\n\n")

	for i, section := range helpContent {
		if noColor {
			sb.WriteString(section.Title + "\n")
		} else {
			sb.WriteString(styleHelpTitle.Render(section.Title) + "\n")
		}
		for _, b := range section.Bindings {
			if noColor {
				sb.WriteString("  " + padRight(b.Key, 16) + b.Desc + "\n")
			} else {
				sb.WriteString("  " + styleHelpKey.Render(padRight(b.Key, 16)) + styleHelpDesc.Render(b.Desc) + "\n")
			}
		}
		if i < len(helpContent)-1 {
			sb.WriteByte('\n')
		}
	}

	content := sb.String()
	if noColor {
		return content
	}

	maxW := width - 8
	if maxW > 60 {
		maxW = 60
	}
	maxH := height - 4
	if maxH > 40 {
		maxH = 40
	}

	return styleHelpOverlay.
		Width(maxW).
		MaxHeight(maxH).
		Render(content)
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}
