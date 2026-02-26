package probes

import (
	"strings"

	"github.com/dpopsuev/origami/ouroboros"
)

// DebugInput is a synthetic log extract with one red herring (high memory)
// and one subtle root cause (connection pool exhaustion from leaked goroutine).
const DebugInput = `=== Application logs (last 30 minutes) ===
14:01:02 WARN  memory usage high: 4.2GB / 8GB (52%)
14:01:05 INFO  request /api/users completed in 230ms
14:01:08 ERROR connection pool exhausted: 0/50 available
14:01:08 ERROR failed to acquire connection: context deadline exceeded (30s)
14:01:09 WARN  memory usage high: 4.3GB / 8GB (53%)
14:01:10 INFO  goroutine count: 12847 (baseline: ~200)
14:01:12 ERROR connection pool exhausted: 0/50 available
14:01:15 WARN  GC pause 45ms (target: <10ms)
14:01:18 ERROR request /api/orders failed: context deadline exceeded
14:01:20 INFO  goroutine count: 13102
14:01:22 ERROR connection pool exhausted: 0/50 available
14:01:25 WARN  memory usage high: 4.5GB / 8GB (56%)

=== Deployment history ===
13:30:00 Deployed v2.14.0 (added async notification worker)
13:45:00 Feature flag "async-notifications" enabled

=== Question ===
What is the root cause of the errors? Provide your analysis as structured text:
1. Root cause (one sentence)
2. Evidence (list the specific log lines)
3. Red herrings you considered and rejected (with reasoning)
4. Recommended fix`

// BuildDebugPrompt returns the prompt text using the given stimulus.
func BuildDebugPrompt(s ProbeStimulus) string {
	return s.Input
}

// DebugPrompt returns the prompt text using the default stimulus.
func DebugPrompt() string {
	return BuildDebugPrompt(DefaultStimuli()["debug"])
}

// ScoreDebug maps debugging output to behavioral dimension scores.
//
// Scoring signals:
//   - Root cause identified (goroutine leak / connection pool) -> convergence
//   - Red herring rejected (memory) -> not shortcut-prone
//   - Deployment correlation (v2.14.0) -> evidence depth
//   - Brevity vs detail -> speed indicator
func ScoreDebug(raw string) map[ouroboros.Dimension]float64 {
	lower := strings.ToLower(raw)
	lines := strings.Split(raw, "\n")

	rootCauseFound := containsAny(lower,
		"goroutine leak", "goroutine count", "connection pool",
		"leaked goroutine", "goroutine explosion",
	)
	deploymentLinked := containsAny(lower,
		"v2.14", "async notification", "async-notification",
		"deployment", "feature flag",
	)
	redHerringRejected := containsAny(lower,
		"red herring", "not the root cause", "symptom",
		"memory is not", "52%", "not critical",
	)
	hasStructuredOutput := countSections(raw) >= 3

	convergence := 0.0
	if rootCauseFound {
		convergence += 0.5
	}
	if deploymentLinked {
		convergence += 0.3
	}
	if hasStructuredOutput {
		convergence += 0.2
	}

	shortcut := 0.5
	if !redHerringRejected && rootCauseFound {
		shortcut = 0.8
	}
	if redHerringRejected {
		shortcut = 0.2
	}

	speed := 0.5
	nonEmptyLines := countNonEmpty(lines)
	if nonEmptyLines < 10 {
		speed = 0.9
	} else if nonEmptyLines < 20 {
		speed = 0.6
	} else {
		speed = 0.3
	}

	return map[ouroboros.Dimension]float64{
		ouroboros.DimSpeed:                clamp(speed),
		ouroboros.DimShortcutAffinity:     clamp(shortcut),
		ouroboros.DimConvergenceThreshold: clamp(convergence),
	}
}

func containsAny(haystack string, needles ...string) bool {
	for _, n := range needles {
		if strings.Contains(haystack, n) {
			return true
		}
	}
	return false
}

func countSections(text string) int {
	count := 0
	for _, marker := range []string{"1.", "2.", "3.", "4.", "root cause", "evidence", "red herring", "fix", "recommend"} {
		if strings.Contains(strings.ToLower(text), marker) {
			count++
		}
	}
	return count
}

func countNonEmpty(lines []string) int {
	count := 0
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			count++
		}
	}
	return count
}
