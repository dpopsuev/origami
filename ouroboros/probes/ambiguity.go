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

// BuildAmbiguityPrompt returns the prompt text using the given stimulus.
func BuildAmbiguityPrompt(s ProbeStimulus) string {
	return s.Input
}

// AmbiguityPrompt returns the prompt text using the default stimulus.
func AmbiguityPrompt() string {
	return BuildAmbiguityPrompt(DefaultStimuli()["ambiguity"])
}

// ScoreAmbiguity maps ambiguity-handling output to behavioral dimension scores.
// Prefers structured output (CONTRADICTION/RESOLUTION/TRADE_OFFS/QUESTIONS fields)
// with keyword fallback for unstructured responses.
func ScoreAmbiguity(raw string) map[ouroboros.Dimension]float64 {
	lower := strings.ToLower(raw)
	parsed := ParseStructured(raw)

	acknowledgesTimeout := parsed.HasField("CONTRADICTION") ||
		containsAny(lower,
			"contradiction", "conflict", "incompatible",
			"7s", "7 second", "exceeds 2s", "exceeds the 2",
			"budget", "backoff.*budget",
		)
	acknowledgesScope := parsed.FieldContains("CONTRADICTION", "idempoten") ||
		parsed.FieldContains("CONTRADICTION", "mutating") ||
		containsAny(lower,
			"idempoten", "mutating", "post", "put", "delete",
			"non-idempotent", "scope conflict",
		)
	proposesResolution := parsed.HasField("RESOLUTION") ||
		containsAny(lower,
			"resolve", "compromise", "recommend", "suggest",
			"propose", "solution", "adjust", "reduce",
		)
	asksClarification := parsed.ListLen("QUESTIONS") > 0 ||
		containsAny(lower,
			"clarif", "ask product", "ask sre", "confirm with",
			"need to discuss", "ambiguous",
		)
	hasTradeOffs := parsed.HasField("TRADE_OFFS")

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
	if hasTradeOffs {
		failureMode += 0.1
	}
	if !acknowledgesTimeout && !acknowledgesScope {
		failureMode = 0.1
	}

	return map[ouroboros.Dimension]float64{
		ouroboros.DimFailureMode:          clamp(failureMode),
		ouroboros.DimConvergenceThreshold: clamp(convergence),
	}
}
