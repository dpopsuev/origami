// Package mathutil provides basic statistical functions used across Origami
// packages. Extracted from calibrate/ so that observability/ and other
// packages can use stats without importing calibration concerns.
package mathutil

import "math"

// Mean returns the arithmetic mean of vals. Returns 0 for empty input.
func Mean(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}

// Stddev returns the sample standard deviation (Bessel-corrected, N-1).
// Returns 0 when fewer than 2 values are provided.
func Stddev(vals []float64) float64 {
	if len(vals) < 2 {
		return 0
	}
	m := Mean(vals)
	sum := 0.0
	for _, v := range vals {
		sum += (v - m) * (v - m)
	}
	return math.Sqrt(sum / float64(len(vals)-1))
}

// SafeDiv divides two integers. Returns 1.0 when denom is 0
// (0/0 = perfect: nothing to measure).
func SafeDiv(num, denom int) float64 {
	if denom == 0 {
		return 1.0
	}
	return float64(num) / float64(denom)
}

// SafeDivFloat divides two float64 values. Returns 1.0 when denom is 0.
func SafeDivFloat(num, denom float64) float64 {
	if denom == 0 {
		return 1.0
	}
	return num / denom
}
