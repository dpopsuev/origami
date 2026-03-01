package ouroboros

import (
	"fmt"
	"time"

	framework "github.com/dpopsuev/origami"
	"gopkg.in/yaml.v3"
)

// PersonaSheet is a per-model routing document combining ModelProfile with
// circuit step affinity. It is the output artifact that the AffinityScheduler
// and agent router consume for performance optimization.
type PersonaSheet struct {
	Model             string                         `yaml:"model"              json:"model"`
	ElementMatch      framework.Element              `yaml:"element_match"      json:"element_match"`
	DimensionScores   map[Dimension]float64          `yaml:"dimension_scores"   json:"dimension_scores"`
	ElementScores     map[framework.Element]float64  `yaml:"element_scores"     json:"element_scores"`
	SuggestedPersonas map[string]string              `yaml:"suggested_personas" json:"suggested_personas"`
	CostProfile       framework.CostProfile          `yaml:"cost_profile"       json:"cost_profile"`
	GeneratedAt       time.Time                      `yaml:"generated_at"       json:"generated_at"`
}

// EmitPersonaSheet combines a ModelProfile with a circuit definition to produce
// a per-model routing document. The circuit steps determine which persona
// suggestions to include based on step affinity scores.
func EmitPersonaSheet(profile ModelProfile, circuit framework.CircuitDef) (*PersonaSheet, error) {
	if profile.Model.ModelName == "" {
		return nil, fmt.Errorf("model identity is empty")
	}

	stepAffinity := DeriveStepAffinity(profile)

	suggestions := make(map[string]string)
	for _, node := range circuit.Nodes {
		if node.Name == "_done" || node.Name == "" {
			continue
		}
		affinity, ok := stepAffinity[node.Name]
		if !ok || affinity <= 0 {
			continue
		}
		element := suggestElementForStep(node.Name, profile)
		suggestions[node.Name] = string(element) + "-specialist"
	}

	if len(circuit.Nodes) > 0 && len(suggestions) == 0 {
		for _, node := range circuit.Nodes {
			if node.Name == "_done" || node.Name == "" {
				continue
			}
			suggestions[node.Name] = string(profile.ElementMatch) + "-generalist"
		}
	}

	return &PersonaSheet{
		Model:             profile.Model.String(),
		ElementMatch:      profile.ElementMatch,
		DimensionScores:   profile.Dimensions,
		ElementScores:     profile.ElementScores,
		SuggestedPersonas: suggestions,
		CostProfile:       profile.CostProfile,
		GeneratedAt:       time.Now(),
	}, nil
}

// suggestElementForStep returns the best element for a circuit step based on
// the step's dimensional requirements and the model's measured profile.
func suggestElementForStep(step string, profile ModelProfile) framework.Element {
	stepDimMap := map[string][]Dimension{
		"recall":      {DimSpeed, DimShortcutAffinity},
		"triage":      {DimSpeed, DimConvergenceThreshold},
		"resolve":     {DimEvidenceDepth, DimConvergenceThreshold},
		"investigate": {DimEvidenceDepth, DimPersistence, DimConvergenceThreshold},
		"correlate":   {DimPersistence, DimEvidenceDepth},
		"review":      {DimConvergenceThreshold, DimFailureMode},
		"report":      {DimSpeed, DimEvidenceDepth},
	}

	dims, ok := stepDimMap[step]
	if !ok {
		return profile.ElementMatch
	}

	stepProfile := ModelProfile{Dimensions: make(map[Dimension]float64)}
	for _, dim := range dims {
		stepProfile.Dimensions[dim] = profile.Dimensions[dim]
	}

	return ElementMatch(stepProfile)
}

// ProviderHints returns a map of circuit step names to preferred provider
// names, derived from element affinity and known provider-element mappings.
// Consumers (e.g., ProviderRouter) use this for empirical routing.
func (ps *PersonaSheet) ProviderHints(providerElements map[string]framework.Element) map[string]string {
	elementProviders := make(map[framework.Element]string)
	for provider, element := range providerElements {
		elementProviders[element] = provider
	}

	hints := make(map[string]string)
	for step, persona := range ps.SuggestedPersonas {
		element := extractElement(persona)
		if element == "" {
			continue
		}
		if provider, ok := elementProviders[framework.Element(element)]; ok {
			hints[step] = provider
		}
	}
	return hints
}

func extractElement(persona string) string {
	for i := 0; i < len(persona); i++ {
		if persona[i] == '-' {
			return persona[:i]
		}
	}
	return persona
}

// MarshalYAML returns the PersonaSheet as human-readable YAML bytes.
func (ps *PersonaSheet) MarshalYAML() ([]byte, error) {
	return yaml.Marshal(ps)
}
