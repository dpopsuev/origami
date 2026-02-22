package metacalmcp

import (
	"strings"
	"testing"
)

func TestProbeRegistry_AllFiveProbes(t *testing.T) {
	r := NewProbeRegistry()
	expected := []string{"refactor-v1", "debug-v1", "summarize-v1", "ambiguity-v1", "persistence-v1"}

	for _, id := range expected {
		h, err := r.Get(id)
		if err != nil {
			t.Errorf("Get(%q): %v", id, err)
			continue
		}
		if h.ID != id {
			t.Errorf("handler ID = %q, want %q", h.ID, id)
		}
		if h.Prompt == nil || h.Score == nil {
			t.Errorf("handler %q has nil Prompt or Score", id)
		}
		prompt := h.Prompt()
		if prompt == "" {
			t.Errorf("handler %q returned empty prompt", id)
		}
	}
}

func TestProbeRegistry_UnknownProbe_ReturnsError(t *testing.T) {
	r := NewProbeRegistry()
	_, err := r.Get("nonexistent-v99")
	if err == nil {
		t.Fatal("expected error for unknown probe ID")
	}
	if !strings.Contains(err.Error(), "unknown probe_id") {
		t.Errorf("error = %q, want 'unknown probe_id'", err.Error())
	}
}

func TestProbeRegistry_RefactorNeedsCodeBlock(t *testing.T) {
	r := NewProbeRegistry()
	h, _ := r.Get("refactor-v1")
	if !h.NeedsCodeBlock {
		t.Error("refactor probe should have NeedsCodeBlock=true")
	}
}

func TestProbeRegistry_NonRefactorProbes_DontNeedCodeBlock(t *testing.T) {
	r := NewProbeRegistry()
	for _, id := range []string{"debug-v1", "summarize-v1", "ambiguity-v1", "persistence-v1"} {
		h, _ := r.Get(id)
		if h.NeedsCodeBlock {
			t.Errorf("probe %q should have NeedsCodeBlock=false", id)
		}
	}
}

func TestProbeRegistry_ScorersReturnDimensions(t *testing.T) {
	r := NewProbeRegistry()

	cases := []struct {
		probeID string
		input   string
	}{
		{"refactor-v1", "func improved(nums []int) int { return 0 }"},
		{"debug-v1", "1. Root cause: goroutine leak\n2. Evidence: goroutine count 12847\n3. Red herring: memory is not critical at 52%\n4. Fix: close goroutines"},
		{"summarize-v1", "1. FetchUserMetrics feature added\n2. FetchAll refactored\n3. Error handling bugfix\n4. Cache RLock performance\nRisk: low"},
		{"ambiguity-v1", "The timeout contradiction means 7s exceeds the 2s budget. Idempotency conflict for POST. I propose reducing retries."},
		{"persistence-v1", "func ParseConfig(input string) (map[string]interface{}, error) {\n\tcurrentSection := \"\"\n\tbase64.StdEncoding.DecodeString(val)\n\tos.Getenv(key)\n\treturn nil, fmt.Errorf(\"parse error\")\n}"},
	}

	for _, tc := range cases {
		t.Run(tc.probeID, func(t *testing.T) {
			h, _ := r.Get(tc.probeID)
			scores := h.Score(tc.input)
			if len(scores) == 0 {
				t.Errorf("scorer for %q returned no dimension scores", tc.probeID)
			}
			for dim, score := range scores {
				if score < 0 || score > 1 {
					t.Errorf("dimension %q score %.2f out of [0,1] range", dim, score)
				}
			}
		})
	}
}

func TestProbeRegistry_IDs(t *testing.T) {
	r := NewProbeRegistry()
	ids := r.IDs()
	if len(ids) != 5 {
		t.Errorf("IDs() returned %d probes, want 5", len(ids))
	}
}
