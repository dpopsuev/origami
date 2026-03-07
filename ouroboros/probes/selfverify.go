package probes

import (
	"strings"
)

// SelfVerifyScore analyzes a subject response for self-verification behavior.
// Returns 0.0-1.0 indicating how much the model attempted to check its work.
//
// Detected signals (weighted):
//
//   - Includes test cases or assertions (0.25)
//   - Mentions edge cases explicitly (0.15)
//   - Shows self-correction ("wait", "actually", "let me fix") (0.20)
//   - Acknowledges limitations or known issues (0.15)
//   - Validates assumptions ("let me verify", "checking", "confirmed") (0.15)
//   - Includes error handling or defensive code (0.10)
func SelfVerifyScore(response string) float64 {
	lower := strings.ToLower(response)
	score := 0.0

	if detectTestCases(lower) {
		score += 0.25
	}
	if detectEdgeCases(lower) {
		score += 0.15
	}
	if detectSelfCorrection(lower) {
		score += 0.20
	}
	if detectLimitationAck(lower) {
		score += 0.15
	}
	if detectVerification(lower) {
		score += 0.15
	}
	if detectDefensiveCode(lower) {
		score += 0.10
	}

	return clamp(score)
}

func detectTestCases(lower string) bool {
	signals := []string{
		"func test", "def test_", "it(\"", "it('",
		"assert", "expect(", "t.error", "t.fatal",
		"#[test]", "#[cfg(test)]",
		"test case", "test:", "testing:",
		"let me test", "here's a test", "unit test",
	}
	return containsAnyOf(lower, signals)
}

func detectEdgeCases(lower string) bool {
	signals := []string{
		"edge case", "corner case", "boundary",
		"empty input", "nil", "null", "zero",
		"overflow", "underflow", "negative",
		"concurrent", "race condition",
		"what if", "what happens when",
		"special case", "empty string", "empty list",
	}
	return containsAnyOf(lower, signals)
}

func detectSelfCorrection(lower string) bool {
	signals := []string{
		"wait,", "actually,", "let me fix",
		"on second thought", "correction:",
		"i made a mistake", "that's wrong",
		"let me reconsider", "i should",
		"scratch that", "better approach",
		"oops", "hold on",
	}
	return containsAnyOf(lower, signals)
}

func detectLimitationAck(lower string) bool {
	signals := []string{
		"limitation", "caveat", "known issue",
		"doesn't handle", "won't work for",
		"assumes", "assumption",
		"note:", "warning:", "caution:",
		"not thread-safe", "not production-ready",
		"todo", "fixme", "hack",
	}
	return containsAnyOf(lower, signals)
}

func detectVerification(lower string) bool {
	signals := []string{
		"let me verify", "let me check",
		"verified", "confirmed", "checking",
		"running this", "compiling",
		"this compiles", "this should work",
		"i've tested", "testing this",
		"let me trace through", "dry run",
		"walking through", "step through",
	}
	return containsAnyOf(lower, signals)
}

func detectDefensiveCode(lower string) bool {
	signals := []string{
		"if err != nil", "try:", "except:",
		"catch (", "catch(", "?.let",
		"guard let", "unwrap_or",
		"validate", "sanitize",
		"bounds check", "nil check",
		"return err", "return error",
	}
	return containsAnyOf(lower, signals)
}

func containsAnyOf(haystack string, needles []string) bool {
	for _, n := range needles {
		if strings.Contains(haystack, n) {
			return true
		}
	}
	return false
}
