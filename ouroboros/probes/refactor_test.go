package probes

import (
	"testing"

	"github.com/dpopsuev/origami/ouroboros"
)

func TestScoreRefactor_NoChanges(t *testing.T) {
	scores := ScoreRefactor(MessyInput)

	if scores[ouroboros.DimEvidenceDepth] != 0.0 {
		t.Errorf("EvidenceDepth = %f, want 0.0 (no changes)", scores[ouroboros.DimEvidenceDepth])
	}
	if scores[ouroboros.DimSpeed] != 1.0 {
		t.Errorf("Speed = %f, want 1.0 (no changes = fast)", scores[ouroboros.DimSpeed])
	}
	if scores[ouroboros.DimShortcutAffinity] != 1.0 {
		t.Errorf("ShortcutAffinity = %f, want 1.0 (no changes = shortcut)", scores[ouroboros.DimShortcutAffinity])
	}
}

func TestScoreRefactor_ThoroughRefactor(t *testing.T) {
	refactored := `// calculateSum processes a slice of numbers and returns the total,
// a formatted string of the processed values, and any error.
func calculateSum(numbers []int, label string, verbose bool) (int, string, error) {
	total := 0
	var details strings.Builder

	for _, number := range numbers {
		if number > 0 {
			total += number
			if verbose {
				fmt.Fprintf(&details, "%d,", number)
			}
		} else if number < 0 {
			total -= number
			if verbose {
				fmt.Fprintf(&details, "(%d),", number)
			}
		}
	}

	if total == 0 {
		return 0, "", fmt.Errorf("empty result for %s", label)
	}

	result := formatResult(label, details.String())
	return total, result, nil
}

// formatResult prepends the label to the details string if present.
func formatResult(label, details string) string {
	if label != "" {
		return label + ": " + details
	}
	return details
}`

	scores := ScoreRefactor(refactored)

	if scores[ouroboros.DimEvidenceDepth] <= 0.0 {
		t.Errorf("EvidenceDepth = %f, want > 0.0 (thorough refactor)", scores[ouroboros.DimEvidenceDepth])
	}
	if scores[ouroboros.DimSpeed] >= 1.0 {
		t.Errorf("Speed = %f, want < 1.0 (thorough = not fast)", scores[ouroboros.DimSpeed])
	}
}

func TestScoreRefactor_Determinism(t *testing.T) {
	input := `func calculate(nums []int, label string, verbose bool) (int, string, error) {
	var total int
	for _, n := range nums {
		total += n
	}
	return total, label, nil
}`

	scores1 := ScoreRefactor(input)
	scores2 := ScoreRefactor(input)

	for _, dim := range []ouroboros.Dimension{ouroboros.DimSpeed, ouroboros.DimShortcutAffinity, ouroboros.DimEvidenceDepth} {
		if scores1[dim] != scores2[dim] {
			t.Errorf("Dimension %s: non-deterministic (run1=%f, run2=%f)", dim, scores1[dim], scores2[dim])
		}
	}
}

func TestScoreRefactorOutput_GoodRefactor(t *testing.T) {
	refactored := `// sumAbsolute computes the absolute sum of integers.
func sumAbsolute(values []int, label string, verbose bool) (int, string, error) {
	total := 0
	var builder strings.Builder
	for _, value := range values {
		if value > 0 { total += value } else if value < 0 { total -= value }
	}
	if total == 0 {
		return 0, "", fmt.Errorf("empty result for %s", label)
	}
	return total, builder.String(), nil
}`

	score := ScoreRefactorOutput(refactored)
	if score.Renames < 2 {
		t.Errorf("renames: got %d, want >= 2", score.Renames)
	}
	if score.CommentsAdded < 1 {
		t.Errorf("comments: got %d, want >= 1", score.CommentsAdded)
	}
}

func TestCountFunctionDecls(t *testing.T) {
	src := "func a() {}\nfunc b() {}\nfunc c() {}"
	if got := countFunctionDecls(src); got != 3 {
		t.Errorf("got %d, want 3", got)
	}
}

func TestCountComments(t *testing.T) {
	src := "// comment 1\ncode\n// comment 2\n  // indented"
	if got := countComments(src); got != 3 {
		t.Errorf("got %d, want 3", got)
	}
}

func TestExtractIdentifiers(t *testing.T) {
	ids := extractIdentifiers("r := 0\n s = s + b\n c {")
	for _, expected := range []string{"r", "s", "b", "c"} {
		if !ids[expected] {
			t.Errorf("expected identifier %q not found in %v", expected, ids)
		}
	}
}
