package calibrate

import (
	"testing"
)

func syntheticResults() []StepResult {
	return []StepResult{
		{CaseID: "C1", Step: "F0_RECALL", Metrics: MetricSet{Metrics: []Metric{{ID: "acc", Value: 0.9, Threshold: 0.8, Pass: true}}}},
		{CaseID: "C2", Step: "F0_RECALL", Metrics: MetricSet{Metrics: []Metric{{ID: "acc", Value: 0.8, Threshold: 0.8, Pass: true}}}},
		{CaseID: "C3", Step: "F0_RECALL", Metrics: MetricSet{Metrics: []Metric{{ID: "acc", Value: 0.7, Threshold: 0.8, Pass: false}}}},
		{CaseID: "C1", Step: "F1_TRIAGE", Metrics: MetricSet{Metrics: []Metric{{ID: "acc", Value: 0.4, Threshold: 0.8, Pass: false}}}},
		{CaseID: "C2", Step: "F1_TRIAGE", Metrics: MetricSet{Metrics: []Metric{{ID: "acc", Value: 0.3, Threshold: 0.8, Pass: false}}}},
		{CaseID: "C3", Step: "F1_TRIAGE", Metrics: MetricSet{Metrics: []Metric{{ID: "acc", Value: 0.5, Threshold: 0.8, Pass: false}}}},
		{CaseID: "C1", Step: "F5_REVIEW", Metrics: MetricSet{Metrics: []Metric{{ID: "acc", Value: 1.0, Threshold: 0.8, Pass: true}}}},
		{CaseID: "C2", Step: "F5_REVIEW", Metrics: MetricSet{Metrics: []Metric{{ID: "acc", Value: 0.95, Threshold: 0.8, Pass: true}}}},
	}
}

func simpleScorer(_, _ string, ms MetricSet) float64 {
	if len(ms.Metrics) == 0 {
		return 0
	}
	return ms.Metrics[0].Value
}

func TestStepAnalyzer_Analyze_RanksByAccuracy(t *testing.T) {
	analyzer := NewStepAnalyzer(simpleScorer)
	rankings := analyzer.Analyze(syntheticResults())

	if len(rankings) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(rankings))
	}

	// Worst performer first
	if rankings[0].Step != "F1_TRIAGE" {
		t.Errorf("expected F1_TRIAGE as worst performer, got %q", rankings[0].Step)
	}
	if rankings[0].Rank != 1 {
		t.Errorf("expected rank 1, got %d", rankings[0].Rank)
	}

	// F0_RECALL should be middle
	if rankings[1].Step != "F0_RECALL" {
		t.Errorf("expected F0_RECALL as middle performer, got %q", rankings[1].Step)
	}

	// F5_REVIEW should be best
	if rankings[2].Step != "F5_REVIEW" {
		t.Errorf("expected F5_REVIEW as best performer, got %q", rankings[2].Step)
	}
}

func TestStepAnalyzer_Analyze_SamplesCount(t *testing.T) {
	analyzer := NewStepAnalyzer(simpleScorer)
	rankings := analyzer.Analyze(syntheticResults())

	for _, sa := range rankings {
		switch sa.Step {
		case "F0_RECALL":
			if sa.Samples != 3 {
				t.Errorf("F0_RECALL: expected 3 samples, got %d", sa.Samples)
			}
		case "F1_TRIAGE":
			if sa.Samples != 3 {
				t.Errorf("F1_TRIAGE: expected 3 samples, got %d", sa.Samples)
			}
		case "F5_REVIEW":
			if sa.Samples != 2 {
				t.Errorf("F5_REVIEW: expected 2 samples, got %d", sa.Samples)
			}
		}
	}
}

func TestStepAnalyzer_Analyze_EmptyInput(t *testing.T) {
	analyzer := NewStepAnalyzer(simpleScorer)
	rankings := analyzer.Analyze(nil)
	if len(rankings) != 0 {
		t.Errorf("expected 0 rankings for nil input, got %d", len(rankings))
	}
}

func TestPromptCalibrationLoop_FullCycle(t *testing.T) {
	analyzer := NewStepAnalyzer(simpleScorer)
	rankings := analyzer.Analyze(syntheticResults())

	proposer := func(sa StepAccuracy) *TuningProposal {
		return &TuningProposal{
			Step:          sa.Step,
			CurrentHash:   PromptHash("original prompt for " + sa.Step),
			Suggestion:    "improve " + sa.Step + " prompt",
			Rationale:     "accuracy is " + formatFloat(sa.Accuracy),
			ExpectedDelta: 0.1,
		}
	}

	loop := NewPromptCalibrationLoop(rankings, 0.85, proposer)

	// F1_TRIAGE (0.4) and F0_RECALL (0.8) are below 0.85
	// F5_REVIEW (0.975) is above threshold
	if loop.Remaining() != 2 {
		t.Fatalf("expected 2 proposals, got %d", loop.Remaining())
	}

	// Accept first proposal
	p := loop.Next()
	if p == nil {
		t.Fatal("expected non-nil proposal")
	}
	if p.Step != "F1_TRIAGE" {
		t.Errorf("expected F1_TRIAGE proposal first, got %q", p.Step)
	}
	loop.Accept()

	// Reject second proposal
	p = loop.Next()
	if p == nil {
		t.Fatal("expected non-nil proposal")
	}
	if p.Step != "F0_RECALL" {
		t.Errorf("expected F0_RECALL proposal second, got %q", p.Step)
	}
	loop.Reject("acceptable accuracy")

	// No more proposals
	if loop.HasNext() {
		t.Error("expected no more proposals")
	}
	if loop.Next() != nil {
		t.Error("Next() should return nil when exhausted")
	}

	// Check history
	history := loop.History()
	if len(history) != 2 {
		t.Fatalf("expected 2 history entries, got %d", len(history))
	}
	if !history[0].Accepted {
		t.Error("first proposal should be accepted")
	}
	if history[1].Accepted {
		t.Error("second proposal should be rejected")
	}
	if history[1].RejectReason != "acceptable accuracy" {
		t.Errorf("expected reject reason, got %q", history[1].RejectReason)
	}

	accepted := loop.AcceptedProposals()
	if len(accepted) != 1 {
		t.Errorf("expected 1 accepted proposal, got %d", len(accepted))
	}
}

func TestPromptCalibrationLoop_AllAboveThreshold(t *testing.T) {
	analyzer := NewStepAnalyzer(simpleScorer)
	rankings := analyzer.Analyze(syntheticResults())

	proposer := func(sa StepAccuracy) *TuningProposal {
		return &TuningProposal{Step: sa.Step}
	}

	loop := NewPromptCalibrationLoop(rankings, 0.0, proposer)
	if loop.Remaining() != 0 {
		t.Errorf("expected 0 proposals when all above threshold, got %d", loop.Remaining())
	}
}

func TestPromptCalibrationLoop_NilProposer(t *testing.T) {
	rankings := []StepAccuracy{{Step: "F0", Accuracy: 0.3, Samples: 5}}

	loop := NewPromptCalibrationLoop(rankings, 0.8, func(_ StepAccuracy) *TuningProposal {
		return nil
	})
	if loop.Remaining() != 0 {
		t.Errorf("expected 0 proposals when proposer returns nil, got %d", loop.Remaining())
	}
}

func TestPromptHash_Deterministic(t *testing.T) {
	h1 := PromptHash("test prompt")
	h2 := PromptHash("test prompt")
	if h1 != h2 {
		t.Errorf("same input should produce same hash: %q vs %q", h1, h2)
	}
	h3 := PromptHash("different prompt")
	if h1 == h3 {
		t.Error("different inputs should produce different hashes")
	}
	if len(h1) != 16 {
		t.Errorf("expected 16-char hex hash (8 bytes), got %d chars: %q", len(h1), h1)
	}
}

func TestPromptCalibrationLoop_AcceptRejectIdempotent(t *testing.T) {
	rankings := []StepAccuracy{{Step: "F0", Accuracy: 0.3, Samples: 5}}
	proposer := func(sa StepAccuracy) *TuningProposal {
		return &TuningProposal{Step: sa.Step}
	}
	loop := NewPromptCalibrationLoop(rankings, 0.8, proposer)

	loop.Accept()
	// Calling Accept/Reject after exhaustion should be safe
	loop.Accept()
	loop.Reject("extra")
	if len(loop.History()) != 1 {
		t.Errorf("expected 1 history entry, got %d", len(loop.History()))
	}
}

func formatFloat(v float64) string {
	return string([]byte{
		byte('0' + int(v*10)/10),
		'.',
		byte('0' + int(v*100)%10),
		byte('0' + int(v*1000)%10),
	})
}
