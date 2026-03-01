package probes

import (
	"math"
	"testing"

	"github.com/dpopsuev/origami/ouroboros"
)

func TestScoreGBWP_CorrectHighConfidence(t *testing.T) {
	raw := "VERDICT: CORRECT\nCONFIDENCE: 0.92\nJUSTIFICATION: Race condition confirmed by all evidence."
	scores := ScoreGBWP(raw)
	if scores[ouroboros.DimGBWP] != 1.0 {
		t.Errorf("expected 1.0, got %f", scores[ouroboros.DimGBWP])
	}
}

func TestScoreGBWP_CorrectMediumConfidence(t *testing.T) {
	raw := "VERDICT: CORRECT\nCONFIDENCE: 0.75\nJUSTIFICATION: Likely race."
	scores := ScoreGBWP(raw)
	if scores[ouroboros.DimGBWP] != 0.75 {
		t.Errorf("expected 0.75, got %f", scores[ouroboros.DimGBWP])
	}
}

func TestScoreGBWP_CorrectLowConfidence(t *testing.T) {
	raw := "VERDICT: CORRECT\nCONFIDENCE: 0.5\nJUSTIFICATION: Maybe."
	scores := ScoreGBWP(raw)
	if scores[ouroboros.DimGBWP] != 0.5 {
		t.Errorf("expected 0.5, got %f", scores[ouroboros.DimGBWP])
	}
}

func TestScoreGBWP_IncorrectVerdict(t *testing.T) {
	raw := "VERDICT: INCORRECT\nCONFIDENCE: 0.95\nJUSTIFICATION: Not a race."
	scores := ScoreGBWP(raw)
	if scores[ouroboros.DimGBWP] != 0.0 {
		t.Errorf("expected 0.0, got %f", scores[ouroboros.DimGBWP])
	}
}

func TestComputeGBWP_LinearImprovement(t *testing.T) {
	points := []GBWPPoint{
		{Rounds: 1, Accuracy: 0.0},
		{Rounds: 8, Accuracy: 1.0},
	}
	gbwp := ComputeGBWP(points)
	if math.Abs(gbwp-0.5) > 0.01 {
		t.Errorf("expected ~0.5 (triangle AUC), got %f", gbwp)
	}
}

func TestComputeGBWP_PerfectAccuracy(t *testing.T) {
	points := []GBWPPoint{
		{Rounds: 1, Accuracy: 1.0},
		{Rounds: 2, Accuracy: 1.0},
		{Rounds: 4, Accuracy: 1.0},
		{Rounds: 8, Accuracy: 1.0},
	}
	gbwp := ComputeGBWP(points)
	if gbwp != 1.0 {
		t.Errorf("expected 1.0, got %f", gbwp)
	}
}

func TestComputeGBWP_RealWorldCurve(t *testing.T) {
	points := []GBWPPoint{
		{Rounds: 1, Accuracy: 0.6},
		{Rounds: 2, Accuracy: 0.75},
		{Rounds: 4, Accuracy: 0.85},
		{Rounds: 6, Accuracy: 0.88},
		{Rounds: 8, Accuracy: 0.90},
	}
	gbwp := ComputeGBWP(points)
	if gbwp < 0.7 || gbwp > 0.95 {
		t.Errorf("expected GBWP in [0.7, 0.95], got %f", gbwp)
	}
}

func TestComputeGBWP_SinglePoint(t *testing.T) {
	gbwp := ComputeGBWP([]GBWPPoint{{Rounds: 4, Accuracy: 0.8}})
	if gbwp != 0.8 {
		t.Errorf("expected 0.8 for single point, got %f", gbwp)
	}
}

func TestComputeGBWP_Empty(t *testing.T) {
	gbwp := ComputeGBWP(nil)
	if gbwp != 0 {
		t.Errorf("expected 0 for empty, got %f", gbwp)
	}
}

func TestBuildGBWPPrompt(t *testing.T) {
	s := DefaultStimuli()["gbwp"]
	prompt := BuildGBWPPrompt(s)
	if prompt == "" {
		t.Error("expected non-empty prompt")
	}
}
