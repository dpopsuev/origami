package probes

import (
	"testing"

	"github.com/dpopsuev/origami/ouroboros"
)

func TestExtractCodeBlock_TripleBacktick(t *testing.T) {
	raw := "Here's the code:\n```go\nfunc hello() {}\n```\nDone."
	code := ExtractCodeBlock(raw)
	if code != "func hello() {}\n" {
		t.Errorf("got %q", code)
	}
}

func TestExtractCodeBlock_NoBacktick(t *testing.T) {
	raw := "func hello() {}"
	code := ExtractCodeBlock(raw)
	if code != raw {
		t.Errorf("expected raw input, got %q", code)
	}
}

func TestScoreVerifyResult_AllPass(t *testing.T) {
	vr := &VerifyResult{Compiled: true, TestsPassed: true, BenchmarkPassed: true}
	scores := ScoreVerifyResult(vr)
	if scores[ouroboros.DimEvidenceDepth] != 1.0 {
		t.Errorf("expected 1.0, got %v", scores[ouroboros.DimEvidenceDepth])
	}
}

func TestScoreVerifyResult_CompileAndTest(t *testing.T) {
	vr := &VerifyResult{Compiled: true, TestsPassed: true}
	scores := ScoreVerifyResult(vr)
	if scores[ouroboros.DimEvidenceDepth] != 0.7 {
		t.Errorf("expected 0.7, got %v", scores[ouroboros.DimEvidenceDepth])
	}
}

func TestScoreVerifyResult_CompileOnly(t *testing.T) {
	vr := &VerifyResult{Compiled: true, TestsPassed: false}
	scores := ScoreVerifyResult(vr)
	if scores[ouroboros.DimEvidenceDepth] != 0.3 {
		t.Errorf("expected 0.3, got %v", scores[ouroboros.DimEvidenceDepth])
	}
}

func TestScoreVerifyResult_NothingPasses(t *testing.T) {
	vr := &VerifyResult{Compiled: false, TestsPassed: false}
	scores := ScoreVerifyResult(vr)
	if scores[ouroboros.DimEvidenceDepth] != 0.0 {
		t.Errorf("expected evidence 0.0, got %v", scores[ouroboros.DimEvidenceDepth])
	}
	if scores[ouroboros.DimShortcutAffinity] != 0.9 {
		t.Errorf("expected shortcut 0.9, got %v", scores[ouroboros.DimShortcutAffinity])
	}
}

func TestToMechanical_Fields(t *testing.T) {
	vr := &VerifyResult{
		Compiled:        true,
		TestsPassed:     true,
		BenchmarkPassed: true,
		BenchmarkMs:     42,
	}
	mvr := ToMechanical(vr)
	if !mvr.Compiled || !mvr.TestsPassed || !mvr.BenchmarkPassed {
		t.Error("expected all flags true")
	}
	if mvr.BenchmarkMs != 42 {
		t.Errorf("benchmark_ms: got %d, want 42", mvr.BenchmarkMs)
	}
	if mvr.Score != 1.0 {
		t.Errorf("score: got %f, want 1.0", mvr.Score)
	}
}

func TestToMechanical_Partial(t *testing.T) {
	vr := &VerifyResult{Compiled: true, TestsPassed: false, TestErr: "assertion failed"}
	mvr := ToMechanical(vr)
	if mvr.Score != 0.3 {
		t.Errorf("score: got %f, want 0.3", mvr.Score)
	}
	if mvr.TestErr != "assertion failed" {
		t.Errorf("test_err: got %q", mvr.TestErr)
	}
}

func TestMergeVerifyScores(t *testing.T) {
	dim := map[ouroboros.Dimension]float64{
		ouroboros.DimEvidenceDepth: 0.6,
	}
	verify := map[ouroboros.Dimension]float64{
		ouroboros.DimEvidenceDepth:    0.8,
		ouroboros.DimShortcutAffinity: 0.9,
	}
	MergeVerifyScores(dim, verify)
	if dim[ouroboros.DimEvidenceDepth] != 0.7 {
		t.Errorf("evidence: got %f, want 0.7", dim[ouroboros.DimEvidenceDepth])
	}
	if dim[ouroboros.DimShortcutAffinity] != 0.9 {
		t.Errorf("shortcut: got %f, want 0.9", dim[ouroboros.DimShortcutAffinity])
	}
}

func TestLanguageExtension(t *testing.T) {
	cases := map[string]string{
		"go":         ".go",
		"Go":         ".go",
		"python":     ".py",
		"rust":       ".rs",
		"typescript": ".ts",
		"unknown":    ".txt",
	}
	for lang, want := range cases {
		if got := languageExtension(lang); got != want {
			t.Errorf("languageExtension(%q) = %q, want %q", lang, got, want)
		}
	}
}
