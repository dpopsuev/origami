package probes

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultStimuli_AllProbesPresent(t *testing.T) {
	stimuli := DefaultStimuli()
	expected := []string{"refactor", "debug", "summarize", "ambiguity", "persistence"}
	for _, name := range expected {
		s, ok := stimuli[name]
		if !ok {
			t.Errorf("missing stimulus %q", name)
			continue
		}
		if s.Input == "" {
			t.Errorf("stimulus %q has empty Input", name)
		}
		if s.Name != name {
			t.Errorf("stimulus Name = %q, want %q", s.Name, name)
		}
		if s.ExpectedBehavior == "" {
			t.Errorf("stimulus %q has empty ExpectedBehavior", name)
		}
	}
}

func TestDefaultStimuli_LanguageFields(t *testing.T) {
	stimuli := DefaultStimuli()
	wantLang := map[string]string{
		"refactor":    "Go",
		"debug":       "",
		"summarize":   "",
		"ambiguity":   "Go",
		"persistence": "Go",
	}
	for name, want := range wantLang {
		if got := stimuli[name].Language; got != want {
			t.Errorf("stimulus %q Language = %q, want %q", name, got, want)
		}
	}
}

func TestLoadStimuli(t *testing.T) {
	yaml := `stimuli:
  - name: debug
    input: "custom debug input"
    language: "Python"
    expected_behavior: "Find the bug"
  - name: newprobe
    input: "brand new probe"
`
	dir := t.TempDir()
	path := filepath.Join(dir, "stimuli.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadStimuli(path)
	if err != nil {
		t.Fatalf("LoadStimuli: %v", err)
	}

	if loaded["debug"].Input != "custom debug input" {
		t.Errorf("debug input = %q, want custom", loaded["debug"].Input)
	}
	if loaded["debug"].Language != "Python" {
		t.Errorf("debug Language = %q, want Python", loaded["debug"].Language)
	}
	if loaded["debug"].ExpectedBehavior != "Find the bug" {
		t.Errorf("debug ExpectedBehavior = %q, want 'Find the bug'", loaded["debug"].ExpectedBehavior)
	}
	if _, ok := loaded["newprobe"]; !ok {
		t.Error("newprobe not loaded")
	}
	if loaded["newprobe"].Language != "" {
		t.Errorf("newprobe Language = %q, want empty (omitted)", loaded["newprobe"].Language)
	}
}

func TestStimuliSet_Merge(t *testing.T) {
	base := DefaultStimuli()
	override := StimuliSet{
		"debug": {Name: "debug", Input: "overridden"},
	}

	merged := base.Merge(override)
	if merged["debug"].Input != "overridden" {
		t.Errorf("debug not overridden: %q", merged["debug"].Input)
	}
	if merged["ambiguity"].Input == "" {
		t.Error("ambiguity should be preserved from base")
	}
}

func TestBuildPromptFunctions_UseStimulus(t *testing.T) {
	custom := ProbeStimulus{Name: "test", Input: "custom input text"}

	tests := []struct {
		name  string
		build func(ProbeStimulus) string
	}{
		{"ambiguity", BuildAmbiguityPrompt},
		{"debug", BuildDebugPrompt},
		{"summarize", BuildSummarizePrompt},
		{"persistence", BuildPersistencePrompt},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.build(custom)
			if result != "custom input text" {
				t.Errorf("expected custom input, got %q", result)
			}
		})
	}

	t.Run("refactor", func(t *testing.T) {
		result := BuildRefactorPrompt(custom)
		if result == "" {
			t.Error("refactor prompt is empty")
		}
		if !contains(result, "custom input text") {
			t.Error("refactor prompt should contain stimulus input")
		}
	})
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
