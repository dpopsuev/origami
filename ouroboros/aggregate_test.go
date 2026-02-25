package ouroboros

import (
	"testing"

	framework "github.com/dpopsuev/origami"
)

func approxEqual(a, b, eps float64) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d < eps
}

func TestProfileFromPoleResults_Aggregation(t *testing.T) {
	results := []PoleResult{
		{
			SelectedPole: "systematic",
			Confidence:   0.9,
			DimensionScores: map[Dimension]float64{
				DimSpeed:         0.3,
				DimEvidenceDepth: 0.9,
			},
		},
		{
			SelectedPole: "methodical",
			Confidence:   0.8,
			DimensionScores: map[Dimension]float64{
				DimSpeed:         0.5,
				DimEvidenceDepth: 0.7,
				DimPersistence:   0.8,
			},
		},
	}

	model := framework.ModelIdentity{ModelName: "test-model", Provider: "test"}
	profile := ProfileFromPoleResults(model, results, []string{"probe-a", "probe-b"})

	if profile.BatteryVersion != SeedBatteryVersion {
		t.Errorf("battery version = %q, want %q", profile.BatteryVersion, SeedBatteryVersion)
	}
	if len(profile.RawResults) != 2 {
		t.Errorf("raw results = %d, want 2", len(profile.RawResults))
	}

	wantSpeed := (0.3 + 0.5) / 2
	if !approxEqual(profile.Dimensions[DimSpeed], wantSpeed, 0.01) {
		t.Errorf("speed = %v, want %v", profile.Dimensions[DimSpeed], wantSpeed)
	}

	wantEvidence := (0.9 + 0.7) / 2
	if !approxEqual(profile.Dimensions[DimEvidenceDepth], wantEvidence, 0.01) {
		t.Errorf("evidence_depth = %v, want %v", profile.Dimensions[DimEvidenceDepth], wantEvidence)
	}

	wantPersistence := 0.8
	if !approxEqual(profile.Dimensions[DimPersistence], wantPersistence, 0.01) {
		t.Errorf("persistence = %v, want %v", profile.Dimensions[DimPersistence], wantPersistence)
	}
}

func TestProfileFromPoleResults_ElementMatchWorks(t *testing.T) {
	results := []PoleResult{
		{
			SelectedPole: "deep",
			Confidence:   0.9,
			DimensionScores: map[Dimension]float64{
				DimSpeed:                0.2,
				DimPersistence:          1.0,
				DimConvergenceThreshold: 0.85,
				DimShortcutAffinity:     0.1,
				DimEvidenceDepth:        0.8,
				DimFailureMode:          0.5,
			},
		},
	}

	model := framework.ModelIdentity{ModelName: "deep-thinker", Provider: "test"}
	profile := ProfileFromPoleResults(model, results, []string{"deep-probe"})

	if profile.ElementMatch == "" {
		t.Fatal("ElementMatch is empty")
	}
	if len(profile.ElementScores) == 0 {
		t.Fatal("ElementScores is empty")
	}
	if len(profile.SuggestedPersonas) == 0 {
		t.Fatal("SuggestedPersonas is empty")
	}

	t.Logf("ElementMatch: %s", profile.ElementMatch)
	t.Logf("SuggestedPersonas: %v", profile.SuggestedPersonas)
}

func TestProfileFromPoleResults_DeriveStepAffinityWorks(t *testing.T) {
	results := []PoleResult{
		{
			SelectedPole: "systematic",
			Confidence:   0.9,
			DimensionScores: map[Dimension]float64{
				DimSpeed:                0.4,
				DimPersistence:          0.6,
				DimConvergenceThreshold: 0.7,
				DimShortcutAffinity:     0.3,
				DimEvidenceDepth:        0.8,
				DimFailureMode:          0.5,
			},
		},
	}

	model := framework.ModelIdentity{ModelName: "balanced", Provider: "test"}
	profile := ProfileFromPoleResults(model, results, []string{"balanced-probe"})

	affinity := DeriveStepAffinity(profile)
	if len(affinity) == 0 {
		t.Fatal("DeriveStepAffinity returned empty map")
	}

	expectedSteps := []string{"recall", "triage", "resolve", "investigate", "correlate", "review", "report"}
	for _, step := range expectedSteps {
		if _, ok := affinity[step]; !ok {
			t.Errorf("missing step affinity for %q", step)
		}
	}

	if affinity["investigate"] <= 0 {
		t.Error("investigate affinity should be > 0 with evidence_depth=0.8")
	}
}

func TestPoleResultToProbeResult_Fields(t *testing.T) {
	pr := &PoleResult{
		SelectedPole: "systematic",
		Confidence:   0.85,
		DimensionScores: map[Dimension]float64{
			DimSpeed:         0.3,
			DimEvidenceDepth: 0.9,
		},
		Reasoning: "Shows thorough analysis",
	}

	result := PoleResultToProbeResult("test-seed", pr, 0)
	if result.ProbeID != "test-seed" {
		t.Errorf("ProbeID = %q, want test-seed", result.ProbeID)
	}
	if result.DimensionScores[DimSpeed] != 0.3 {
		t.Errorf("speed = %v, want 0.3", result.DimensionScores[DimSpeed])
	}
	if result.RawOutput != "Shows thorough analysis" {
		t.Errorf("RawOutput = %q, want reasoning text", result.RawOutput)
	}
}
