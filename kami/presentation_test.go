package kami

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

// mockTheme is a minimal Theme for testing API endpoints.
type mockTheme struct{}

func (mockTheme) Name() string { return "Test Theme" }
func (mockTheme) AgentIntros() []AgentIntro {
	return []AgentIntro{
		{PersonaName: "Herald", Element: "Fire", Role: "Lead", Catchphrase: "I know."},
	}
}
func (mockTheme) NodeDescriptions() map[string]string {
	return map[string]string{"triage": "Sort cases", "investigate": "Find evidence"}
}
func (mockTheme) CostumeAssets() map[string]string {
	return map[string]string{"hat": "detective-hat"}
}
func (mockTheme) CooperationDialogs() []Dialog {
	return []Dialog{{From: "Herald", To: "Seeker", Message: "I already solved it."}}
}

// mockKabuki is a KabukiConfig with all sections populated.
type mockKabuki struct{}

func (mockKabuki) Hero() *HeroSection {
	return &HeroSection{Title: "Demo", Subtitle: "A test"}
}
func (mockKabuki) Problem() *ProblemSection {
	return &ProblemSection{Title: "The Problem", Narrative: "Things break."}
}
func (mockKabuki) Results() *ResultsSection {
	return &ResultsSection{
		Title:   "Results",
		Metrics: []Metric{{Label: "M19", Value: 0.83}},
	}
}
func (mockKabuki) Competitive() []Competitor {
	return []Competitor{{Name: "Us", Fields: map[string]string{"model": "graph"}, Highlight: true}}
}
func (mockKabuki) Architecture() *ArchitectureSection {
	return &ArchitectureSection{
		Title:      "Architecture",
		Components: []ArchComponent{{Name: "Core", Description: "The engine"}},
	}
}
func (mockKabuki) Roadmap() []Milestone {
	return []Milestone{{ID: "S1", Label: "Foundation", Status: "done"}}
}
func (mockKabuki) Closing() *ClosingSection {
	return &ClosingSection{Headline: "Thank you."}
}
func (mockKabuki) TransitionLine() string { return "Time to investigate." }

// partialKabuki returns nil for some sections.
type partialKabuki struct{ mockKabuki }

func (partialKabuki) Results() *ResultsSection       { return nil }
func (partialKabuki) Architecture() *ArchitectureSection { return nil }
func (partialKabuki) Competitive() []Competitor       { return nil }

func startTestServer(t *testing.T, cfg Config) string {
	t.Helper()
	bridge := NewEventBridge(nil)
	t.Cleanup(func() { bridge.Close() })

	cfg.Bridge = bridge
	srv := NewServer(cfg)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	httpAddr, _, err := srv.StartOnAvailablePort(ctx)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	return httpAddr
}

func getJSON(t *testing.T, url string) map[string]any {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET %s status = %d, want 200", url, resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if ct != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", ct)
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return body
}

func TestAPI_ThemeWithTheme(t *testing.T) {
	addr := startTestServer(t, Config{Theme: mockTheme{}})
	body := getJSON(t, fmt.Sprintf("http://%s/api/theme", addr))

	if body["name"] != "Test Theme" {
		t.Errorf("name = %v, want Test Theme", body["name"])
	}
	intros, ok := body["agent_intros"].([]any)
	if !ok || len(intros) != 1 {
		t.Errorf("agent_intros len = %v, want 1", len(intros))
	}
	nodes := body["node_descriptions"].(map[string]any)
	if nodes["triage"] != "Sort cases" {
		t.Errorf("node triage = %v, want 'Sort cases'", nodes["triage"])
	}
}

func TestAPI_ThemeWithoutTheme(t *testing.T) {
	addr := startTestServer(t, Config{})
	body := getJSON(t, fmt.Sprintf("http://%s/api/theme", addr))

	if body["name"] != "" {
		t.Errorf("name = %v, want empty", body["name"])
	}
}

func TestAPI_PipelineWithTheme(t *testing.T) {
	addr := startTestServer(t, Config{Theme: mockTheme{}})
	body := getJSON(t, fmt.Sprintf("http://%s/api/pipeline", addr))

	nodes := body["nodes"].(map[string]any)
	if len(nodes) != 2 {
		t.Errorf("nodes count = %d, want 2", len(nodes))
	}
	if nodes["investigate"] != "Find evidence" {
		t.Errorf("investigate = %v", nodes["investigate"])
	}
}

func TestAPI_KabukiAllSections(t *testing.T) {
	addr := startTestServer(t, Config{Theme: mockTheme{}, Kabuki: mockKabuki{}})
	body := getJSON(t, fmt.Sprintf("http://%s/api/kabuki", addr))

	hero := body["hero"].(map[string]any)
	if hero["title"] != "Demo" {
		t.Errorf("hero.title = %v, want Demo", hero["title"])
	}
	if body["transition_line"] != "Time to investigate." {
		t.Errorf("transition_line = %v", body["transition_line"])
	}
	if body["problem"] == nil {
		t.Error("expected problem section")
	}
	if body["results"] == nil {
		t.Error("expected results section")
	}
	if body["closing"] == nil {
		t.Error("expected closing section")
	}
}

func TestAPI_KabukiNilConfig(t *testing.T) {
	addr := startTestServer(t, Config{})
	body := getJSON(t, fmt.Sprintf("http://%s/api/kabuki", addr))

	if body["hero"] != nil {
		t.Errorf("hero should be nil when no KabukiConfig, got %v", body["hero"])
	}
}

func TestAPI_KabukiPartialSections(t *testing.T) {
	addr := startTestServer(t, Config{Theme: mockTheme{}, Kabuki: partialKabuki{}})
	body := getJSON(t, fmt.Sprintf("http://%s/api/kabuki", addr))

	if body["hero"] == nil {
		t.Error("expected hero section from partial")
	}
	if body["results"] != nil {
		t.Errorf("results should be nil in partial, got %v", body["results"])
	}
	if body["architecture"] != nil {
		t.Errorf("architecture should be nil in partial, got %v", body["architecture"])
	}
	if body["competitive"] != nil {
		t.Errorf("competitive should be nil in partial, got %v", body["competitive"])
	}
	if body["closing"] == nil {
		t.Error("expected closing section from partial")
	}
}
