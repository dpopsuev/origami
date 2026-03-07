package calibrate

import (
	"context"
	"fmt"
	"log/slog"

	framework "github.com/dpopsuev/origami"
)

// ScenarioLoader prepares domain-specific scenarios for BatchWalk.
// Each call to Load returns a fresh set of cases (e.g., with a new
// in-memory store), enabling independent multi-run calibration.
type ScenarioLoader interface {
	Load(ctx context.Context) ([]framework.BatchCase, error)
}

// CaseCollector extracts domain-specific results from BatchWalk output
// and produces metric values for ScoreCard evaluation.
// Implementations typically store domain state internally (e.g., per-case
// results) that callers retrieve after Run() for post-processing.
type CaseCollector interface {
	Collect(ctx context.Context, results []framework.BatchWalkResult) (
		values map[string]float64, details map[string]string, err error)
}

// ReportRenderer produces human-readable output from a calibration report.
type ReportRenderer interface {
	Render(report *CalibrationReport) (string, error)
}

// HarnessConfig configures a generic calibration run.
type HarnessConfig struct {
	Loader    ScenarioLoader
	Collector CaseCollector
	Renderer  ReportRenderer

	CircuitDef *framework.CircuitDef
	ScoreCard  *ScoreCard
	Shared     framework.GraphRegistries

	Scenario    string
	Transformer string
	Runs        int
	Parallel    int

	OnCaseComplete func(index int, result framework.BatchWalkResult)
}

// Run orchestrates a generic calibration: load → walk → collect → score → aggregate.
// It returns the generic CalibrationReport. Domain-specific state (e.g., per-case
// results) is stored inside the CaseCollector and can be retrieved by the caller.
func Run(ctx context.Context, cfg HarnessConfig) (*CalibrationReport, error) {
	if cfg.Loader == nil {
		return nil, fmt.Errorf("calibrate.Run: Loader is required")
	}
	if cfg.Collector == nil {
		return nil, fmt.Errorf("calibrate.Run: Collector is required")
	}
	if cfg.CircuitDef == nil {
		return nil, fmt.Errorf("calibrate.Run: CircuitDef is required")
	}
	if cfg.ScoreCard == nil {
		return nil, fmt.Errorf("calibrate.Run: ScoreCard is required")
	}
	if cfg.Runs < 1 {
		cfg.Runs = 1
	}

	logger := slog.Default().With("component", "calibrate")
	var allRunMetrics []MetricSet

	for run := 0; run < cfg.Runs; run++ {
		logger.Info("starting run", "run", run+1, "total", cfg.Runs)

		cases, err := cfg.Loader.Load(ctx)
		if err != nil {
			return nil, fmt.Errorf("run %d: load: %w", run+1, err)
		}

		batchResults := framework.BatchWalk(ctx, framework.BatchWalkConfig{
			Def:            cfg.CircuitDef,
			Shared:         cfg.Shared,
			Cases:          cases,
			Parallel:       cfg.Parallel,
			OnCaseComplete: cfg.OnCaseComplete,
		})

		values, details, err := cfg.Collector.Collect(ctx, batchResults)
		if err != nil {
			return nil, fmt.Errorf("run %d: collect: %w", run+1, err)
		}

		ms := cfg.ScoreCard.Evaluate(values, details)
		if cfg.ScoreCard.Aggregate != nil {
			agg, err := cfg.ScoreCard.ComputeAggregate(ms)
			if err == nil {
				ms.Metrics = append(ms.Metrics, agg)
			}
		}

		allRunMetrics = append(allRunMetrics, ms)
	}

	report := &CalibrationReport{
		Scenario:    cfg.Scenario,
		Transformer: cfg.Transformer,
		Runs:        cfg.Runs,
	}

	eval := func(m Metric) bool {
		if def := cfg.ScoreCard.FindDef(m.ID); def != nil {
			return def.Evaluate(m.Value)
		}
		return m.Value >= m.Threshold
	}

	if len(allRunMetrics) == 1 {
		report.Metrics = allRunMetrics[0]
	} else {
		report.RunMetrics = allRunMetrics
		report.Metrics = AggregateRunMetrics(allRunMetrics, eval)
	}

	return report, nil
}
