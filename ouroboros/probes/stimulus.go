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
	Name             string `yaml:"name"`
	Input            string `yaml:"input"`
	Language         string `yaml:"language,omitempty"`
	ExpectedBehavior string `yaml:"expected_behavior,omitempty"`
}

// StimuliSet maps probe names to their stimuli.
type StimuliSet map[string]ProbeStimulus

// DefaultStimuli returns the built-in stimuli for all 5 Ouroboros probes.
func DefaultStimuli() StimuliSet {
	return StimuliSet{
		"refactor": {
			Name:             "refactor",
			Input:            ouroboros.MessyInput,
			Language:         "Go",
			ExpectedBehavior: "Refactored code with descriptive names, split functions, and comments",
		},
		"debug": {
			Name:             "debug",
			Input:            DebugInput,
			Language:         "",
			ExpectedBehavior: "Structured analysis identifying root cause, evidence, red herrings, and recommended fix",
		},
		"summarize": {
			Name:             "summarize",
			Input:            SummarizeInput,
			Language:         "",
			ExpectedBehavior: "Per-change summary with category (feature/refactor/bugfix/performance) and risk level",
		},
		"ambiguity": {
			Name:             "ambiguity",
			Input:            AmbiguityInput,
			Language:         "Go",
			ExpectedBehavior: "Implementation plan explicitly addressing contradictions with justified resolutions",
		},
		"persistence": {
			Name:             "persistence",
			Input:            PersistenceInput,
			Language:         "Go",
			ExpectedBehavior: "ParseConfig function handling all four input formats using only standard library",
		},
		"gbwp": {
			Name:             "gbwp",
			Input:            GBWPInput,
			Language:         "",
			ExpectedBehavior: "Correct verdict with high confidence and brief justification",
		},
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
