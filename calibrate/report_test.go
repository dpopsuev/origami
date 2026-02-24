package calibrate_test

import (
	"strings"
	"testing"

	"github.com/dpopsuev/origami/calibrate"
	"github.com/dpopsuev/origami/dispatch"
)

func TestFormatReport_ContainsHeader(t *testing.T) {
	report := &calibrate.CalibrationReport{
		Scenario: "test-scenario",
		Adapter:  "stub",
		Runs:     3,
		Metrics:  sampleMetricSet(),
	}
	cfg := calibrate.FormatConfig{
		Title: "Test Report",
		Sections: []calibrate.MetricSection{
			{Title: "Section A", Metrics: report.Metrics.Structured},
		},
	}
	out := calibrate.FormatReport(report, cfg)

	for _, want := range []string{
		"=== Test Report ===",
		"Scenario: test-scenario",
		"Adapter:  stub",
		"Runs:     3",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("header: missing %q", want)
		}
	}
}

func TestFormatReport_ContainsMetricTable(t *testing.T) {
	report := &calibrate.CalibrationReport{
		Scenario: "s",
		Adapter:  "a",
		Runs:     1,
		Metrics: calibrate.MetricSet{
			Structured: []calibrate.Metric{
				{ID: "X1", Name: "test_metric", Value: 0.90, Threshold: 0.80, Pass: true, Detail: "9/10"},
			},
		},
	}
	cfg := calibrate.FormatConfig{
		Sections: []calibrate.MetricSection{
			{Title: "Test Section", Metrics: report.Metrics.Structured},
		},
	}
	out := calibrate.FormatReport(report, cfg)

	for _, want := range []string{
		"--- Test Section ---",
		"X1",
		"test_metric",
		"0.90",
		"9/10",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("table: missing %q", want)
		}
	}
}

func TestFormatReport_ResultLine(t *testing.T) {
	report := &calibrate.CalibrationReport{
		Scenario: "s",
		Adapter:  "a",
		Runs:     1,
		Metrics: calibrate.MetricSet{
			Structured: []calibrate.Metric{
				{ID: "A", Value: 1.0, Threshold: 0.5, Pass: true},
				{ID: "B", Value: 0.1, Threshold: 0.5, Pass: false},
			},
		},
	}
	out := calibrate.FormatReport(report, calibrate.FormatConfig{})
	if !strings.Contains(out, "RESULT: FAIL (1/2 metrics within threshold)") {
		t.Errorf("result line: want FAIL 1/2, got:\n%s", out)
	}
}

func TestFormatReport_PassResult(t *testing.T) {
	report := &calibrate.CalibrationReport{
		Scenario: "s",
		Adapter:  "a",
		Runs:     1,
		Metrics: calibrate.MetricSet{
			Structured: []calibrate.Metric{
				{ID: "A", Value: 1.0, Threshold: 0.5, Pass: true},
			},
		},
	}
	out := calibrate.FormatReport(report, calibrate.FormatConfig{})
	if !strings.Contains(out, "RESULT: PASS") {
		t.Errorf("result line: want PASS, got:\n%s", out)
	}
}

func TestFormatReport_TokenSummary(t *testing.T) {
	report := &calibrate.CalibrationReport{
		Scenario: "s",
		Adapter:  "a",
		Runs:     1,
		Metrics:  calibrate.MetricSet{},
		Tokens: &dispatch.TokenSummary{
			TotalTokens:  5000,
			TotalCostUSD: 0.05,
		},
	}
	out := calibrate.FormatReport(report, calibrate.FormatConfig{})
	if !strings.Contains(out, "Token") {
		t.Errorf("token summary: want Token section, got:\n%s", out)
	}
}

func TestFormatReport_CustomNameFunc(t *testing.T) {
	report := &calibrate.CalibrationReport{
		Scenario: "s",
		Adapter:  "a",
		Runs:     1,
		Metrics: calibrate.MetricSet{
			Structured: []calibrate.Metric{
				{ID: "X1", Name: "fallback_name", Value: 0.5, Threshold: 0.5, Pass: true},
			},
		},
	}
	cfg := calibrate.FormatConfig{
		Sections: []calibrate.MetricSection{
			{Title: "Sec", Metrics: report.Metrics.Structured},
		},
		MetricNameFunc: func(id string) string {
			if id == "X1" {
				return "Custom Display Name"
			}
			return ""
		},
	}
	out := calibrate.FormatReport(report, cfg)
	if !strings.Contains(out, "Custom Display Name") {
		t.Errorf("custom name: missing 'Custom Display Name' in:\n%s", out)
	}
}

func TestFormatReport_DryCappedMark(t *testing.T) {
	report := &calibrate.CalibrationReport{
		Scenario: "s",
		Adapter:  "a",
		Runs:     1,
		Metrics: calibrate.MetricSet{
			Structured: []calibrate.Metric{
				{ID: "D1", Name: "dry", Value: 0.0, Threshold: 0.5, Pass: false, DryCapped: true},
			},
		},
	}
	cfg := calibrate.FormatConfig{
		Sections: []calibrate.MetricSection{
			{Title: "Sec", Metrics: report.Metrics.Structured},
		},
	}
	out := calibrate.FormatReport(report, cfg)
	if !strings.Contains(out, "~") {
		t.Errorf("dry capped: missing '~' marker in:\n%s", out)
	}
}
