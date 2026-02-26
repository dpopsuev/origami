package dispatch_test

import (
	"strings"
	"testing"

	"github.com/dpopsuev/origami/dispatch"
)

func sampleTokenSummary() *dispatch.TokenSummary {
	return &dispatch.TokenSummary{
		TotalPromptTokens:   100_000,
		TotalArtifactTokens: 5_000,
		TotalTokens:         105_000,
		TotalCostUSD:        0.375,
		TotalSteps:          12,
		TotalWallClockMs:    60_000,
		PerCase: map[string]dispatch.CaseTokenSummary{
			"C1": {PromptTokens: 60000, ArtifactTokens: 3000, TotalTokens: 63000, Steps: 7, WallClockMs: 35000},
			"C2": {PromptTokens: 40000, ArtifactTokens: 2000, TotalTokens: 42000, Steps: 5, WallClockMs: 25000},
		},
		PerStep: map[string]dispatch.StepTokenSummary{
			"STEP_A": {PromptTokens: 20000, ArtifactTokens: 1000, TotalTokens: 21000, Invocations: 2},
			"STEP_B": {PromptTokens: 30000, ArtifactTokens: 2000, TotalTokens: 32000, Invocations: 2},
			"STEP_C": {PromptTokens: 50000, ArtifactTokens: 2000, TotalTokens: 52000, Invocations: 8},
		},
	}
}

func TestBuildCostBill_Nil(t *testing.T) {
	bill := dispatch.BuildCostBill(nil)
	if bill != nil {
		t.Error("nil TokenSummary should produce nil bill")
	}
}

func TestBuildCostBill_Basic(t *testing.T) {
	bill := dispatch.BuildCostBill(sampleTokenSummary())
	if bill == nil {
		t.Fatal("expected non-nil bill")
	}
	if bill.CaseCount != 2 {
		t.Errorf("CaseCount: want 2, got %d", bill.CaseCount)
	}
	if len(bill.CaseLines) != 2 {
		t.Errorf("CaseLines: want 2, got %d", len(bill.CaseLines))
	}
	if len(bill.StepLines) != 3 {
		t.Errorf("StepLines: want 3, got %d", len(bill.StepLines))
	}
	if bill.Title != "Cost Bill" {
		t.Errorf("Title: want 'Cost Bill', got %q", bill.Title)
	}
}

func TestBuildCostBill_WithOptions(t *testing.T) {
	bill := dispatch.BuildCostBill(sampleTokenSummary(),
		dispatch.WithTitle("TokiMeter"),
		dispatch.WithSubtitle("scenario: test | adapter: llm"),
		dispatch.WithStepOrder([]string{"STEP_C", "STEP_A", "STEP_B"}),
		dispatch.WithStepNames(func(s string) string {
			names := map[string]string{"STEP_A": "Alpha", "STEP_B": "Beta", "STEP_C": "Gamma"}
			return names[s]
		}),
		dispatch.WithCaseLabels(func(id string) string { return "case-" + id }),
		dispatch.WithCaseDetails(func(id string) string { return "detail for " + id }),
	)

	if bill.Title != "TokiMeter" {
		t.Errorf("Title: got %q", bill.Title)
	}
	if bill.StepLines[0].Step != "STEP_C" {
		t.Errorf("step order: first should be STEP_C, got %s", bill.StepLines[0].Step)
	}
	if bill.StepLines[0].DisplayName != "Gamma" {
		t.Errorf("display name: want Gamma, got %s", bill.StepLines[0].DisplayName)
	}
	if bill.CaseLines[0].Label != "case-C1" {
		t.Errorf("case label: want case-C1, got %s", bill.CaseLines[0].Label)
	}
	if bill.CaseLines[0].Detail != "detail for C1" {
		t.Errorf("case detail: want 'detail for C1', got %s", bill.CaseLines[0].Detail)
	}
}

func TestBuildCostBill_StepOrderPartial(t *testing.T) {
	bill := dispatch.BuildCostBill(sampleTokenSummary(),
		dispatch.WithStepOrder([]string{"STEP_B"}),
	)
	if bill.StepLines[0].Step != "STEP_B" {
		t.Errorf("first step should be STEP_B, got %s", bill.StepLines[0].Step)
	}
	// Remaining steps appear after in alphabetical order
	if len(bill.StepLines) != 3 {
		t.Fatalf("want 3 steps, got %d", len(bill.StepLines))
	}
	if bill.StepLines[1].Step != "STEP_A" {
		t.Errorf("second step should be STEP_A, got %s", bill.StepLines[1].Step)
	}
}

func TestBuildCostBill_CustomCostConfig(t *testing.T) {
	bill := dispatch.BuildCostBill(sampleTokenSummary(),
		dispatch.WithCostConfig(dispatch.CostConfig{InputPricePerMToken: 1.0, OutputPricePerMToken: 2.0}),
	)
	// C1: 60000 in * 1/M + 3000 out * 2/M = 0.06 + 0.006 = 0.066
	for _, cl := range bill.CaseLines {
		if cl.CaseID == "C1" {
			expected := float64(60000)/1e6*1.0 + float64(3000)/1e6*2.0
			if cl.CostUSD != expected {
				t.Errorf("C1 cost: want %f, got %f", expected, cl.CostUSD)
			}
		}
	}
}

func TestFormatCostBill_Nil(t *testing.T) {
	if dispatch.FormatCostBill(nil) != "" {
		t.Error("nil bill should produce empty string")
	}
}

func TestFormatCostBill_Markdown(t *testing.T) {
	bill := dispatch.BuildCostBill(sampleTokenSummary(),
		dispatch.WithTitle("TokiMeter"),
		dispatch.WithSubtitle("test scenario"),
	)
	md := dispatch.FormatCostBill(bill)

	checks := []string{
		"# TokiMeter",
		"## Summary",
		"## Per-case costs",
		"## Per-step costs",
		"| Case |",
		"| Step |",
		"| **TOTAL**",
		"test scenario",
		"105.0K",
		"C1",
		"C2",
	}
	for _, check := range checks {
		if !strings.Contains(md, check) {
			t.Errorf("markdown missing: %q", check)
		}
	}
}

func TestFormatCostBill_NoCases(t *testing.T) {
	ts := &dispatch.TokenSummary{
		TotalPromptTokens:   1000,
		TotalArtifactTokens: 500,
		TotalTokens:         1500,
		TotalSteps:          1,
	}
	bill := dispatch.BuildCostBill(ts)
	md := dispatch.FormatCostBill(bill)
	if strings.Contains(md, "Per-case costs") {
		t.Error("should not show per-case section when no cases")
	}
}
