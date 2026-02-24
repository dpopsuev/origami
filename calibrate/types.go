// Package calibrate provides generic calibration primitives for measuring
// domain accuracy across scenario-vs-ground-truth runs. Consumers (Asterisk,
// Achilles, etc.) supply domain-specific scoring; this package provides the
// shared types, aggregation, and report formatting.
package calibrate

import "github.com/dpopsuev/origami/dispatch"

// Metric is a single calibration metric with value, threshold, and pass/fail.
type Metric struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Value     float64 `json:"value"`
	Threshold float64 `json:"threshold"`
	Pass      bool    `json:"pass"`
	Detail    string  `json:"detail"`
	DryCapped bool    `json:"dry_capped,omitempty"`
}

// MetricSet holds all computed metrics for a calibration run, organized
// into domain-agnostic groups. Consumers decide which metrics go in
// which group.
type MetricSet struct {
	Structured []Metric `json:"structured"`
	Workspace  []Metric `json:"workspace"`
	Evidence   []Metric `json:"evidence"`
	Semantic   []Metric `json:"semantic"`
	Pipeline   []Metric `json:"pipeline"`
	Aggregate  []Metric `json:"aggregate"`
}

// AllMetrics returns all metrics as a flat list.
func (ms *MetricSet) AllMetrics() []Metric {
	var all []Metric
	all = append(all, ms.Structured...)
	all = append(all, ms.Workspace...)
	all = append(all, ms.Evidence...)
	all = append(all, ms.Semantic...)
	all = append(all, ms.Pipeline...)
	all = append(all, ms.Aggregate...)
	return all
}

// PassCount returns (passed, total), excluding dry-capped metrics from both counts.
func (ms *MetricSet) PassCount() (int, int) {
	all := ms.AllMetrics()
	passed, total := 0, 0
	for _, m := range all {
		if m.DryCapped {
			continue
		}
		total++
		if m.Pass {
			passed++
		}
	}
	return passed, total
}

// CalibrationReport is the generic output of a calibration run.
// Consumers embed this struct and add domain-specific fields
// (e.g. CaseResults, DatasetHealth).
type CalibrationReport struct {
	Scenario   string                  `json:"scenario"`
	Adapter    string                  `json:"adapter"`
	Runs       int                     `json:"runs"`
	Metrics    MetricSet               `json:"metrics"`
	RunMetrics []MetricSet             `json:"run_metrics,omitempty"`
	Tokens     *dispatch.TokenSummary  `json:"tokens,omitempty"`
}
