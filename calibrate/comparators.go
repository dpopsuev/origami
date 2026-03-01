package calibrate

import (
	"fmt"
	"math"
	"path/filepath"
	"strings"
)

// KeywordMatch counts how many keywords appear (case-insensitive) in text.
func KeywordMatch(text string, keywords []string) int {
	lower := strings.ToLower(text)
	count := 0
	for _, kw := range keywords {
		if strings.Contains(lower, strings.ToLower(kw)) {
			count++
		}
	}
	return count
}

// EvidenceOverlap computes lenient set overlap between actual and expected
// evidence references. Format: "repo:file_path:identifier". Matching uses:
//  1. Exact substring match (either direction)
//  2. filepath.Base match
//  3. Same repo + same file path (ignoring identifier suffix)
func EvidenceOverlap(actual, expected []string) (found, total int) {
	total = len(expected)
	if total == 0 {
		return 0, 0
	}
	for _, exp := range expected {
		expNorm := filepath.Base(exp)
		matched := false
		for _, act := range actual {
			if strings.Contains(act, expNorm) || strings.Contains(exp, act) || act == exp {
				matched = true
				break
			}
		}
		if !matched {
			expParts := strings.SplitN(exp, ":", 3)
			if len(expParts) >= 2 {
				for _, act := range actual {
					if strings.HasPrefix(act, expParts[0]+":") && strings.Contains(act, expParts[1]) {
						matched = true
						break
					}
				}
			}
		}
		if matched {
			found++
		}
	}
	return found, total
}

// PathsEqual compares two ordered string slices element-wise.
func PathsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// SmokingGunWords tokenizes a phrase into significant lowercase words (len > 3).
func SmokingGunWords(phrase string) []string {
	var words []string
	for _, w := range strings.Fields(strings.ToLower(phrase)) {
		if len(w) > 3 {
			words = append(words, w)
		}
	}
	return words
}

// PearsonCorrelation computes Pearson's r between two equal-length float slices.
// Returns 0 for fewer than 2 data points. When y has zero variance and all
// values are 1.0, returns 1.0 (perfect-answer scenario, e.g. stub mode).
func PearsonCorrelation(x, y []float64) float64 {
	if len(x) != len(y) || len(x) < 2 {
		return 0
	}
	mx, my := Mean(x), Mean(y)
	var num, dx2, dy2 float64
	for i := range x {
		dx := x[i] - mx
		dy := y[i] - my
		num += dx * dy
		dx2 += dx * dx
		dy2 += dy * dy
	}
	denom := math.Sqrt(dx2 * dy2)
	if denom == 0 {
		allOne := true
		for _, v := range y {
			if v != 1.0 {
				allOne = false
				break
			}
		}
		if allOne && len(y) > 0 {
			return 1.0
		}
		return 0
	}
	return num / denom
}

// valuesMatch compares two values for batch_field_match scoring.
// Handles []string (ordered slice comparison) and string (case-insensitive).
func valuesMatch(a, b any) bool {
	aSlice := toStringSlice(a)
	bSlice := toStringSlice(b)
	if aSlice != nil && bSlice != nil {
		return PathsEqual(aSlice, bSlice)
	}
	return strings.EqualFold(fmt.Sprintf("%v", a), fmt.Sprintf("%v", b))
}

// toStringSlice attempts to coerce an any value to []string.
// Handles []string, []any (with string elements), and nil.
func toStringSlice(v any) []string {
	switch sv := v.(type) {
	case []string:
		return sv
	case []any:
		out := make([]string, len(sv))
		for i, item := range sv {
			out[i] = fmt.Sprintf("%v", item)
		}
		return out
	default:
		return nil
	}
}
