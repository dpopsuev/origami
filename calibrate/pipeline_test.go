package calibrate

import (
	"context"
	"fmt"
	"math"
	"testing"

	framework "github.com/dpopsuev/origami"
)

const floatTolerance = 1e-9

func approxEqual(a, b float64) bool {
	return math.Abs(a-b) < floatTolerance
}

func TestPipelineDef_LoadsAndBuilds(t *testing.T) {
	def, err := PipelineDef()
	if err != nil {
		t.Fatalf("PipelineDef: %v", err)
	}

	if def.Pipeline != "calibration-runner" {
		t.Errorf("pipeline = %q, want calibration-runner", def.Pipeline)
	}
	if len(def.Nodes) != 7 {
		t.Errorf("nodes = %d, want 7", len(def.Nodes))
	}
	if len(def.Edges) != 7 {
		t.Errorf("edges = %d, want 7", len(def.Edges))
	}

	edgeIDs := make([]string, len(def.Edges))
	for i, ed := range def.Edges {
		edgeIDs[i] = ed.ID
	}

	reg := framework.GraphRegistries{
		Nodes: CalibrationNodeRegistry(),
		Edges: forwardEdgeFactory(edgeIDs...),
	}

	graph, err := def.BuildGraphWith(reg)
	if err != nil {
		t.Fatalf("BuildGraphWith: %v", err)
	}

	if len(graph.Nodes()) != 7 {
		t.Errorf("graph nodes = %d, want 7", len(graph.Nodes()))
	}
}

func TestRunPipeline_EndToEnd(t *testing.T) {
	sc := NewScoreCardBuilder("test").
		WithMetrics(
			MetricDef{ID: "accuracy", Name: "Accuracy", Tier: TierOutcome, Direction: HigherIsBetter, Threshold: 0.50, Weight: 1.0},
			MetricDef{ID: "speed", Name: "Speed", Tier: TierEfficiency, Direction: LowerIsBetter, Threshold: 100, Weight: 0},
		).
		WithAggregate(AggregateConfig{
			ID: "overall", Name: "Overall", Formula: "weighted_average",
			Threshold: 0.50, Include: []string{"accuracy"},
		}).
		Build()

	input := &CalibrationInput{
		Scenario: "test-scenario",
		Adapter:  "stub",
		Runs:     1,
		Cases: []CaseInput{
			{ID: "case-1", Input: map[string]any{"data": "hello"}},
			{ID: "case-2", Input: map[string]any{"data": "world"}},
			{ID: "case-3", Input: map[string]any{"data": "test"}},
		},
		GroundTruth: map[string]any{
			"case-1": "expected-1",
			"case-2": "expected-2",
			"case-3": "expected-3",
		},
		ScoreCard: &sc,
		CaseRunner: CaseRunnerFunc(func(_ context.Context, caseID string, input any) (any, error) {
			return map[string]any{"result": caseID, "input": input}, nil
		}),
		CaseScorer: CaseScorerFunc(func(_, _ any) (map[string]float64, error) {
			return map[string]float64{"accuracy": 0.80, "speed": 50}, nil
		}),
	}

	report, err := RunPipeline(context.Background(), input)
	if err != nil {
		t.Fatalf("RunPipeline: %v", err)
	}

	if report.Scenario != "test-scenario" {
		t.Errorf("scenario = %q, want test-scenario", report.Scenario)
	}
	if report.Adapter != "stub" {
		t.Errorf("adapter = %q, want stub", report.Adapter)
	}

	byID := report.Metrics.ByID()

	acc, ok := byID["accuracy"]
	if !ok {
		t.Fatal("missing accuracy metric")
	}
	if !approxEqual(acc.Value, 0.80) {
		t.Errorf("accuracy value = %f, want 0.80", acc.Value)
	}
	if !acc.Pass {
		t.Error("accuracy should pass (0.80 >= 0.50)")
	}

	speed, ok := byID["speed"]
	if !ok {
		t.Fatal("missing speed metric")
	}
	if !approxEqual(speed.Value, 50) {
		t.Errorf("speed value = %f, want 50", speed.Value)
	}
	if !speed.Pass {
		t.Error("speed should pass (50 <= 100)")
	}

	agg, ok := byID["overall"]
	if !ok {
		t.Fatal("missing overall aggregate")
	}
	if !approxEqual(agg.Value, 0.80) {
		t.Errorf("overall value = %f, want 0.80", agg.Value)
	}
	if !agg.Pass {
		t.Error("overall should pass (0.80 >= 0.50)")
	}
}

func TestRunPipeline_ParallelExecution(t *testing.T) {
	sc := NewScoreCardBuilder("test").
		WithMetrics(MetricDef{ID: "m1", Name: "M1", Tier: TierOutcome, Direction: HigherIsBetter, Threshold: 0.0, Weight: 1.0}).
		Build()

	cases := make([]CaseInput, 10)
	gt := make(map[string]any, 10)
	for i := range cases {
		id := fmt.Sprintf("case-%d", i)
		cases[i] = CaseInput{ID: id, Input: i}
		gt[id] = i
	}

	input := &CalibrationInput{
		Scenario:  "parallel-test",
		Adapter:   "stub",
		Runs:      1,
		Cases:     cases,
		GroundTruth: gt,
		ScoreCard: &sc,
		Parallel:  4,
		CaseRunner: CaseRunnerFunc(func(_ context.Context, caseID string, _ any) (any, error) {
			return caseID, nil
		}),
		CaseScorer: CaseScorerFunc(func(_, _ any) (map[string]float64, error) {
			return map[string]float64{"m1": 1.0}, nil
		}),
	}

	report, err := RunPipeline(context.Background(), input)
	if err != nil {
		t.Fatalf("RunPipeline: %v", err)
	}

	m1 := report.Metrics.ByID()["m1"]
	if !approxEqual(m1.Value, 1.0) {
		t.Errorf("m1 = %f, want 1.0", m1.Value)
	}
}

func TestRunPipeline_CaseRunnerError(t *testing.T) {
	sc := NewScoreCardBuilder("test").
		WithMetrics(MetricDef{ID: "m1", Name: "M1", Tier: TierOutcome, Direction: HigherIsBetter, Threshold: 0.0, Weight: 1.0}).
		Build()

	input := &CalibrationInput{
		Scenario: "error-test",
		Adapter:  "stub",
		Runs:     1,
		Cases: []CaseInput{
			{ID: "good-case", Input: nil},
			{ID: "bad-case", Input: nil},
		},
		GroundTruth: map[string]any{"good-case": nil, "bad-case": nil},
		ScoreCard:   &sc,
		CaseRunner: CaseRunnerFunc(func(_ context.Context, caseID string, _ any) (any, error) {
			if caseID == "bad-case" {
				return nil, fmt.Errorf("simulated failure")
			}
			return "ok", nil
		}),
		CaseScorer: CaseScorerFunc(func(_, _ any) (map[string]float64, error) {
			return map[string]float64{"m1": 1.0}, nil
		}),
	}

	report, err := RunPipeline(context.Background(), input)
	if err != nil {
		t.Fatalf("RunPipeline: %v", err)
	}

	m1 := report.Metrics.ByID()["m1"]
	if !approxEqual(m1.Value, 1.0) {
		t.Errorf("m1 = %f, want 1.0 (only good-case scored)", m1.Value)
	}
}

func TestRunPipeline_WithObserver(t *testing.T) {
	sc := NewScoreCardBuilder("test").
		WithMetrics(MetricDef{ID: "m1", Name: "M1", Tier: TierOutcome, Direction: HigherIsBetter, Threshold: 0.0, Weight: 1.0}).
		Build()

	var events []framework.WalkEvent
	obs := framework.WalkObserverFunc(func(e framework.WalkEvent) {
		events = append(events, e)
	})

	input := &CalibrationInput{
		Scenario:    "observer-test",
		Adapter:     "stub",
		Runs:        1,
		Cases:       []CaseInput{{ID: "c1", Input: nil}},
		GroundTruth: map[string]any{"c1": nil},
		ScoreCard:   &sc,
		CaseRunner:  CaseRunnerFunc(func(_ context.Context, _ string, _ any) (any, error) { return "ok", nil }),
		CaseScorer:  CaseScorerFunc(func(_, _ any) (map[string]float64, error) { return map[string]float64{"m1": 1.0}, nil }),
	}

	_, err := RunPipeline(context.Background(), input, WithObserver(obs))
	if err != nil {
		t.Fatalf("RunPipeline: %v", err)
	}

	if len(events) == 0 {
		t.Error("expected walk events from observer")
	}

	nodesSeen := make(map[string]bool)
	for _, e := range events {
		if e.Node != "" {
			nodesSeen[e.Node] = true
		}
	}

	for _, name := range []string{"load_scenario", "fan_out", "walk_case", "score_case", "fan_in", "aggregate", "report"} {
		if !nodesSeen[name] {
			t.Errorf("missing event for node %q", name)
		}
	}
}

func TestRunPipeline_MissingInput(t *testing.T) {
	_, err := RunPipeline(context.Background(), &CalibrationInput{})
	if err == nil {
		t.Fatal("expected error for empty input")
	}
}

func TestRunPipeline_IdenticalToProceduralScoring(t *testing.T) {
	sc := NewScoreCardBuilder("procedural-compare").
		WithMetrics(
			MetricDef{ID: "m1", Name: "M1", Tier: TierOutcome, Direction: HigherIsBetter, Threshold: 0.70, Weight: 0.6},
			MetricDef{ID: "m2", Name: "M2", Tier: TierEfficiency, Direction: LowerIsBetter, Threshold: 100, Weight: 0.4},
		).
		WithAggregate(AggregateConfig{
			ID: "agg", Name: "Aggregate", Formula: "weighted_average",
			Threshold: 0.50, Include: []string{"m1", "m2"},
		}).
		Build()

	caseScores := map[string]map[string]float64{
		"c1": {"m1": 0.90, "m2": 50},
		"c2": {"m1": 0.70, "m2": 80},
		"c3": {"m1": 0.80, "m2": 60},
	}

	input := &CalibrationInput{
		Scenario:    "compare",
		Adapter:     "stub",
		Runs:        1,
		Cases:       []CaseInput{{ID: "c1"}, {ID: "c2"}, {ID: "c3"}},
		GroundTruth: map[string]any{"c1": nil, "c2": nil, "c3": nil},
		ScoreCard:   &sc,
		CaseRunner:  CaseRunnerFunc(func(_ context.Context, id string, _ any) (any, error) { return id, nil }),
		CaseScorer: CaseScorerFunc(func(result, _ any) (map[string]float64, error) {
			return caseScores[result.(string)], nil
		}),
	}

	pipelineReport, err := RunPipeline(context.Background(), input)
	if err != nil {
		t.Fatalf("RunPipeline: %v", err)
	}

	// Procedural path: compute averages, evaluate via ScoreCard
	avgM1 := (0.90 + 0.70 + 0.80) / 3
	avgM2 := (50.0 + 80.0 + 60.0) / 3

	proceduralMS := sc.Evaluate(
		map[string]float64{"m1": avgM1, "m2": avgM2},
		nil,
	)
	proceduralAgg, err := sc.ComputeAggregate(proceduralMS)
	if err != nil {
		t.Fatalf("ComputeAggregate: %v", err)
	}

	pipelineByID := pipelineReport.Metrics.ByID()
	proceduralByID := proceduralMS.ByID()

	for _, id := range []string{"m1", "m2"} {
		pVal := pipelineByID[id].Value
		procVal := proceduralByID[id].Value
		if !approxEqual(pVal, procVal) {
			t.Errorf("metric %s: pipeline=%f, procedural=%f", id, pVal, procVal)
		}
		if pipelineByID[id].Pass != proceduralByID[id].Pass {
			t.Errorf("metric %s: pipeline pass=%v, procedural pass=%v", id, pipelineByID[id].Pass, proceduralByID[id].Pass)
		}
	}

	pipelineAgg := pipelineByID["agg"]
	if !approxEqual(pipelineAgg.Value, proceduralAgg.Value) {
		t.Errorf("aggregate: pipeline=%f, procedural=%f", pipelineAgg.Value, proceduralAgg.Value)
	}
	if pipelineAgg.Pass != proceduralAgg.Pass {
		t.Errorf("aggregate pass: pipeline=%v, procedural=%v", pipelineAgg.Pass, proceduralAgg.Pass)
	}
}
