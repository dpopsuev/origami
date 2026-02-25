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
}

// GeneratorOutput is the structured output of the Generator node.
// The Subject only sees Question — PoleAnswers are judge-side reference material.
type GeneratorOutput struct {
	Question    string            `json:"question"    yaml:"question"`
	PoleAnswers map[string]string `json:"pole_answers" yaml:"pole_answers"`
}

// PoleResult is the structured output of the Judge node.
type PoleResult struct {
	SelectedPole    string               `json:"selected_pole"    yaml:"selected_pole"`
	Confidence      float64              `json:"confidence"       yaml:"confidence"`
	DimensionScores map[Dimension]float64 `json:"dimension_scores" yaml:"dimension_scores"`
	Reasoning       string               `json:"reasoning"        yaml:"reasoning"`
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
	return nil
}
