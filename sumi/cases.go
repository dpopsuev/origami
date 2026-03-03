package sumi

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dpopsuev/origami/view"
)

// CasesPanel implements Panel for the case results tab bar.
type CasesPanel struct {
	cases    *view.CaseResultSet
	noColor  bool
	selected int
}

// NewCasesPanel creates a case results panel stub.
func NewCasesPanel(cases *view.CaseResultSet, noColor bool) *CasesPanel {
	return &CasesPanel{
		cases:   cases,
		noColor: noColor,
	}
}

func (p *CasesPanel) ID() string                { return "cases" }
func (p *CasesPanel) Title() string             { return "Cases" }
func (p *CasesPanel) Focusable() bool           { return true }
func (p *CasesPanel) PreferredSize() (int, int) { return 60, bottomTabsH }

// SelectedCase returns the currently selected case ID, or "".
func (p *CasesPanel) SelectedCase() string {
	if p.cases == nil || p.selected >= p.cases.Len() {
		return ""
	}
	return p.cases.Cases[p.selected].CaseID
}

func (p *CasesPanel) Update(msg tea.Msg) tea.Cmd {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return nil
	}
	if p.cases == nil || p.cases.Len() == 0 {
		return nil
	}
	switch km.String() {
	case "left":
		if p.selected > 0 {
			p.selected--
		}
	case "right":
		if p.selected < p.cases.Len()-1 {
			p.selected++
		}
	}
	return nil
}

func (p *CasesPanel) View(area Rect) string {
	inner := area.Inner()
	if inner.W <= 0 || inner.H <= 0 {
		return ""
	}

	if p.cases == nil || p.cases.Len() == 0 {
		return "No case results yet."
	}

	var sb strings.Builder

	for i, c := range p.cases.Cases {
		tab := formatCaseTab(c, p.noColor)
		if i == p.selected {
			if p.noColor {
				tab = "[" + tab + "]"
			} else {
				tab = lipgloss.NewStyle().Bold(true).Reverse(true).Render(tab)
			}
		}
		sb.WriteString(tab)
		if i < p.cases.Len()-1 {
			sb.WriteString("  ")
		}
	}
	sb.WriteByte('\n')

	if p.selected < p.cases.Len() {
		c := p.cases.Cases[p.selected]
		detail := fmt.Sprintf("  %s  Confidence: %.0f%%  %s",
			c.DefectType, c.Confidence*100, c.Summary)
		sb.WriteString(padOrTruncate(detail, inner.W))
	}

	return sb.String()
}

// SelectByClick selects the case tab at the given local X offset.
func (p *CasesPanel) SelectByClick(localX int) {
	if p.cases == nil || p.cases.Len() == 0 {
		return
	}
	tabW := 12
	idx := localX / tabW
	if idx >= 0 && idx < p.cases.Len() {
		p.selected = idx
	}
}

func formatCaseTab(c view.CaseResult, noColor bool) string {
	status := c.Status
	if status == "" {
		status = "?"
	}

	label := fmt.Sprintf(" %s [%s] ", c.CaseID, status)

	if noColor {
		return label
	}

	switch c.Status {
	case "pass":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("34")).Render(label)
	case "fail":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(label)
	case "skip":
		return lipgloss.NewStyle().Faint(true).Render(label)
	default:
		return label
	}
}
