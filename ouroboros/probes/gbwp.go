package probes

import (
	"sort"
	"strconv"
	"strings"

	"github.com/dpopsuev/origami/ouroboros"
)

// GBWPInput is a fixed dialectic scenario for measuring accuracy vs rounds.
// The scenario presents a claim with a known correct verdict so the scorer
// can measure how accurately the model resolves it.
const GBWPInput = `=== Dialectic Accuracy Scenario ===

CLAIM: "The test failure in TestNetworkRetry is caused by a race condition
in the retry logic where the timeout channel and the success channel are
read without synchronization."

EVIDENCE:
1. Stack trace shows goroutine overlap at retry.go:47 and retry.go:82
2. The failure is intermittent — passes 95% of the time
3. Adding -race flag reproduces the failure consistently
4. The retry counter is incremented without mutex protection

KNOWN VERDICT: The claim is CORRECT. Confidence should be HIGH (>= 0.85).
The race condition is the root cause, supported by all four evidence items.

YOUR TASK: Analyze the claim against the evidence. Provide:
1. Your verdict: CORRECT or INCORRECT
2. Your confidence: a decimal between 0.0 and 1.0
3. Brief justification (1-2 sentences)

Format your response as:
VERDICT: <CORRECT|INCORRECT>
CONFIDENCE: <0.0-1.0>
JUSTIFICATION: <text>`

// BuildGBWPPrompt returns the prompt for a GBWP probe run.
func BuildGBWPPrompt(s ProbeStimulus) string {
	return s.Input
}

// ScoreGBWP scores a single GBWP probe response against the known verdict.
func ScoreGBWP(raw string) map[ouroboros.Dimension]float64 {
	lower := strings.ToLower(raw)

	verdictCorrect := strings.Contains(lower, "verdict: correct") ||
		strings.Contains(lower, "verdict:correct")

	confidence := parseConfidence(lower)

	accuracy := 0.0
	if verdictCorrect {
		accuracy = 0.5
		if confidence >= 0.85 {
			accuracy = 1.0
		} else if confidence >= 0.7 {
			accuracy = 0.75
		}
	}

	return map[ouroboros.Dimension]float64{
		ouroboros.DimGBWP: accuracy,
	}
}

func parseConfidence(lower string) float64 {
	for _, prefix := range []string{"confidence: ", "confidence:"} {
		idx := strings.Index(lower, prefix)
		if idx < 0 {
			continue
		}
		rest := strings.TrimSpace(lower[idx+len(prefix):])
		end := strings.IndexAny(rest, " \n\r\t")
		if end > 0 {
			rest = rest[:end]
		}
		val, err := strconv.ParseFloat(rest, 64)
		if err == nil {
			return val
		}
	}
	return 0
}

// GBWPPoint represents a single measurement in the GBWP curve:
// accuracy achieved at a given number of dialectic rounds.
type GBWPPoint struct {
	Rounds   int
	Accuracy float64
}

// ComputeGBWP computes the Gain-Bandwidth Product score from accuracy
// measurements at different MaxTurns settings. Uses trapezoidal AUC
// normalized to [0, 1] where 1.0 means perfect accuracy at all round counts.
func ComputeGBWP(points []GBWPPoint) float64 {
	if len(points) < 2 {
		if len(points) == 1 {
			return points[0].Accuracy
		}
		return 0
	}

	sort.Slice(points, func(i, j int) bool {
		return points[i].Rounds < points[j].Rounds
	})

	totalWidth := float64(points[len(points)-1].Rounds - points[0].Rounds)
	if totalWidth == 0 {
		return points[0].Accuracy
	}

	auc := 0.0
	for i := 1; i < len(points); i++ {
		dx := float64(points[i].Rounds - points[i-1].Rounds)
		avgY := (points[i-1].Accuracy + points[i].Accuracy) / 2
		auc += dx * avgY
	}

	return clamp(auc / totalWidth)
}

// clamp is in refactor.go (shared across probes)
