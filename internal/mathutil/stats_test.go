package mathutil_test

import (
	"math"
	"testing"

	"github.com/dpopsuev/origami/internal/mathutil"
)

func TestMean(t *testing.T) {
	tests := []struct {
		vals []float64
		want float64
	}{
		{nil, 0},
		{[]float64{}, 0},
		{[]float64{5}, 5},
		{[]float64{2, 4}, 3},
		{[]float64{1, 2, 3, 4, 5}, 3},
	}
	for _, tt := range tests {
		if got := mathutil.Mean(tt.vals); math.Abs(got-tt.want) > 1e-9 {
			t.Errorf("Mean(%v): want %.4f, got %.4f", tt.vals, tt.want, got)
		}
	}
}

func TestStddev(t *testing.T) {
	tests := []struct {
		vals []float64
		want float64
	}{
		{nil, 0},
		{[]float64{5}, 0},
		{[]float64{2, 4}, math.Sqrt(2)},
	}
	for _, tt := range tests {
		if got := mathutil.Stddev(tt.vals); math.Abs(got-tt.want) > 1e-9 {
			t.Errorf("Stddev(%v): want %.4f, got %.4f", tt.vals, tt.want, got)
		}
	}
}

func TestSafeDiv(t *testing.T) {
	if got := mathutil.SafeDiv(3, 4); math.Abs(got-0.75) > 1e-9 {
		t.Errorf("SafeDiv(3,4): want 0.75, got %.4f", got)
	}
	if got := mathutil.SafeDiv(0, 0); got != 1.0 {
		t.Errorf("SafeDiv(0,0): want 1.0, got %.4f", got)
	}
}

func TestSafeDivFloat(t *testing.T) {
	if got := mathutil.SafeDivFloat(1.5, 3.0); math.Abs(got-0.5) > 1e-9 {
		t.Errorf("SafeDivFloat(1.5,3.0): want 0.5, got %.4f", got)
	}
	if got := mathutil.SafeDivFloat(0, 0); got != 1.0 {
		t.Errorf("SafeDivFloat(0,0): want 1.0, got %.4f", got)
	}
}
