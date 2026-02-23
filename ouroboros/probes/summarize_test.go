package probes

import (
	"testing"

	"github.com/dpopsuev/origami/ouroboros"
)

func TestSummarizeSpec(t *testing.T) {
	spec := SummarizeSpec()
	if spec.ID != "summarize-v1" {
		t.Errorf("ID = %q, want summarize-v1", spec.ID)
	}
	if spec.Step != ouroboros.StepSummarize {
		t.Errorf("Step = %q, want %q", spec.Step, ouroboros.StepSummarize)
	}
	if len(spec.Dimensions) != 2 {
		t.Errorf("Dimensions count = %d, want 2", len(spec.Dimensions))
	}
}

func TestScoreSummarize_CompleteSummary(t *testing.T) {
	response := `1. **New FetchUserMetrics method** - Added a new function to retrieve aggregated user metrics.
   Category: feature | Risk: low

2. **FetchAll signature refactor** - Changed return type from interface{} to ([]DashboardItem, error) for type safety.
   Category: refactor | Risk: medium

3. **Error handling in GetDashboard** - Added proper error handling instead of ignoring the error from FetchAll.
   Category: bugfix | Risk: low

4. **Cache read lock optimization** - Changed from full mutex Lock to RLock for read operations.
   Category: performance | Risk: low`

	scores := ScoreSummarize(response)

	if scores[ouroboros.DimEvidenceDepth] < 0.7 {
		t.Errorf("EvidenceDepth = %f, want >= 0.7 (found all changes + categories + risk)", scores[ouroboros.DimEvidenceDepth])
	}
	if scores[ouroboros.DimFailureMode] < 0.4 {
		t.Errorf("FailureMode = %f, want >= 0.4 (concise, well-structured)", scores[ouroboros.DimFailureMode])
	}
}

func TestScoreSummarize_PartialSummary(t *testing.T) {
	response := `This PR adds user metrics and fixes some error handling.`

	scores := ScoreSummarize(response)

	if scores[ouroboros.DimEvidenceDepth] > 0.5 {
		t.Errorf("EvidenceDepth = %f, want <= 0.5 (missed most changes)", scores[ouroboros.DimEvidenceDepth])
	}
}

func TestScoreSummarize_Determinism(t *testing.T) {
	response := `The PR contains four changes: a new feature, a refactoring, a bugfix for error handling, and a performance improvement using RLock in the cache.`

	s1 := ScoreSummarize(response)
	s2 := ScoreSummarize(response)

	for _, dim := range []ouroboros.Dimension{ouroboros.DimEvidenceDepth, ouroboros.DimFailureMode} {
		if s1[dim] != s2[dim] {
			t.Errorf("Dimension %s: non-deterministic (run1=%f, run2=%f)", dim, s1[dim], s2[dim])
		}
	}
}
