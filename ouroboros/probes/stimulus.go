package probes

import (
	"fmt"
	"os"

	"github.com/dpopsuev/origami/ouroboros"
	"gopkg.in/yaml.v3"
)

// ProbeStimulus is a named input stimulus for an Ouroboros behavioral probe.
// Consumers can load custom stimuli from YAML to replace or extend the defaults.
type ProbeStimulus struct {
	Name  string `yaml:"name"`
	Input string `yaml:"input"`
}

// StimuliSet maps probe names to their stimuli.
type StimuliSet map[string]ProbeStimulus

// DefaultStimuli returns the built-in stimuli for all 5 Ouroboros probes.
func DefaultStimuli() StimuliSet {
	return StimuliSet{
		"refactor":    {Name: "refactor", Input: ouroboros.MessyInput},
		"debug":       {Name: "debug", Input: DebugInput},
		"summarize":   {Name: "summarize", Input: SummarizeInput},
		"ambiguity":   {Name: "ambiguity", Input: AmbiguityInput},
		"persistence": {Name: "persistence", Input: PersistenceInput},
	}
}

// LoadStimuli reads probe stimuli from a YAML file. The file format is:
//
//	stimuli:
//	  - name: debug
//	    input: |
//	      === Custom debug scenario ===
//	      ...
func LoadStimuli(path string) (StimuliSet, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read stimuli file %s: %w", path, err)
	}

	var f struct {
		Stimuli []ProbeStimulus `yaml:"stimuli"`
	}
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parse stimuli file %s: %w", path, err)
	}

	set := make(StimuliSet, len(f.Stimuli))
	for _, s := range f.Stimuli {
		set[s.Name] = s
	}
	return set, nil
}

// Merge returns a new StimuliSet with overrides applied on top of base.
func (s StimuliSet) Merge(overrides StimuliSet) StimuliSet {
	merged := make(StimuliSet, len(s)+len(overrides))
	for k, v := range s {
		merged[k] = v
	}
	for k, v := range overrides {
		merged[k] = v
	}
	return merged
}
