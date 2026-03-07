package probes

import (
	"testing"
)

func TestMultilangStimuli_LoadsAll(t *testing.T) {
	loaded, err := LoadStimuli("stimuli/multilang.yaml")
	if err != nil {
		t.Fatalf("LoadStimuli: %v", err)
	}
	if len(loaded) < 8 {
		t.Errorf("expected 8+ multilang stimuli, got %d", len(loaded))
	}
	for name, s := range loaded {
		if s.Input == "" {
			t.Errorf("multilang stimulus %q has empty Input", name)
		}
		if s.Language == "" {
			t.Errorf("multilang stimulus %q has empty Language", name)
		}
	}
}

func TestMultilangStimuli_Languages(t *testing.T) {
	loaded, err := LoadStimuli("stimuli/multilang.yaml")
	if err != nil {
		t.Fatalf("LoadStimuli: %v", err)
	}

	langs := make(map[string]int)
	for _, s := range loaded {
		langs[s.Language]++
	}

	for _, lang := range []string{"Python", "Rust", "TypeScript"} {
		if langs[lang] == 0 {
			t.Errorf("no stimuli for language %q", lang)
		}
	}
}
