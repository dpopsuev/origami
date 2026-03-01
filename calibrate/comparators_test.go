package calibrate

import (
	"testing"
)

func TestKeywordMatch(t *testing.T) {
	tests := []struct {
		text     string
		keywords []string
		want     int
	}{
		{"ptp4l clock drift detected", []string{"ptp4l", "clock", "drift"}, 3},
		{"PTP4L CLOCK DRIFT", []string{"ptp4l", "clock"}, 2},
		{"unrelated text", []string{"ptp4l", "clock"}, 0},
		{"", []string{"ptp4l"}, 0},
		{"some text", nil, 0},
	}
	for _, tt := range tests {
		got := KeywordMatch(tt.text, tt.keywords)
		if got != tt.want {
			t.Errorf("KeywordMatch(%q, %v) = %d, want %d", tt.text, tt.keywords, got, tt.want)
		}
	}
}

func TestEvidenceOverlap_ExactMatch(t *testing.T) {
	actual := []string{"repo:file.go:func1", "repo:other.go"}
	expected := []string{"repo:file.go:func1"}
	found, total := EvidenceOverlap(actual, expected)
	if found != 1 || total != 1 {
		t.Errorf("got %d/%d, want 1/1", found, total)
	}
}

func TestEvidenceOverlap_BaseMatch(t *testing.T) {
	actual := []string{"repo:src/pkg/file.go"}
	expected := []string{"repo:deep/nested/file.go"}
	found, total := EvidenceOverlap(actual, expected)
	if found != 1 || total != 1 {
		t.Errorf("got %d/%d, want 1/1 (filepath.Base match)", found, total)
	}
}

func TestEvidenceOverlap_RepoPrefixMatch(t *testing.T) {
	actual := []string{"myrepo:pkg/handler.go:HandleRequest"}
	expected := []string{"myrepo:handler.go:something"}
	found, total := EvidenceOverlap(actual, expected)
	if found != 1 || total != 1 {
		t.Errorf("got %d/%d, want 1/1 (repo+path prefix match)", found, total)
	}
}

func TestEvidenceOverlap_Empty(t *testing.T) {
	found, total := EvidenceOverlap(nil, nil)
	if found != 0 || total != 0 {
		t.Errorf("got %d/%d, want 0/0", found, total)
	}
}

func TestPathsEqual(t *testing.T) {
	if !PathsEqual([]string{"F0", "F1", "F2"}, []string{"F0", "F1", "F2"}) {
		t.Error("equal paths should match")
	}
	if PathsEqual([]string{"F0", "F1"}, []string{"F0", "F2"}) {
		t.Error("different paths should not match")
	}
	if PathsEqual([]string{"F0"}, []string{"F0", "F1"}) {
		t.Error("different lengths should not match")
	}
}

func TestSmokingGunWords(t *testing.T) {
	words := SmokingGunWords("ptp4l clock drift in version 4.20")
	if len(words) != 5 {
		t.Errorf("got %d words %v, want 5 (ptp4l, clock, drift, version, 4.20)", len(words), words)
	}
	for _, w := range words {
		if len(w) <= 3 {
			t.Errorf("word %q should be >3 chars", w)
		}
	}
}

func TestPearsonCorrelation_Perfect(t *testing.T) {
	x := []float64{1, 2, 3, 4, 5}
	y := []float64{2, 4, 6, 8, 10}
	r := PearsonCorrelation(x, y)
	if r < 0.999 {
		t.Errorf("r = %v, want ~1.0", r)
	}
}

func TestPearsonCorrelation_AllCorrect(t *testing.T) {
	x := []float64{0.8, 0.9, 0.7}
	y := []float64{1.0, 1.0, 1.0}
	r := PearsonCorrelation(x, y)
	if r != 1.0 {
		t.Errorf("r = %v, want 1.0 (all-correct stub mode)", r)
	}
}

func TestPearsonCorrelation_TooFew(t *testing.T) {
	r := PearsonCorrelation([]float64{1.0}, []float64{1.0})
	if r != 0 {
		t.Errorf("r = %v, want 0 (too few points)", r)
	}
}

func TestValuesMatch_Strings(t *testing.T) {
	if !valuesMatch("infrastructure", "Infrastructure") {
		t.Error("case-insensitive string match should work")
	}
	if valuesMatch("product", "infrastructure") {
		t.Error("different strings should not match")
	}
}

func TestValuesMatch_Slices(t *testing.T) {
	if !valuesMatch([]string{"F0", "F1"}, []string{"F0", "F1"}) {
		t.Error("equal slices should match")
	}
	if valuesMatch([]string{"F0", "F1"}, []string{"F0", "F2"}) {
		t.Error("different slices should not match")
	}
}

func TestToStringSlice(t *testing.T) {
	got := toStringSlice([]any{"a", "b", "c"})
	if len(got) != 3 || got[0] != "a" {
		t.Errorf("got %v, want [a b c]", got)
	}

	got2 := toStringSlice([]string{"x", "y"})
	if len(got2) != 2 || got2[0] != "x" {
		t.Errorf("got %v, want [x y]", got2)
	}

	if toStringSlice("not a slice") != nil {
		t.Error("non-slice should return nil")
	}
}
