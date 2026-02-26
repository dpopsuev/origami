package probes

import (
	"fmt"
	"strings"

	"github.com/dpopsuev/origami/ouroboros"
)

// BuildRefactorPrompt returns the prompt text using the given stimulus.
// The stimulus Input is embedded in a refactoring instruction template.
// When Language is set, the code fence and instructions are language-specific.
func BuildRefactorPrompt(s ProbeStimulus) string {
	lang := s.Language
	if lang == "" {
		lang = "Go"
	}
	return fmt.Sprintf(`You are given the following %s function. Refactor it for production quality.
Return ONLY the refactored %s code between triple backticks. No explanation needed.

%s%s%s

Rules:
- Rename variables and the function to be descriptive
- Split into smaller functions if appropriate
- Add comments where they aid understanding
- Preserve the exact behavior (same inputs produce same outputs)
- Use idiomatic %s patterns`, lang, lang, "```"+strings.ToLower(lang)+"\n", s.Input, "\n```", lang)
}

// RefactorPrompt returns the prompt text using the default stimulus.
func RefactorPrompt() string {
	return BuildRefactorPrompt(DefaultStimuli()["refactor"])
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
