package probes

import (
	"testing"
)

func TestSelfVerifyScore_NoSignals(t *testing.T) {
	response := "Here is the function that adds two numbers."
	score := SelfVerifyScore(response)
	if score != 0.0 {
		t.Errorf("expected 0.0, got %f", score)
	}
}

func TestSelfVerifyScore_TestCases(t *testing.T) {
	response := `Here's my solution:
func Add(a, b int) int { return a + b }

Here's a test:
func TestAdd(t *testing.T) {
    if Add(1, 2) != 3 { t.Error("failed") }
}`
	score := SelfVerifyScore(response)
	if score < 0.25 {
		t.Errorf("expected >= 0.25 for test cases, got %f", score)
	}
}

func TestSelfVerifyScore_EdgeCases(t *testing.T) {
	response := `I've considered the edge case where the input is nil and the boundary
condition when the list is empty.`
	score := SelfVerifyScore(response)
	if score < 0.15 {
		t.Errorf("expected >= 0.15 for edge cases, got %f", score)
	}
}

func TestSelfVerifyScore_SelfCorrection(t *testing.T) {
	response := `Wait, actually, I made a mistake in the above. Let me fix the loop
condition to use <= instead of <.`
	score := SelfVerifyScore(response)
	if score < 0.20 {
		t.Errorf("expected >= 0.20 for self-correction, got %f", score)
	}
}

func TestSelfVerifyScore_MultipleSignals(t *testing.T) {
	response := `func Process(input []string) error {
    if len(input) == 0 { return fmt.Errorf("empty input") }
    // ... processing ...
    if err != nil { return err }
}

Wait, I should also handle the nil case. Let me verify...

Edge case: what if the list contains empty strings?

Note: this doesn't handle concurrent access — not thread-safe.

func TestProcess(t *testing.T) {
    err := Process(nil)
    assert.Error(t, err)
}`
	score := SelfVerifyScore(response)
	if score < 0.7 {
		t.Errorf("expected >= 0.7 for multiple signals, got %f", score)
	}
}

func TestSelfVerifyScore_ClampsToOne(t *testing.T) {
	response := `Let me verify this. Wait, actually let me check the edge case where
input is nil. func TestFoo(t *testing.T) { assert(true) }
This is a limitation. if err != nil { return err }
Let me trace through the code...`
	score := SelfVerifyScore(response)
	if score > 1.0 {
		t.Errorf("expected <= 1.0, got %f", score)
	}
}
