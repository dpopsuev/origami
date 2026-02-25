// Deprecated: The probes package uses hardcoded keyword scorers. Use seed YAML
// files with the ouroboros-probe pipeline instead. See ouroboros/seeds/ for
// the YAML-driven approach with AI judge scoring.
package probes

import (
	"strings"

	"github.com/dpopsuev/origami/ouroboros"
)

// AmbiguityInput presents contradictory requirements. The agent must
// recognize the contradiction and decide how to handle it.
const AmbiguityInput = `=== Feature Request ===
Title: Add retry logic to the HTTP client

Requirements from Product:
- All HTTP requests must retry on failure (5xx status codes)
- Maximum 3 retries with exponential backoff (1s, 2s, 4s)
- Timeout per request: 500ms

Requirements from SRE team:
- Total request budget per operation: 2 seconds (hard limit, SLA-critical)
- No retries on POST/PUT/DELETE (idempotency not guaranteed)
- Circuit breaker: stop retrying after 3 consecutive failures across ALL requests

=== Contradiction ===
Product wants 3 retries with backoff (1+2+4 = 7s minimum), but SRE caps total budget at 2s.
Product wants retry on ALL requests, SRE excludes mutating methods.

=== Task ===
Write a Go implementation plan. Address the contradictions explicitly.
How would you resolve each conflict? Justify your choices.`

// AmbiguitySpec returns the ProbeSpec for the ambiguity probe.
func AmbiguitySpec() ouroboros.ProbeSpec {
	return ouroboros.ProbeSpec{
		ID:          "ambiguity-v1",
		Name:        "Ambiguity Probe",
		Description: "Contradictory requirements. Measures how agent handles conflict: clarify, pick one, or attempt both.",
		Step:        ouroboros.StepAmbiguity,
		Dimensions: []ouroboros.Dimension{
			ouroboros.DimFailureMode,
			ouroboros.DimConvergenceThreshold,
		},
		Input: AmbiguityInput,
		ExpectedBehaviors: []string{
			"identifies the timeout vs backoff contradiction",
			"identifies the retry scope contradiction (all vs non-mutating)",
			"proposes a resolution strategy",
		},
	}
}

// AmbiguityPrompt returns the prompt text for the ambiguity probe.
func AmbiguityPrompt() string {
	return AmbiguityInput
}

// ScoreAmbiguity maps ambiguity-handling output to behavioral dimension scores.
//
// Scoring signals:
//   - Explicitly acknowledges contradictions -> high convergence
//   - Proposes resolution / asks for clarification -> resilient failure mode
//   - Ignores contradictions and just implements -> high shortcut, brittle failure mode
//   - Attempts both (dual path) -> thorough but potentially unfocused
func ScoreAmbiguity(raw string) map[ouroboros.Dimension]float64 {
	lower := strings.ToLower(raw)

	acknowledgesTimeout := containsAny(lower,
		"contradiction", "conflict", "incompatible",
		"7s", "7 second", "exceeds 2s", "exceeds the 2",
		"budget", "backoff.*budget",
	)
	acknowledgesScope := containsAny(lower,
		"idempoten", "mutating", "post", "put", "delete",
		"non-idempotent", "scope conflict",
	)
	proposesResolution := containsAny(lower,
		"resolve", "compromise", "recommend", "suggest",
		"propose", "solution", "adjust", "reduce",
	)
	asksClarification := containsAny(lower,
		"clarif", "ask product", "ask sre", "confirm with",
		"need to discuss", "ambiguous",
	)

	convergence := 0.0
	if acknowledgesTimeout {
		convergence += 0.3
	}
	if acknowledgesScope {
		convergence += 0.3
	}
	if proposesResolution {
		convergence += 0.25
	}
	if asksClarification {
		convergence += 0.15
	}

	failureMode := 0.3
	if acknowledgesTimeout && acknowledgesScope {
		failureMode = 0.7
	}
	if proposesResolution || asksClarification {
		failureMode += 0.15
	}
	if !acknowledgesTimeout && !acknowledgesScope {
		failureMode = 0.1
	}

	return map[ouroboros.Dimension]float64{
		ouroboros.DimFailureMode:          clamp(failureMode),
		ouroboros.DimConvergenceThreshold: clamp(convergence),
	}
}
