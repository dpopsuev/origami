package probes

import (
	"testing"

	"github.com/dpopsuev/origami/metacal"
)

func TestRefactorSpec(t *testing.T) {
	spec := RefactorSpec()
	if spec.ID != "refactor-v1" {
		t.Errorf("ID = %q, want refactor-v1", spec.ID)
	}
	if spec.Step != metacal.StepRefactor {
		t.Errorf("Step = %q, want %q", spec.Step, metacal.StepRefactor)
	}
	if len(spec.Dimensions) != 3 {
		t.Errorf("Dimensions count = %d, want 3", len(spec.Dimensions))
	}
	if spec.Input == "" {
		t.Error("Input should contain the messy code")
	}
}

func TestScoreRefactor_NoChanges(t *testing.T) {
	scores := ScoreRefactor(metacal.MessyInput)

	if scores[metacal.DimEvidenceDepth] != 0.0 {
		t.Errorf("EvidenceDepth = %f, want 0.0 (no changes)", scores[metacal.DimEvidenceDepth])
	}
	if scores[metacal.DimSpeed] != 1.0 {
		t.Errorf("Speed = %f, want 1.0 (no changes = fast)", scores[metacal.DimSpeed])
	}
	if scores[metacal.DimShortcutAffinity] != 1.0 {
		t.Errorf("ShortcutAffinity = %f, want 1.0 (no changes = shortcut)", scores[metacal.DimShortcutAffinity])
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

	if scores[metacal.DimEvidenceDepth] <= 0.0 {
		t.Errorf("EvidenceDepth = %f, want > 0.0 (thorough refactor)", scores[metacal.DimEvidenceDepth])
	}
	if scores[metacal.DimSpeed] >= 1.0 {
		t.Errorf("Speed = %f, want < 1.0 (thorough = not fast)", scores[metacal.DimSpeed])
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

	for _, dim := range []metacal.Dimension{metacal.DimSpeed, metacal.DimShortcutAffinity, metacal.DimEvidenceDepth} {
		if scores1[dim] != scores2[dim] {
			t.Errorf("Dimension %s: non-deterministic (run1=%f, run2=%f)", dim, scores1[dim], scores2[dim])
		}
	}
}
