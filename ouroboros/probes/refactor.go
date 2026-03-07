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
	legacy := ScoreRefactorOutput(raw)

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

// ScoreRefactorOutput compares the original messy input against the
// refactored output and produces a ProbeScore.
func ScoreRefactorOutput(refactored string) ouroboros.ProbeScore {
	score := ouroboros.ProbeScore{}

	originalNames := extractIdentifiers(MessyInput)
	refactoredNames := extractIdentifiers(refactored)
	score.Renames = countRenames(originalNames, refactoredNames)

	score.FunctionSplits = countFunctionDecls(refactored) - 1
	if score.FunctionSplits < 0 {
		score.FunctionSplits = 0
	}

	score.CommentsAdded = countComments(refactored) - countComments(MessyInput)
	if score.CommentsAdded < 0 {
		score.CommentsAdded = 0
	}

	score.StructuralChanges = countStructuralChanges(MessyInput, refactored)

	total := float64(score.Renames)*0.3 +
		float64(score.FunctionSplits)*0.25 +
		float64(score.CommentsAdded)*0.2 +
		float64(score.StructuralChanges)*0.25
	if total > 1.0 {
		total = 1.0
	}
	score.TotalScore = total

	return score
}

func extractIdentifiers(src string) map[string]bool {
	ids := map[string]bool{}
	for _, word := range strings.Fields(src) {
		clean := strings.Trim(word, "(){}[],;:=!<>+-.\"")
		if len(clean) >= 1 && len(clean) <= 2 && isIdentChar(clean[0]) {
			ids[clean] = true
		}
	}
	return ids
}

func isIdentChar(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || b == '_'
}

func countRenames(original, refactored map[string]bool) int {
	count := 0
	for id := range original {
		if id == "i" || id == "if" || id == "go" {
			continue
		}
		if !refactored[id] {
			count++
		}
	}
	return count
}

func countFunctionDecls(src string) int {
	return strings.Count(src, "func ")
}

func countComments(src string) int {
	count := 0
	for _, line := range strings.Split(src, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") {
			count++
		}
	}
	return count
}

func countStructuralChanges(original, refactored string) int {
	changes := 0
	if !strings.Contains(original, "var ") && strings.Contains(refactored, "var ") {
		changes++
	}
	if strings.Contains(refactored, "fmt.Errorf(") && strings.Contains(refactored, "%w") {
		changes++
	}
	if strings.Count(refactored, "return") > strings.Count(original, "return") {
		changes++
	}
	if strings.Contains(refactored, "strings.Builder") || strings.Contains(refactored, "bytes.Buffer") {
		changes++
	}
	if strings.Contains(refactored, "range ") && !strings.Contains(original, "range ") {
		changes++
	}
	return changes
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
