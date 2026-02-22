package metacal

import (
	"context"
	"fmt"
	"testing"

	"github.com/dpopsuev/origami"
)

func approxEqual(a, b, eps float64) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d < eps
}

func stubDispatcher(responses map[ProbeStep]string) Dispatcher {
	return func(_ context.Context, step ProbeStep, _ string) (string, error) {
		resp, ok := responses[step]
		if !ok {
			return "", fmt.Errorf("no stub response for step %s", step)
		}
		return resp, nil
	}
}

func TestRunSingleProbe_ReturnsResult(t *testing.T) {
	RegisterScorer(StepRefactor, func(raw string) map[Dimension]float64 {
		return map[Dimension]float64{DimSpeed: 0.7, DimEvidenceDepth: 0.3}
	})
	defer func() { delete(defaultScorers, StepRefactor) }()

	spec := ProbeSpec{
		ID:   "test-refactor",
		Step: StepRefactor,
		Input: "test input",
	}

	dispatcher := stubDispatcher(map[ProbeStep]string{
		StepRefactor: "refactored code here",
	})

	result, err := RunSingleProbe(context.Background(), dispatcher, spec)
	if err != nil {
		t.Fatalf("RunSingleProbe: %v", err)
	}

	if result.ProbeID != "test-refactor" {
		t.Errorf("ProbeID = %q, want test-refactor", result.ProbeID)
	}
	if result.RawOutput != "refactored code here" {
		t.Errorf("RawOutput = %q, want refactored code", result.RawOutput)
	}
	if result.DimensionScores[DimSpeed] != 0.7 {
		t.Errorf("DimSpeed = %f, want 0.7", result.DimensionScores[DimSpeed])
	}
	if result.Elapsed <= 0 {
		t.Error("Elapsed should be > 0")
	}
}

func TestRunSingleProbe_NoScorer(t *testing.T) {
	spec := ProbeSpec{
		ID:    "unscored",
		Step:  "UNKNOWN_STEP",
		Input: "test",
	}

	dispatcher := stubDispatcher(map[ProbeStep]string{
		"UNKNOWN_STEP": "some output",
	})

	result, err := RunSingleProbe(context.Background(), dispatcher, spec)
	if err != nil {
		t.Fatalf("RunSingleProbe: %v", err)
	}

	if result.DimensionScores != nil {
		t.Errorf("DimensionScores should be nil for unscored probe, got %v", result.DimensionScores)
	}
	if result.RawOutput != "some output" {
		t.Errorf("RawOutput = %q, want some output", result.RawOutput)
	}
}

func TestRunOuroboros_FullBattery(t *testing.T) {
	RegisterScorer(StepRefactor, func(_ string) map[Dimension]float64 {
		return map[Dimension]float64{DimSpeed: 0.8, DimEvidenceDepth: 0.6}
	})
	RegisterScorer(StepDebug, func(_ string) map[Dimension]float64 {
		return map[Dimension]float64{DimSpeed: 0.4, DimConvergenceThreshold: 0.9}
	})
	defer func() {
		delete(defaultScorers, StepRefactor)
		delete(defaultScorers, StepDebug)
	}()

	battery := OuroborosBattery(
		ProbeSpec{ID: "refactor-v1", Step: StepRefactor, Input: "messy code"},
		ProbeSpec{ID: "debug-v1", Step: StepDebug, Input: "log output"},
	)

	dispatcher := stubDispatcher(map[ProbeStep]string{
		StepRefactor: "clean code",
		StepDebug:    "root cause: goroutine leak",
	})

	model := framework.ModelIdentity{ModelName: "test-model", Provider: "test"}
	profile, err := RunOuroboros(context.Background(), model, dispatcher, battery)
	if err != nil {
		t.Fatalf("RunOuroboros: %v", err)
	}

	if profile.Model.ModelName != "test-model" {
		t.Errorf("Model = %q, want test-model", profile.Model.ModelName)
	}
	if profile.BatteryVersion != BatteryVersion {
		t.Errorf("BatteryVersion = %q, want %q", profile.BatteryVersion, BatteryVersion)
	}
	if len(profile.RawResults) != 2 {
		t.Fatalf("RawResults count = %d, want 2", len(profile.RawResults))
	}

	const eps = 1e-9
	wantSpeed := (0.8 + 0.4) / 2.0
	if got := profile.Dimensions[DimSpeed]; !approxEqual(got, wantSpeed, eps) {
		t.Errorf("DimSpeed = %f, want %f (average of two probes)", got, wantSpeed)
	}

	if got := profile.Dimensions[DimEvidenceDepth]; !approxEqual(got, 0.6, eps) {
		t.Errorf("DimEvidenceDepth = %f, want 0.6 (only one probe)", got)
	}

	if got := profile.Dimensions[DimConvergenceThreshold]; !approxEqual(got, 0.9, eps) {
		t.Errorf("DimConvergenceThreshold = %f, want 0.9 (only one probe)", got)
	}
}

func TestRunOuroboros_EmptyBattery(t *testing.T) {
	dispatcher := stubDispatcher(nil)
	model := framework.ModelIdentity{ModelName: "test", Provider: "test"}

	_, err := RunOuroboros(context.Background(), model, dispatcher, nil)
	if err == nil {
		t.Fatal("expected error for empty battery")
	}
}

func TestRunOuroboros_DispatcherError(t *testing.T) {
	dispatcher := func(_ context.Context, _ ProbeStep, _ string) (string, error) {
		return "", fmt.Errorf("connection refused")
	}

	battery := OuroborosBattery(
		ProbeSpec{ID: "failing", Step: StepRefactor, Input: "test"},
	)
	model := framework.ModelIdentity{ModelName: "test", Provider: "test"}

	_, err := RunOuroboros(context.Background(), model, dispatcher, battery)
	if err == nil {
		t.Fatal("expected error from dispatcher failure")
	}
}
