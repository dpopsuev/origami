package mcpconfig

import "testing"

func TestStepToNode(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"F0_RECALL", "recall"},
		{"F1_TRIAGE", "triage"},
		{"F2_RESOLVE", "resolve"},
		{"F3_INVESTIGATE", "investigate"},
		{"F4_CORRELATE", "correlate"},
		{"F5_REVIEW", "review"},
		{"F6_REPORT", "report"},
		{"STEP_A", "step_a"},
		{"plain", "plain"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := stepToNode(tt.input)
			if got != tt.want {
				t.Errorf("stepToNode(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
