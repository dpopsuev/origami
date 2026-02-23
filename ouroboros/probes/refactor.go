package probes

import (
	"github.com/dpopsuev/origami/ouroboros"
)

// RefactorSpec returns the ProbeSpec for the refactoring probe.
// Stimulus: valid but messy Go code. The agent refactors it.
// We observe structure-oriented behavior (renames, splits, comments)
// vs speed-oriented behavior (minimal changes, quick return).
func RefactorSpec() ouroboros.ProbeSpec {
	return ouroboros.ProbeSpec{
		ID:          "refactor-v1",
		Name:        "Refactoring Probe",
		Description: "Messy Go function; agent refactors for production quality. Measures structure vs speed.",
		Step:        ouroboros.StepRefactor,
		Dimensions: []ouroboros.Dimension{
			ouroboros.DimSpeed,
			ouroboros.DimShortcutAffinity,
			ouroboros.DimEvidenceDepth,
		},
		Input: ouroboros.MessyInput,
	}
}

// RefactorPrompt returns the prompt text for the refactoring probe.
func RefactorPrompt() string {
	return ouroboros.BuildProbePrompt()
}

// ScoreRefactor maps refactoring output to behavioral dimension scores.
// High renames + splits + comments -> high EvidenceDepth (thoroughness).
// Few changes -> high Speed and ShortcutAffinity (took the fast path).
func ScoreRefactor(raw string) map[ouroboros.Dimension]float64 {
	legacy := ouroboros.ScoreRefactorOutput(raw)

	thoroughness := float64(legacy.Renames)*0.15 +
		float64(legacy.FunctionSplits)*0.15 +
		float64(legacy.CommentsAdded)*0.10 +
		float64(legacy.StructuralChanges)*0.15
	if thoroughness > 1.0 {
		thoroughness = 1.0
	}

	speed := 1.0 - thoroughness
	shortcut := 1.0 - thoroughness

	return map[ouroboros.Dimension]float64{
		ouroboros.DimSpeed:            clamp(speed),
		ouroboros.DimShortcutAffinity: clamp(shortcut),
		ouroboros.DimEvidenceDepth:    clamp(thoroughness),
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
