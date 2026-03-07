package ouroboros

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// SeedCategory classifies the type of behavioral probe a seed represents.
type SeedCategory string

const (
	CategorySkill    SeedCategory = "SKILL"
	CategoryTrap     SeedCategory = "TRAP"
	CategoryBoundary SeedCategory = "BOUNDARY"
	CategoryIdentity SeedCategory = "IDENTITY"
	CategoryReframe  SeedCategory = "REFRAME"
)

var validCategories = map[SeedCategory]bool{
	CategorySkill: true, CategoryTrap: true,
	CategoryBoundary: true, CategoryIdentity: true,
	CategoryReframe: true,
}

var knownDimensions = map[Dimension]bool{
	DimSpeed: true, DimPersistence: true,
	DimConvergenceThreshold: true, DimShortcutAffinity: true,
	DimEvidenceDepth: true, DimFailureMode: true,
	DimGBWP: true,
}

// Pole represents one end of a dichotomous behavioral spectrum.
// Both poles are valid — they measure element affinity, not correctness.
type Pole struct {
	Signal          string               `yaml:"signal"`
	ElementAffinity map[Dimension]float64 `yaml:"element_affinity"`
}

// Seed is a YAML-defined behavioral probe. Adding a new probe means adding
// a YAML file, not writing Go code.
type Seed struct {
	Name                  string              `yaml:"name"`
	Version               string              `yaml:"version"`
	Dimensions            []Dimension         `yaml:"dimensions"`
	Category              SeedCategory        `yaml:"category"`
	Poles                 map[string]Pole     `yaml:"poles"`
	Context               string              `yaml:"context"`
	Rubric                string              `yaml:"rubric"`
	GeneratorInstructions string              `yaml:"generator_instructions"`
	GoldAnswer            string              `yaml:"gold_answer,omitempty"`
	GoldSignals           map[string][]string `yaml:"gold_signals,omitempty"`
	Difficulty            string              `yaml:"difficulty,omitempty"`
	OutputFormat          string              `yaml:"output_format,omitempty"`
	Rounds                int                 `yaml:"rounds,omitempty"`
	Verify                *SeedVerify         `yaml:"verify,omitempty"`
}

// Difficulty levels for progressive calibration.
const (
	DifficultyEasy   = "easy"
	DifficultyMedium = "medium"
	DifficultyHard   = "hard"
)

var validDifficulties = map[string]bool{
	"": true, DifficultyEasy: true, DifficultyMedium: true, DifficultyHard: true,
}

// SeedVerify defines execution-based verification for code-producing probes.
// When present, the Judge extracts code from the subject's response, runs
// compile + test + benchmark commands, and merges the binary pass/fail
// result into dimension scores. Behavioral LLM-based scoring is secondary.
type SeedVerify struct {
	Language           string `yaml:"language"`
	Compile            string `yaml:"compile"`
	Test               string `yaml:"test"`
	TestFile           string `yaml:"test_file"`
	Benchmark          string `yaml:"benchmark,omitempty"`
	BenchmarkThreshMs  int    `yaml:"benchmark_threshold_ms,omitempty"`
	SetupCommands      string `yaml:"setup_commands,omitempty"`
}

// GeneratorOutput is the structured output of the Generator node.
// The Subject only sees Question — PoleAnswers are judge-side reference material.
type GeneratorOutput struct {
	Question    string            `json:"question"    yaml:"question"`
	PoleAnswers map[string]string `json:"pole_answers" yaml:"pole_answers"`
}

// MechanicalVerifyResult records the outcome of compiling and testing
// a code-producing subject response. Nil when the seed has no verify config.
type MechanicalVerifyResult struct {
	Compiled         bool    `json:"compiled"`
	TestsPassed      bool    `json:"tests_passed"`
	BenchmarkPassed  bool    `json:"benchmark_passed,omitempty"`
	BenchmarkMs      int     `json:"benchmark_ms,omitempty"`
	CompileErr       string  `json:"compile_error,omitempty"`
	TestErr          string  `json:"test_error,omitempty"`
	BenchmarkErr     string  `json:"benchmark_error,omitempty"`
	Score            float64 `json:"score"`
}

// PoleResult is the structured output of the Judge node.
type PoleResult struct {
	SelectedPole     string                  `json:"selected_pole"    yaml:"selected_pole"`
	Confidence       float64                 `json:"confidence"       yaml:"confidence"`
	DimensionScores  map[Dimension]float64   `json:"dimension_scores" yaml:"dimension_scores"`
	Reasoning        string                  `json:"reasoning"        yaml:"reasoning"`
	GoldSignalScore  float64                 `json:"gold_signal_score,omitempty" yaml:"gold_signal_score,omitempty"`
	MechanicalVerify *MechanicalVerifyResult `json:"mechanical_verify,omitempty" yaml:"mechanical_verify,omitempty"`
	SelfVerifyScore  float64                 `json:"self_verify_score,omitempty" yaml:"self_verify_score,omitempty"`
}

// LoadSeed reads a seed YAML file and validates its structure.
func LoadSeed(path string) (*Seed, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("read seed %s: %w", path, err)
	}
	return ParseSeed(data)
}

// ParseSeed deserializes and validates a seed from YAML bytes.
func ParseSeed(data []byte) (*Seed, error) {
	var s Seed
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse seed YAML: %w", err)
	}
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return &s, nil
}

// Validate checks structural invariants: exactly 2 poles, known dimensions,
// known category, non-empty rubric and context.
func (s *Seed) Validate() error {
	if s.Name == "" {
		return fmt.Errorf("seed validation: name is required")
	}
	if !validCategories[s.Category] {
		return fmt.Errorf("seed validation: unknown category %q", s.Category)
	}
	if len(s.Dimensions) == 0 {
		return fmt.Errorf("seed validation: at least one dimension is required")
	}
	for _, dim := range s.Dimensions {
		if !knownDimensions[dim] {
			return fmt.Errorf("seed validation: unknown dimension %q", dim)
		}
	}
	if len(s.Poles) != 2 {
		return fmt.Errorf("seed validation: exactly 2 poles required, got %d", len(s.Poles))
	}
	if s.Context == "" {
		return fmt.Errorf("seed validation: context is required")
	}
	if s.Rubric == "" {
		return fmt.Errorf("seed validation: rubric is required")
	}
	if !validDifficulties[s.Difficulty] {
		return fmt.Errorf("seed validation: unknown difficulty %q", s.Difficulty)
	}
	if len(s.GoldSignals) > 0 {
		for pole := range s.GoldSignals {
			if _, ok := s.Poles[pole]; !ok {
				return fmt.Errorf("seed validation: gold_signals references unknown pole %q", pole)
			}
		}
	}
	if s.Rounds < 0 {
		return fmt.Errorf("seed validation: rounds must be non-negative")
	}
	return nil
}
