package ouroboros

import (
	"os"
	"path/filepath"
	"testing"
)

const validSeedYAML = `
name: test-seed
version: "1.0"
dimensions: [speed, evidence_depth]
category: SKILL
poles:
  systematic:
    signal: step-by-step, thorough analysis
    element_affinity:
      speed: 0.3
      evidence_depth: 0.9
  heuristic:
    signal: pattern matching, quick answer
    element_affinity:
      speed: 0.9
      evidence_depth: 0.3
context: |
  You are reviewing a complex Go function.
rubric: |
  Evaluate whether the response uses systematic analysis or heuristic shortcuts.
generator_instructions: |
  Create a realistic code review scenario.
`

func TestParseSeed_Valid(t *testing.T) {
	s, err := ParseSeed([]byte(validSeedYAML))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Name != "test-seed" {
		t.Errorf("name = %q, want test-seed", s.Name)
	}
	if s.Category != CategorySkill {
		t.Errorf("category = %q, want SKILL", s.Category)
	}
	if len(s.Poles) != 2 {
		t.Errorf("poles count = %d, want 2", len(s.Poles))
	}
	if len(s.Dimensions) != 2 {
		t.Errorf("dimensions count = %d, want 2", len(s.Dimensions))
	}
	pole, ok := s.Poles["systematic"]
	if !ok {
		t.Fatal("missing pole 'systematic'")
	}
	if pole.ElementAffinity[DimEvidenceDepth] != 0.9 {
		t.Errorf("systematic.evidence_depth = %v, want 0.9", pole.ElementAffinity[DimEvidenceDepth])
	}
}

func TestParseSeed_MissingPoles(t *testing.T) {
	yaml := `
name: one-pole
version: "1.0"
dimensions: [speed]
category: SKILL
poles:
  only_one:
    signal: solo
    element_affinity:
      speed: 0.5
context: some context
rubric: some rubric
`
	_, err := ParseSeed([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for single pole, got nil")
	}
}

func TestParseSeed_UnknownDimension(t *testing.T) {
	yaml := `
name: bad-dim
version: "1.0"
dimensions: [speed, creativity]
category: SKILL
poles:
  a:
    signal: a
    element_affinity: {speed: 0.5}
  b:
    signal: b
    element_affinity: {speed: 0.5}
context: ctx
rubric: rubric
`
	_, err := ParseSeed([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for unknown dimension, got nil")
	}
}

func TestParseSeed_UnknownCategory(t *testing.T) {
	yaml := `
name: bad-cat
version: "1.0"
dimensions: [speed]
category: INVALID
poles:
  a:
    signal: a
    element_affinity: {speed: 0.5}
  b:
    signal: b
    element_affinity: {speed: 0.5}
context: ctx
rubric: rubric
`
	_, err := ParseSeed([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for unknown category, got nil")
	}
}

func TestParseSeed_MissingContext(t *testing.T) {
	yaml := `
name: no-ctx
version: "1.0"
dimensions: [speed]
category: SKILL
poles:
  a:
    signal: a
    element_affinity: {speed: 0.5}
  b:
    signal: b
    element_affinity: {speed: 0.5}
context: ""
rubric: rubric
`
	_, err := ParseSeed([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for missing context, got nil")
	}
}

func TestParseSeed_MissingRubric(t *testing.T) {
	yaml := `
name: no-rubric
version: "1.0"
dimensions: [speed]
category: SKILL
poles:
  a:
    signal: a
    element_affinity: {speed: 0.5}
  b:
    signal: b
    element_affinity: {speed: 0.5}
context: some context
rubric: ""
`
	_, err := ParseSeed([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for missing rubric, got nil")
	}
}

func TestLoadSeed_FromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.yaml")
	if err := os.WriteFile(path, []byte(validSeedYAML), 0644); err != nil {
		t.Fatal(err)
	}
	s, err := LoadSeed(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Name != "test-seed" {
		t.Errorf("name = %q, want test-seed", s.Name)
	}
}

func TestLoadSeed_FileNotFound(t *testing.T) {
	_, err := LoadSeed("/nonexistent/path/seed.yaml")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestParseSeed_AllCategories(t *testing.T) {
	for _, cat := range []SeedCategory{CategorySkill, CategoryTrap, CategoryBoundary, CategoryIdentity, CategoryReframe} {
		s := Seed{
			Name:       "cat-test",
			Version:    "1.0",
			Dimensions: []Dimension{DimSpeed},
			Category:   cat,
			Poles: map[string]Pole{
				"a": {Signal: "a", ElementAffinity: map[Dimension]float64{DimSpeed: 0.5}},
				"b": {Signal: "b", ElementAffinity: map[Dimension]float64{DimSpeed: 0.5}},
			},
			Context: "ctx",
			Rubric:  "rubric",
		}
		if err := s.Validate(); err != nil {
			t.Errorf("category %q should be valid: %v", cat, err)
		}
	}
}
