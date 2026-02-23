package ouroboros

import (
	"context"
	"fmt"
	"time"

	"github.com/dpopsuev/origami"
)

const BatteryVersion = "ouroboros-v1"

// OuroborosBattery returns the standard battery of all 5 probe specs.
// Callers provide the specs because the probes/ package is a separate Go
// package; this function stitches them into a battery.
func OuroborosBattery(specs ...ProbeSpec) []ProbeSpec {
	return specs
}

// RunOuroboros executes the full Ouroboros cycle: dispatches each probe,
// collects results, and assembles a ModelProfile. The caller must identify
// the model separately (e.g. via a discovery probe); RunOuroboros only
// measures behavior.
func RunOuroboros(
	ctx context.Context,
	model framework.ModelIdentity,
	dispatcher Dispatcher,
	battery []ProbeSpec,
) (ModelProfile, error) {
	if len(battery) == 0 {
		return ModelProfile{}, fmt.Errorf("empty battery: nothing to run")
	}

	start := time.Now()
	profile := ModelProfile{
		Model:          model,
		BatteryVersion: BatteryVersion,
		Timestamp:      start,
		Dimensions:     make(map[Dimension]float64),
		ElementScores:  make(map[framework.Element]float64),
	}

	var totalTokens int
	var totalLatency time.Duration

	for _, spec := range battery {
		result, err := RunSingleProbe(ctx, dispatcher, spec)
		if err != nil {
			return ModelProfile{}, fmt.Errorf("probe %s failed: %w", spec.ID, err)
		}
		profile.RawResults = append(profile.RawResults, result)
		totalTokens += result.TokensUsed
		totalLatency += result.Elapsed
	}

	aggregateDimensions(&profile)

	profile.CostProfile = framework.CostProfile{
		TokensPerStep: totalTokens / max(len(battery), 1),
		LatencyMs:     int(totalLatency.Milliseconds()) / max(len(battery), 1),
	}

	return profile, nil
}

// RunSingleProbe dispatches one probe and scores the response. The scorer
// function is looked up from the ProbeSpec's Step. If no scorer is registered,
// the result has empty DimensionScores (the raw output is still captured).
func RunSingleProbe(
	ctx context.Context,
	dispatcher Dispatcher,
	spec ProbeSpec,
) (ProbeResult, error) {
	start := time.Now()

	raw, err := dispatcher(ctx, spec.Step, spec.Input)
	if err != nil {
		return ProbeResult{}, fmt.Errorf("dispatch probe %s: %w", spec.ID, err)
	}

	elapsed := time.Since(start)

	result := ProbeResult{
		ProbeID:   spec.ID,
		RawOutput: raw,
		Elapsed:   elapsed,
	}

	if scorer, ok := defaultScorers[spec.Step]; ok {
		result.DimensionScores = scorer(raw)
	}

	return result, nil
}

// Scorer is a function that extracts dimension scores from a probe response.
type Scorer func(raw string) map[Dimension]float64

// defaultScorers is populated by RegisterScorer. The probes/ package calls
// RegisterScorer at init time to wire its scorers into the runner.
var defaultScorers = map[ProbeStep]Scorer{}

// RegisterScorer registers a dimension scorer for a probe step.
// Called by the probes/ package during initialization.
func RegisterScorer(step ProbeStep, scorer Scorer) {
	defaultScorers[step] = scorer
}

// aggregateDimensions averages dimension scores across all probe results.
// Each dimension's final score is the mean of all probes that measured it.
func aggregateDimensions(profile *ModelProfile) {
	sums := make(map[Dimension]float64)
	counts := make(map[Dimension]int)

	for _, result := range profile.RawResults {
		for dim, score := range result.DimensionScores {
			sums[dim] += score
			counts[dim]++
		}
	}

	for _, dim := range AllDimensions() {
		if counts[dim] > 0 {
			profile.Dimensions[dim] = sums[dim] / float64(counts[dim])
		}
	}
}
