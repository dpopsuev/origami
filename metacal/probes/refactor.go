package probes

import (
	"github.com/dpopsuev/origami/metacal"
)

// RefactorSpec returns the ProbeSpec for the refactoring probe.
// Stimulus: valid but messy Go code. The agent refactors it.
// We observe structure-oriented behavior (renames, splits, comments)
// vs speed-oriented behavior (minimal changes, quick return).
func RefactorSpec() metacal.ProbeSpec {
	return metacal.ProbeSpec{
		ID:          "refactor-v1",
		Name:        "Refactoring Probe",
		Description: "Messy Go function; agent refactors for production quality. Measures structure vs speed.",
		Step:        metacal.StepRefactor,
		Dimensions: []metacal.Dimension{
			metacal.DimSpeed,
			metacal.DimShortcutAffinity,
			metacal.DimEvidenceDepth,
		},
		Input: metacal.MessyInput,
	}
}

// RefactorPrompt returns the prompt text for the refactoring probe.
func RefactorPrompt() string {
	return metacal.BuildProbePrompt()
}

// ScoreRefactor maps refactoring output to behavioral dimension scores.
// High renames + splits + comments -> high EvidenceDepth (thoroughness).
// Few changes -> high Speed and ShortcutAffinity (took the fast path).
func ScoreRefactor(raw string) map[metacal.Dimension]float64 {
	legacy := metacal.ScoreRefactorOutput(raw)

	thoroughness := float64(legacy.Renames)*0.15 +
		float64(legacy.FunctionSplits)*0.15 +
		float64(legacy.CommentsAdded)*0.10 +
		float64(legacy.StructuralChanges)*0.15
	if thoroughness > 1.0 {
		thoroughness = 1.0
	}

	speed := 1.0 - thoroughness
	shortcut := 1.0 - thoroughness

	return map[metacal.Dimension]float64{
		metacal.DimSpeed:            clamp(speed),
		metacal.DimShortcutAffinity: clamp(shortcut),
		metacal.DimEvidenceDepth:    clamp(thoroughness),
	}
}

func clamp(v float64) float64 {
	if v < 0.0 {
		return 0.0
	}
	if v > 1.0 {
		return 1.0
	}
	return v
}
