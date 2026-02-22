package probes

import (
	"testing"

	"github.com/dpopsuev/origami/metacal"
)

func TestAmbiguitySpec(t *testing.T) {
	spec := AmbiguitySpec()
	if spec.ID != "ambiguity-v1" {
		t.Errorf("ID = %q, want ambiguity-v1", spec.ID)
	}
	if spec.Step != metacal.StepAmbiguity {
		t.Errorf("Step = %q, want %q", spec.Step, metacal.StepAmbiguity)
	}
	if len(spec.Dimensions) != 2 {
		t.Errorf("Dimensions count = %d, want 2", len(spec.Dimensions))
	}
}

func TestScoreAmbiguity_ThoroughResolution(t *testing.T) {
	response := `The requirements contain two contradictions:

1. **Timeout vs backoff**: Product wants 3 retries with exponential backoff (1+2+4 = 7s minimum), 
   but the SRE budget is 2 seconds total. This is incompatible.
   Resolution: Reduce backoff to fit within 2s budget. Use 200ms, 400ms, 800ms (total ~1.4s + request time).

2. **Retry scope**: Product wants retry on ALL requests, but SRE excludes POST/PUT/DELETE 
   because idempotency is not guaranteed.
   Resolution: Retry only GET/HEAD by default. Add an opt-in idempotency token for mutating requests.

I recommend discussing with both teams to confirm these compromises.`

	scores := ScoreAmbiguity(response)

	if scores[metacal.DimConvergenceThreshold] < 0.7 {
		t.Errorf("ConvergenceThreshold = %f, want >= 0.7 (thorough resolution)", scores[metacal.DimConvergenceThreshold])
	}
	if scores[metacal.DimFailureMode] < 0.6 {
		t.Errorf("FailureMode = %f, want >= 0.6 (resilient handling)", scores[metacal.DimFailureMode])
	}
}

func TestScoreAmbiguity_IgnoresContradictions(t *testing.T) {
	response := `Here is my implementation plan:
- Add retry with 3 attempts and exponential backoff
- Set timeout to 500ms per request
- Add circuit breaker after 3 failures`

	scores := ScoreAmbiguity(response)

	if scores[metacal.DimConvergenceThreshold] > 0.4 {
		t.Errorf("ConvergenceThreshold = %f, want <= 0.4 (ignored contradictions)", scores[metacal.DimConvergenceThreshold])
	}
	if scores[metacal.DimFailureMode] > 0.3 {
		t.Errorf("FailureMode = %f, want <= 0.3 (brittle — ignored conflicts)", scores[metacal.DimFailureMode])
	}
}

func TestScoreAmbiguity_Determinism(t *testing.T) {
	response := `There is a contradiction between the timeout budget and the backoff schedule. I recommend reducing the backoff.`

	s1 := ScoreAmbiguity(response)
	s2 := ScoreAmbiguity(response)

	for _, dim := range []metacal.Dimension{metacal.DimFailureMode, metacal.DimConvergenceThreshold} {
		if s1[dim] != s2[dim] {
			t.Errorf("Dimension %s: non-deterministic (run1=%f, run2=%f)", dim, s1[dim], s2[dim])
		}
	}
}
