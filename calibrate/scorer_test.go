package calibrate

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestScorerRegistry_RegisterAndGet(t *testing.T) {
	reg := make(ScorerRegistry)
	reg.Register("custom", func(_, _ any, _ map[string]any) (float64, string, error) {
		return 0.42, "test", nil
	})

	fn, err := reg.Get("custom")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	val, detail, err := fn(nil, nil, nil)
	if err != nil {
		t.Fatalf("scorer: %v", err)
	}
	if val != 0.42 {
		t.Errorf("value = %v, want 0.42", val)
	}
	if detail != "test" {
		t.Errorf("detail = %q, want test", detail)
	}
}

func TestScorerRegistry_GetMissing(t *testing.T) {
	reg := make(ScorerRegistry)
	_, err := reg.Get("nope")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestScorerRegistry_DuplicatePanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	reg := make(ScorerRegistry)
	reg.Register("dup", func(_, _ any, _ map[string]any) (float64, string, error) { return 0, "", nil })
	reg.Register("dup", func(_, _ any, _ map[string]any) (float64, string, error) { return 0, "", nil })
}

func TestDefaultScorerRegistry_HasBuiltins(t *testing.T) {
	reg := DefaultScorerRegistry()
	for _, name := range []string{"accuracy", "rate", "threshold_check"} {
		if _, err := reg.Get(name); err != nil {
			t.Errorf("missing built-in scorer %q: %v", name, err)
		}
	}
}

func TestAccuracyScorer_Match(t *testing.T) {
	result := map[string]any{"defect_type": "infrastructure"}
	truth := map[string]any{"expected_defect": "Infrastructure"}

	val, _, err := accuracyScorer(result, truth, map[string]any{
		"predicted": "defect_type",
		"expected":  "expected_defect",
	})
	if err != nil {
		t.Fatalf("scorer: %v", err)
	}
	if val != 1.0 {
		t.Errorf("value = %v, want 1.0 (case-insensitive match)", val)
	}
}

func TestAccuracyScorer_Mismatch(t *testing.T) {
	result := map[string]any{"defect_type": "product_bug"}
	truth := map[string]any{"expected_defect": "infrastructure"}

	val, _, err := accuracyScorer(result, truth, map[string]any{
		"predicted": "defect_type",
		"expected":  "expected_defect",
	})
	if err != nil {
		t.Fatalf("scorer: %v", err)
	}
	if val != 0.0 {
		t.Errorf("value = %v, want 0.0", val)
	}
}

func TestAccuracyScorer_MissingParams(t *testing.T) {
	_, _, err := accuracyScorer(map[string]any{}, map[string]any{}, map[string]any{})
	if err == nil {
		t.Fatal("expected error for missing params")
	}
}

func TestAccuracyScorer_Detail(t *testing.T) {
	result := map[string]any{"a": "x"}
	truth := map[string]any{"b": "x"}

	_, detail, err := accuracyScorer(result, truth, map[string]any{"predicted": "a", "expected": "b"})
	if err != nil {
		t.Fatalf("scorer: %v", err)
	}
	if detail == "" {
		t.Error("expected non-empty detail")
	}
}

func TestRateScorer_FullMatch(t *testing.T) {
	result := map[string]any{"repos": []any{"A", "B", "C"}}
	truth := map[string]any{"repos": []any{"A", "B"}}

	val, _, err := rateScorer(result, truth, map[string]any{"field": "repos"})
	if err != nil {
		t.Fatalf("scorer: %v", err)
	}
	if val != 1.0 {
		t.Errorf("value = %v, want 1.0", val)
	}
}

func TestRateScorer_PartialMatch(t *testing.T) {
	result := map[string]any{"repos": []any{"A", "D"}}
	truth := map[string]any{"repos": []any{"A", "B", "C"}}

	val, detail, err := rateScorer(result, truth, map[string]any{"field": "repos"})
	if err != nil {
		t.Fatalf("scorer: %v", err)
	}
	expected := 1.0 / 3.0
	if val < expected-0.01 || val > expected+0.01 {
		t.Errorf("value = %v, want ~%v", val, expected)
	}
	if detail != "1/3" {
		t.Errorf("detail = %q, want 1/3", detail)
	}
}

func TestRateScorer_EmptyTruth(t *testing.T) {
	result := map[string]any{"repos": []any{"A"}}
	truth := map[string]any{"repos": []any{}}

	val, _, err := rateScorer(result, truth, map[string]any{"field": "repos"})
	if err != nil {
		t.Fatalf("scorer: %v", err)
	}
	if val != 1.0 {
		t.Errorf("value = %v, want 1.0 (vacuously true)", val)
	}
}

func TestThresholdCheckScorer_Pass(t *testing.T) {
	result := map[string]any{"confidence": 0.85}

	val, _, err := thresholdCheckScorer(result, nil, map[string]any{
		"field": "confidence",
		"min":   0.80,
	})
	if err != nil {
		t.Fatalf("scorer: %v", err)
	}
	if val != 1.0 {
		t.Errorf("value = %v, want 1.0", val)
	}
}

func TestThresholdCheckScorer_Fail(t *testing.T) {
	result := map[string]any{"confidence": 0.3}

	val, _, err := thresholdCheckScorer(result, nil, map[string]any{
		"field": "confidence",
		"min":   0.80,
	})
	if err != nil {
		t.Fatalf("scorer: %v", err)
	}
	if val != 0.0 {
		t.Errorf("value = %v, want 0.0", val)
	}
}

func TestThresholdCheckScorer_Range(t *testing.T) {
	result := map[string]any{"loops": 2.5}

	val, _, err := thresholdCheckScorer(result, nil, map[string]any{
		"field": "loops",
		"min":   0.5,
		"max":   3.0,
	})
	if err != nil {
		t.Fatalf("scorer: %v", err)
	}
	if val != 1.0 {
		t.Errorf("value = %v, want 1.0", val)
	}
}

func TestScoreCard_ScoreCase(t *testing.T) {
	sc := ScoreCard{
		Name: "test",
		MetricDefs: []MetricDef{
			{
				ID: "M1", Name: "defect_accuracy",
				Tier: TierOutcome, Direction: HigherIsBetter,
				Threshold: 0.85, Weight: 0.5,
				Scorer: "accuracy",
				Params: map[string]any{"predicted": "defect", "expected": "expected_defect"},
			},
			{
				ID: "M2", Name: "manual_metric",
				Tier: TierOutcome, Direction: HigherIsBetter,
				Threshold: 0.70, Weight: 0.5,
			},
		},
	}

	reg := DefaultScorerRegistry()
	result := map[string]any{"defect": "infrastructure"}
	truth := map[string]any{"expected_defect": "Infrastructure"}

	values, details, err := sc.ScoreCase(result, truth, reg)
	if err != nil {
		t.Fatalf("ScoreCase: %v", err)
	}

	if values["M1"] != 1.0 {
		t.Errorf("M1 = %v, want 1.0", values["M1"])
	}
	if _, ok := values["M2"]; ok {
		t.Error("M2 should not be scored (no Scorer field)")
	}
	if details["M1"] == "" {
		t.Error("M1 detail should be non-empty")
	}
}

func TestMetricDef_ScorerFieldParsedFromYAML(t *testing.T) {
	data := `
scorecard: test
description: test
version: 1
metrics:
  - id: M1
    name: defect_accuracy
    tier: outcome
    direction: higher_is_better
    threshold: 0.85
    weight: 0.25
    scorer: accuracy
    params:
      predicted: defect_type
      expected: expected_defect
  - id: M2
    name: manual
    tier: outcome
    direction: higher_is_better
    threshold: 0.70
    weight: 0.25
`
	var sc ScoreCard
	if err := yaml.Unmarshal([]byte(data), &sc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if sc.MetricDefs[0].Scorer != "accuracy" {
		t.Errorf("M1 scorer = %q, want accuracy", sc.MetricDefs[0].Scorer)
	}
	if sc.MetricDefs[0].Params["predicted"] != "defect_type" {
		t.Errorf("M1 params.predicted = %v", sc.MetricDefs[0].Params["predicted"])
	}
	if sc.MetricDefs[1].Scorer != "" {
		t.Errorf("M2 scorer = %q, want empty", sc.MetricDefs[1].Scorer)
	}
}
