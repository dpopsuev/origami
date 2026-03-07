package ouroboros

import (
	"testing"
)

func TestMultiRoundNodes_ReturnsAllFamilies(t *testing.T) {
	seed := &Seed{
		Name:       "test",
		Version:    "1.0",
		Category:   CategorySkill,
		Dimensions: []Dimension{DimPersistence},
		Poles: map[string]Pole{
			"a": {Signal: "a", ElementAffinity: map[Dimension]float64{DimPersistence: 0.9}},
			"b": {Signal: "b", ElementAffinity: map[Dimension]float64{DimPersistence: 0.1}},
		},
		Context: "test",
		Rubric:  "test",
		Rounds:  3,
	}

	dispatch := func(_ interface{}, _ string, _ string) (string, error) {
		return "test", nil
	}
	_ = dispatch

	registry := MultiRoundNodes(seed, nil)

	families := []string{"ouroboros-generate", "ouroboros-subject-multiround", "ouroboros-judge-multiround"}
	for _, fam := range families {
		if _, ok := registry[fam]; !ok {
			t.Errorf("missing family %q in registry", fam)
		}
	}
}

func TestRoundTracker_Lifecycle(t *testing.T) {
	rt := &roundTracker{maxRounds: 3, currentRound: 0}

	if rt.isFinalRound() {
		t.Error("should not be final at round 0")
	}

	rt.advance()
	if rt.currentRound != 1 {
		t.Errorf("round = %d, want 1", rt.currentRound)
	}
	if rt.isFinalRound() {
		t.Error("should not be final at round 1 of 3")
	}

	rt.advance()
	rt.advance()
	if !rt.isFinalRound() {
		t.Error("should be final at round 3 of 3")
	}
}

func TestRoundTracker_ZeroRounds(t *testing.T) {
	rt := &roundTracker{maxRounds: 0, currentRound: 0}
	if !rt.isFinalRound() {
		t.Error("rounds=0 should be immediately final")
	}
}

func TestJudgeFeedback_TypeFields(t *testing.T) {
	fb := &JudgeFeedback{
		Round:          1,
		TotalRounds:    3,
		Feedback:       "You missed the deployment correlation",
		OriginalPrompt: "the original question",
	}
	if fb.Round != 1 {
		t.Errorf("Round = %d, want 1", fb.Round)
	}
	if fb.TotalRounds != 3 {
		t.Errorf("TotalRounds = %d, want 3", fb.TotalRounds)
	}
}
