package framework

import "testing"

func TestEvidenceSNR(t *testing.T) {
	cases := []struct {
		in, out int
		want    float64
	}{
		{10, 5, 0.5},
		{10, 10, 1.0},
		{10, 0, 0.0},
		{0, 5, 0.0},
		{0, 0, 0.0},
		{4, 3, 0.75},
	}
	for _, tc := range cases {
		got := EvidenceSNR(tc.in, tc.out)
		if got != tc.want {
			t.Errorf("EvidenceSNR(%d, %d) = %f, want %f", tc.in, tc.out, got, tc.want)
		}
	}
}
