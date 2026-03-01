package ouroboros

import (
	"fmt"
	"strings"
)

// OffsetCompensator generates corrective preamble instructions from measured
// model biases. The corrective text is appended to a walker's PromptPreamble,
// not replacing it. Thresholds are intentionally generous — only significant
// deviations produce corrections.
type OffsetCompensator struct {
	HighThreshold float64 // above this triggers correction (default 0.75)
	LowThreshold  float64 // below this triggers correction (default 0.3)
}

// DefaultOffsetCompensator returns a compensator with standard thresholds.
func DefaultOffsetCompensator() *OffsetCompensator {
	return &OffsetCompensator{
		HighThreshold: 0.75,
		LowThreshold:  0.3,
	}
}

// Compensate analyzes dimension scores and returns corrective instructions.
// Returns empty string if no significant biases are detected.
func (c *OffsetCompensator) Compensate(dimensions map[Dimension]float64) string {
	if len(dimensions) == 0 {
		return ""
	}

	high := c.HighThreshold
	low := c.LowThreshold

	var corrections []string

	speed := dimensions[DimSpeed]
	depth := dimensions[DimEvidenceDepth]
	persistence := dimensions[DimPersistence]
	shortcut := dimensions[DimShortcutAffinity]
	convergence := dimensions[DimConvergenceThreshold]
	failureMode := dimensions[DimFailureMode]

	if depth > high {
		corrections = append(corrections,
			"You tend to overstate confidence in your conclusions. Be more conservative — qualify certainty with evidence strength.")
	}
	if depth < low && speed > high {
		corrections = append(corrections,
			"You tend to rush analysis and skip evidence. Slow down, cite specific evidence items, and explain your reasoning chain.")
	}
	if shortcut > high {
		corrections = append(corrections,
			"You tend to take shortcuts. Resist the urge to skip steps — follow the full analytical path even when the answer seems obvious.")
	}
	if persistence < low {
		corrections = append(corrections,
			"You tend to give up too easily on ambiguous cases. Push through uncertainty — attempt partial answers rather than deferring.")
	}
	if convergence < low {
		corrections = append(corrections,
			"You tend to accept conclusions prematurely. Require stronger evidence before declaring a finding confirmed.")
	}
	if failureMode > high {
		corrections = append(corrections,
			"Your outputs tend to be verbose. Be more concise — prioritize signal over exhaustive coverage.")
	}

	if len(corrections) == 0 {
		return ""
	}

	return fmt.Sprintf("[Calibration offset — auto-generated from model profile]\n%s",
		strings.Join(corrections, "\n"))
}
