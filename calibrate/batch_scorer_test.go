package calibrate

import "testing"

func makeBatch(items ...map[string]any) []map[string]any { return items }

func TestBatchFieldMatch_Basic(t *testing.T) {
	batch := makeBatch(
		map[string]any{"actual": "pb001", "expected": "pb001", "rca_id": "R1"},
		map[string]any{"actual": "infra", "expected": "pb001", "rca_id": "R2"},
		map[string]any{"actual": "pb001", "expected": "pb001", "rca_id": ""},
	)
	val, detail, err := batchFieldMatch(batch, nil, map[string]any{
		"actual": "actual", "expected": "expected", "filter": "rca_id",
	})
	if err != nil {
		t.Fatal(err)
	}
	if val != 0.5 {
		t.Errorf("val = %v, want 0.5 (1/2 with filter)", val)
	}
	if detail != "1/2" {
		t.Errorf("detail = %q, want 1/2", detail)
	}
}

func TestBatchFieldMatch_FallbackContains(t *testing.T) {
	batch := makeBatch(
		map[string]any{"comp": "ptp4l", "expected_comp": "ptp4l", "msg": "", "rca_id": "R1"},
		map[string]any{"comp": "wrong", "expected_comp": "linuxptp", "msg": "the linuxptp daemon crashed", "rca_id": "R2"},
	)
	val, _, err := batchFieldMatch(batch, nil, map[string]any{
		"actual": "comp", "expected": "expected_comp",
		"fallback_text": "msg", "fallback_value": "expected_comp",
		"filter": "rca_id",
	})
	if err != nil {
		t.Fatal(err)
	}
	if val != 1.0 {
		t.Errorf("val = %v, want 1.0 (1 exact + 1 fallback)", val)
	}
}

func TestBatchFieldMatch_SliceComparison(t *testing.T) {
	batch := makeBatch(
		map[string]any{"actual_path": []string{"F0", "F1", "F2"}, "expected_path": []string{"F0", "F1", "F2"}},
		map[string]any{"actual_path": []string{"F0", "F1"}, "expected_path": []string{"F0", "F1", "F2"}},
	)
	val, _, err := batchFieldMatch(batch, nil, map[string]any{
		"actual": "actual_path", "expected": "expected_path",
	})
	if err != nil {
		t.Fatal(err)
	}
	if val != 0.5 {
		t.Errorf("val = %v, want 0.5 (1/2)", val)
	}
}

func TestBatchBoolRate_TrueFilter(t *testing.T) {
	batch := makeBatch(
		map[string]any{"expect_hit": true, "actual_hit": true},
		map[string]any{"expect_hit": true, "actual_hit": false},
		map[string]any{"expect_hit": false, "actual_hit": true},
	)
	val, detail, err := batchBoolRate(batch, nil, map[string]any{
		"filter_field": "expect_hit", "actual_field": "actual_hit",
	})
	if err != nil {
		t.Fatal(err)
	}
	if val != 0.5 {
		t.Errorf("val = %v, want 0.5 (1/2 where filter=true)", val)
	}
	if detail != "1/2" {
		t.Errorf("detail = %q, want 1/2", detail)
	}
}

func TestBatchBoolRate_FalseFilter(t *testing.T) {
	batch := makeBatch(
		map[string]any{"expect_hit": true, "actual_hit": true},
		map[string]any{"expect_hit": false, "actual_hit": true},
		map[string]any{"expect_hit": false, "actual_hit": false},
	)
	val, _, err := batchBoolRate(batch, nil, map[string]any{
		"filter_field": "expect_hit", "filter_value": false, "actual_field": "actual_hit",
	})
	if err != nil {
		t.Fatal(err)
	}
	if val != 0.5 {
		t.Errorf("val = %v, want 0.5 (1/2 where filter=false)", val)
	}
}

func TestBatchSetPrecision_Mean(t *testing.T) {
	batch := makeBatch(
		map[string]any{"selected": []string{"A", "B", "C"}, "relevant": []string{"A", "B"}},
		map[string]any{"selected": []string{"A", "D"}, "relevant": []string{"A", "B"}},
	)
	val, _, err := batchSetPrecision(batch, nil, map[string]any{
		"actual_field": "selected", "relevant_field": "relevant",
	})
	if err != nil {
		t.Fatal(err)
	}
	expected := (2.0/3.0 + 1.0/2.0) / 2.0
	if val < expected-0.01 || val > expected+0.01 {
		t.Errorf("val = %v, want ~%v", val, expected)
	}
}

func TestBatchSetPrecision_SumAggregate(t *testing.T) {
	batch := makeBatch(
		map[string]any{"actual": []string{"A", "B"}, "relevant": []string{"A"}},
		map[string]any{"actual": []string{"C", "D"}, "relevant": []string{"C"}},
	)
	val, _, err := batchSetPrecision(batch, nil, map[string]any{
		"actual_field": "actual", "relevant_field": "relevant", "aggregate": "sum",
	})
	if err != nil {
		t.Fatal(err)
	}
	if val != 0.5 {
		t.Errorf("val = %v, want 0.5 (2 relevant / 4 total)", val)
	}
}

func TestBatchSetRecall_Mean(t *testing.T) {
	batch := makeBatch(
		map[string]any{"selected": []string{"A"}, "relevant": []string{"A", "B"}, "rca_id": "R1"},
		map[string]any{"selected": []string{"A", "B"}, "relevant": []string{"A", "B"}, "rca_id": "R2"},
	)
	val, _, err := batchSetRecall(batch, nil, map[string]any{
		"actual_field": "selected", "relevant_field": "relevant", "filter": "rca_id",
	})
	if err != nil {
		t.Fatal(err)
	}
	expected := (0.5 + 1.0) / 2.0
	if val < expected-0.01 || val > expected+0.01 {
		t.Errorf("val = %v, want ~%v", val, expected)
	}
}

func TestBatchSetRecall_SumWithEvidence(t *testing.T) {
	batch := makeBatch(
		map[string]any{
			"actual": []string{"repo:file.go"}, "expected": []string{"repo:file.go", "repo:other.go"},
			"has_expected": true,
		},
	)
	val, _, err := batchSetRecall(batch, nil, map[string]any{
		"actual_field": "actual", "relevant_field": "expected",
		"match_fn": "evidence_overlap", "aggregate": "sum",
		"filter": "has_expected",
	})
	if err != nil {
		t.Fatal(err)
	}
	if val != 0.5 {
		t.Errorf("val = %v, want 0.5 (1/2 with evidence_overlap)", val)
	}
}

func TestBatchSetExclusion(t *testing.T) {
	batch := makeBatch(
		map[string]any{"selected": []string{"A", "B"}, "has_sel": true},
		map[string]any{"selected": []string{"A", "DECOY"}, "has_sel": true},
		map[string]any{"selected": []string{"C"}, "has_sel": true},
	)
	gt := map[string]any{"excluded": []string{"DECOY"}}
	val, _, err := batchSetExclusion(batch, gt, map[string]any{
		"actual_field": "selected", "excluded_field": "excluded", "filter": "has_sel",
	})
	if err != nil {
		t.Fatal(err)
	}
	expected := 1.0 - 1.0/3.0
	if val < expected-0.01 || val > expected+0.01 {
		t.Errorf("val = %v, want ~%v", val, expected)
	}
}

func TestBatchKeywordScore_Mean(t *testing.T) {
	batch := makeBatch(
		map[string]any{
			"msg": "the ptp4l clock was drifting", "keywords": []string{"ptp4l", "clock", "drift"},
			"threshold": 3, "rca_id": "R1",
		},
		map[string]any{
			"msg": "something unrelated", "keywords": []string{"ptp4l", "clock"},
			"threshold": 2, "rca_id": "R2",
		},
	)
	val, _, err := batchKeywordScore(batch, nil, map[string]any{
		"text_field": "msg", "keywords_field": "keywords",
		"threshold_field": "threshold", "filter": "rca_id",
	})
	if err != nil {
		t.Fatal(err)
	}
	// Item 1: 3/3 matched, min(3/3, 1.0)=1.0. Item 2: 0/2, min(0/2, 1.0)=0.0.
	// Average: 0.5
	if val < 0.49 || val > 0.51 {
		t.Errorf("val = %v, want ~0.5", val)
	}
}

func TestBatchKeywordScore_HitRate(t *testing.T) {
	batch := makeBatch(
		map[string]any{"msg": "ptp4l clock drift", "words": []string{"ptp4l", "clock", "drift", "test"}, "has_msg": true},
		map[string]any{"msg": "nothing here", "words": []string{"ptp4l", "clock"}, "has_msg": true},
	)
	val, _, err := batchKeywordScore(batch, nil, map[string]any{
		"text_field": "msg", "keywords_field": "words",
		"hit_threshold": 0.5, "aggregate": "hit_rate",
		"filter": "has_msg",
	})
	if err != nil {
		t.Fatal(err)
	}
	if val != 0.5 {
		t.Errorf("val = %v, want 0.5 (1/2 items meet 50%% threshold)", val)
	}
}

func TestBatchCorrelation(t *testing.T) {
	batch := makeBatch(
		map[string]any{"conv": 0.9, "correct": 1.0, "has_data": true},
		map[string]any{"conv": 0.8, "correct": 1.0, "has_data": true},
		map[string]any{"conv": 0.3, "correct": 0.0, "has_data": true},
	)
	val, _, err := batchCorrelation(batch, nil, map[string]any{
		"x_field": "conv", "y_field": "correct", "filter": "has_data",
	})
	if err != nil {
		t.Fatal(err)
	}
	if val < 0.5 {
		t.Errorf("val = %v, want positive correlation", val)
	}
}

func TestBatchSumRatio_Normal(t *testing.T) {
	batch := makeBatch(
		map[string]any{"actual_loops": 2, "expected_loops": 3},
		map[string]any{"actual_loops": 1, "expected_loops": 2},
	)
	val, _, err := batchSumRatio(batch, nil, map[string]any{
		"numerator_field": "actual_loops", "denominator_field": "expected_loops",
	})
	if err != nil {
		t.Fatal(err)
	}
	if val != 0.6 {
		t.Errorf("val = %v, want 0.6 (3/5)", val)
	}
}

func TestBatchSumRatio_BothZero(t *testing.T) {
	batch := makeBatch(
		map[string]any{"actual_loops": 0, "expected_loops": 0},
	)
	val, _, err := batchSumRatio(batch, nil, map[string]any{
		"numerator_field": "actual_loops", "denominator_field": "expected_loops",
	})
	if err != nil {
		t.Fatal(err)
	}
	if val != 1.0 {
		t.Errorf("val = %v, want 1.0 (zero_both default)", val)
	}
}

func TestBatchFieldSum_Measured(t *testing.T) {
	batch := makeBatch(
		map[string]any{"tokens": 5000, "steps": 6},
		map[string]any{"tokens": 3000, "steps": 4},
	)
	val, detail, err := batchFieldSum(batch, nil, map[string]any{
		"field": "tokens", "fallback_field": "steps", "fallback_multiplier": 1000,
	})
	if err != nil {
		t.Fatal(err)
	}
	if val != 8000 {
		t.Errorf("val = %v, want 8000", val)
	}
	if detail != "8000 (measured)" {
		t.Errorf("detail = %q", detail)
	}
}

func TestBatchFieldSum_Estimated(t *testing.T) {
	batch := makeBatch(
		map[string]any{"tokens": 0, "steps": 6},
		map[string]any{"tokens": 0, "steps": 4},
	)
	val, _, err := batchFieldSum(batch, nil, map[string]any{
		"field": "tokens", "fallback_field": "steps", "fallback_multiplier": 1000,
	})
	if err != nil {
		t.Fatal(err)
	}
	if val != 10000 {
		t.Errorf("val = %v, want 10000 (10 steps * 1000)", val)
	}
}

func TestBatchGroupLinkage(t *testing.T) {
	batch := makeBatch(
		map[string]any{"rca_id": "R1", "actual_rca_id": 42},
		map[string]any{"rca_id": "R1", "actual_rca_id": 42},
		map[string]any{"rca_id": "R1", "actual_rca_id": 99},
		map[string]any{"rca_id": "R2", "actual_rca_id": 10},
	)
	val, detail, err := batchGroupLinkage(batch, nil, map[string]any{
		"group_field": "rca_id", "value_field": "actual_rca_id", "filter": "rca_id",
	})
	if err != nil {
		t.Fatal(err)
	}
	// R1 group: 3 items, 2 expected links. Link 0-1: match (42==42). Link 0-2: no (42!=99).
	// R2 group: 1 item, no links.
	// 1/2 = 0.5
	if val != 0.5 {
		t.Errorf("val = %v, want 0.5 (1/2 links)", val)
	}
	if detail != "1/2" {
		t.Errorf("detail = %q, want 1/2", detail)
	}
}

func TestBatchScorers_RegisteredInDefault(t *testing.T) {
	reg := DefaultScorerRegistry()
	names := []string{
		"batch_field_match", "batch_bool_rate", "batch_set_precision",
		"batch_set_recall", "batch_set_exclusion", "batch_keyword_score",
		"batch_correlation", "batch_sum_ratio", "batch_field_sum",
		"batch_group_linkage",
	}
	for _, name := range names {
		if _, err := reg.Get(name); err != nil {
			t.Errorf("missing batch scorer %q: %v", name, err)
		}
	}
}

func TestBatchScorers_WrongType(t *testing.T) {
	scorers := []ScorerFunc{
		batchFieldMatch, batchBoolRate, batchSetPrecision,
		batchSetRecall, batchSetExclusion, batchKeywordScore,
		batchCorrelation, batchSumRatio, batchFieldSum,
		batchGroupLinkage,
	}
	for _, fn := range scorers {
		_, _, err := fn("not a batch", nil, map[string]any{})
		if err == nil {
			t.Error("should error on non-batch input")
		}
	}
}
