package calibrate

import (
	"fmt"
	"strings"

	"github.com/dpopsuev/origami/dispatch"
	"github.com/dpopsuev/origami/format"
)

// MetricSection groups metrics under a titled section for report formatting.
type MetricSection struct {
	Title   string
	Metrics []Metric
}

// FormatConfig controls how FormatReport renders the calibration report.
type FormatConfig struct {
	Title          string
	Sections       []MetricSection
	MetricNameFunc func(id string) string
	ThresholdFunc  func(Metric) string
}

// DefaultThresholdFormat renders the threshold as ">= X.XX".
func DefaultThresholdFormat(m Metric) string {
	return fmt.Sprintf("≥%.2f", m.Threshold)
}

// tierOrder defines the display ordering for auto-generated sections.
var tierOrder = []CostTier{TierOutcome, TierInvestigation, TierDetection, TierEfficiency, TierMeta, ""}

// tierTitle maps a CostTier to a human-readable section title.
var tierTitle = map[CostTier]string{
	TierOutcome:       "Outcome",
	TierInvestigation: "Investigation",
	TierDetection:     "Detection",
	TierEfficiency:    "Efficiency",
	TierMeta:          "Meta",
	"":                "Other",
}

// sectionsFromTier auto-generates MetricSections grouped by Tier.
func sectionsFromTier(ms MetricSet) []MetricSection {
	byTier := ms.ByTier()
	var sections []MetricSection
	for _, tier := range tierOrder {
		metrics, ok := byTier[tier]
		if !ok || len(metrics) == 0 {
			continue
		}
		title := tierTitle[tier]
		sections = append(sections, MetricSection{Title: title, Metrics: metrics})
	}
	return sections
}

// FormatReport produces a human-readable calibration report with metric
// tables, a pass/fail result line, and an optional token summary.
// When cfg.Sections is empty, sections are auto-generated from Metric.Tier.
func FormatReport(report *CalibrationReport, cfg FormatConfig) string {
	var b strings.Builder

	title := cfg.Title
	if title == "" {
		title = "Calibration Report"
	}
	b.WriteString(fmt.Sprintf("=== %s ===\n", title))
	b.WriteString(fmt.Sprintf("Scenario: %s\n", report.Scenario))
	b.WriteString(fmt.Sprintf("Adapter:  %s\n", report.Adapter))
	b.WriteString(fmt.Sprintf("Runs:     %d\n\n", report.Runs))

	nameFunc := cfg.MetricNameFunc
	if nameFunc == nil {
		nameFunc = func(_ string) string { return "" }
	}
	threshFunc := cfg.ThresholdFunc
	if threshFunc == nil {
		threshFunc = DefaultThresholdFormat
	}

	sections := cfg.Sections
	if len(sections) == 0 {
		sections = sectionsFromTier(report.Metrics)
	}

	for _, sec := range sections {
		b.WriteString(fmt.Sprintf("--- %s ---\n", sec.Title))
		tbl := format.NewTable(format.ASCII)
		tbl.Header("ID", "Metric", "Value", "Detail", "Pass", "Threshold")
		tbl.Columns(
			format.ColumnConfig{Number: 1, Align: format.AlignLeft},
			format.ColumnConfig{Number: 2, Align: format.AlignLeft},
			format.ColumnConfig{Number: 3, Align: format.AlignRight},
			format.ColumnConfig{Number: 4, Align: format.AlignLeft},
			format.ColumnConfig{Number: 5, Align: format.AlignCenter},
			format.ColumnConfig{Number: 6, Align: format.AlignLeft},
		)
		for _, m := range sec.Metrics {
			displayName := nameFunc(m.ID)
			if displayName == "" {
				displayName = m.Name
			}
			passMark := format.BoolMark(m.Pass)
			if m.DryCapped {
				passMark = "~"
			}
			tbl.Row(
				m.ID,
				displayName,
				fmt.Sprintf("%.2f", m.Value),
				m.Detail,
				passMark,
				threshFunc(m),
			)
		}
		b.WriteString(tbl.String())
		b.WriteString("\n\n")
	}

	passed, total := report.Metrics.PassCount()
	result := "PASS"
	if passed < total {
		result = "FAIL"
	}
	b.WriteString(fmt.Sprintf("RESULT: %s (%d/%d metrics within threshold)\n\n", result, passed, total))

	if report.Tokens != nil {
		b.WriteString(dispatch.FormatTokenSummary(*report.Tokens))
		b.WriteString("\n")
	}

	return b.String()
}
