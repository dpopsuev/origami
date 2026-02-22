package metacal

import (
	"math"
	"testing"

	"github.com/dpopsuev/origami"
)

func TestElementMatch_FastModel_MapsToFireOrLightning(t *testing.T) {
	profile := ModelProfile{
		Dimensions: map[Dimension]float64{
			DimSpeed:                0.95,
			DimPersistence:          0.0,
			DimConvergenceThreshold: 0.4,
			DimShortcutAffinity:     0.9,
			DimEvidenceDepth:        0.1,
			DimFailureMode:          0.3,
		},
	}

	match := ElementMatch(profile)
	if match != framework.ElementFire && match != framework.ElementLightning {
		t.Errorf("expected fire or lightning for fast model, got %s", match)
	}
}

func TestElementMatch_ThoroughModel_MapsToEarthOrWater(t *testing.T) {
	profile := ModelProfile{
		Dimensions: map[Dimension]float64{
			DimSpeed:                0.2,
			DimPersistence:          0.8,
			DimConvergenceThreshold: 0.85,
			DimShortcutAffinity:     0.1,
			DimEvidenceDepth:        0.9,
			DimFailureMode:          0.6,
		},
	}

	match := ElementMatch(profile)
	if match != framework.ElementEarth && match != framework.ElementWater {
		t.Errorf("expected earth or water for thorough model, got %s", match)
	}
}

func TestElementScores_SumPositive(t *testing.T) {
	profile := ModelProfile{
		Dimensions: map[Dimension]float64{
			DimSpeed:                0.5,
			DimPersistence:          0.5,
			DimConvergenceThreshold: 0.5,
			DimShortcutAffinity:     0.5,
			DimEvidenceDepth:        0.5,
			DimFailureMode:          0.5,
		},
	}

	scores := ElementScores(profile)

	for _, e := range framework.AllElements() {
		s, ok := scores[e]
		if !ok {
			t.Errorf("missing score for element %s", e)
			continue
		}
		if s <= 0 || s > 1.0 {
			t.Errorf("score for %s = %f, want (0, 1.0]", e, s)
		}
	}

	var hasMax bool
	for _, s := range scores {
		if math.Abs(s-1.0) < 1e-9 {
			hasMax = true
		}
	}
	if !hasMax {
		t.Error("expected at least one score normalized to 1.0")
	}
}

func TestSuggestPersona_ReturnsTwoSuggestions(t *testing.T) {
	profile := ModelProfile{
		Dimensions: map[Dimension]float64{
			DimSpeed:                0.7,
			DimPersistence:          0.2,
			DimConvergenceThreshold: 0.6,
			DimShortcutAffinity:     0.8,
			DimEvidenceDepth:        0.3,
			DimFailureMode:          0.4,
		},
	}

	personas := SuggestPersona(profile)
	if len(personas) != 2 {
		t.Fatalf("expected 2 persona suggestions, got %d: %v", len(personas), personas)
	}
	for _, p := range personas {
		if p == "" {
			t.Error("persona suggestion should not be empty")
		}
	}
}

func TestDeriveStepAffinity_AllStepsPresent(t *testing.T) {
	profile := ModelProfile{
		Dimensions: map[Dimension]float64{
			DimSpeed:                0.6,
			DimPersistence:          0.4,
			DimConvergenceThreshold: 0.7,
			DimShortcutAffinity:     0.5,
			DimEvidenceDepth:        0.8,
			DimFailureMode:          0.3,
		},
	}

	affinity := DeriveStepAffinity(profile)

	expectedSteps := []string{"recall", "triage", "resolve", "investigate", "correlate", "review", "report"}
	for _, step := range expectedSteps {
		v, ok := affinity[step]
		if !ok {
			t.Errorf("missing affinity for step %s", step)
			continue
		}
		if v < 0 || v > 1.0 {
			t.Errorf("affinity[%s] = %f, want [0, 1.0]", step, v)
		}
	}
}

func TestDeriveStepAffinity_FastModel_HighRecall(t *testing.T) {
	profile := ModelProfile{
		Dimensions: map[Dimension]float64{
			DimSpeed:                0.95,
			DimPersistence:          0.1,
			DimConvergenceThreshold: 0.3,
			DimShortcutAffinity:     0.9,
			DimEvidenceDepth:        0.1,
			DimFailureMode:          0.2,
		},
	}

	affinity := DeriveStepAffinity(profile)

	if affinity["recall"] < affinity["investigate"] {
		t.Errorf("fast model should have higher recall (%f) than investigate (%f)",
			affinity["recall"], affinity["investigate"])
	}
}

func TestIronFromProfile_HighConvergence(t *testing.T) {
	profile := ModelProfile{
		Dimensions: map[Dimension]float64{
			DimConvergenceThreshold: 0.9,
			DimPersistence:          0.7,
			DimEvidenceDepth:        0.8,
		},
	}

	iron := IronFromProfile(profile)

	if iron.Element != framework.ElementIron {
		t.Errorf("Element = %q, want iron", iron.Element)
	}
	if iron.MaxLoops < 1 {
		t.Errorf("MaxLoops = %d, want >= 1 (high persistence)", iron.MaxLoops)
	}
	if iron.EvidenceDepth < 7 {
		t.Errorf("EvidenceDepth = %d, want >= 7 (high evidence)", iron.EvidenceDepth)
	}
}

func TestIronFromProfile_LowConvergence(t *testing.T) {
	profile := ModelProfile{
		Dimensions: map[Dimension]float64{
			DimConvergenceThreshold: 0.3,
			DimPersistence:          0.2,
			DimEvidenceDepth:        0.3,
		},
	}

	iron := IronFromProfile(profile)

	if iron.Element != framework.ElementIron {
		t.Errorf("Element = %q, want iron", iron.Element)
	}
	if iron.ConvergenceThreshold < 0.7 {
		t.Errorf("ConvergenceThreshold = %f, want >= 0.7 (low accuracy raises threshold)", iron.ConvergenceThreshold)
	}
}

func TestDeriveStepAffinity_DeepModel_HighInvestigate(t *testing.T) {
	profile := ModelProfile{
		Dimensions: map[Dimension]float64{
			DimSpeed:                0.1,
			DimPersistence:          0.9,
			DimConvergenceThreshold: 0.85,
			DimShortcutAffinity:     0.1,
			DimEvidenceDepth:        0.95,
			DimFailureMode:          0.6,
		},
	}

	affinity := DeriveStepAffinity(profile)

	if affinity["investigate"] < affinity["recall"] {
		t.Errorf("deep model should have higher investigate (%f) than recall (%f)",
			affinity["investigate"], affinity["recall"])
	}
}
