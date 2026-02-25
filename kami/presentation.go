package kami

import (
	"encoding/json"
	"net/http"
)

// PresentationConfig provides section data for the Kami presentation engine.
// Consumers implement this interface to turn Kami into a branded, section-based
// presentation SPA. Methods that return nil cause that section to be skipped.
//
// Agent intros and pipeline nodes are derived from Theme (not duplicated here).
type PresentationConfig interface {
	Hero() *HeroSection
	Problem() *ProblemSection
	Results() *ResultsSection
	Competitive() []Competitor
	Architecture() *ArchitectureSection
	Roadmap() []Milestone
	Closing() *ClosingSection
	TransitionLine() string
}

// HeroSection is the full-viewport opening slide.
type HeroSection struct {
	Title     string `json:"title"`
	Subtitle  string `json:"subtitle"`
	Presenter string `json:"presenter,omitempty"`
	Logo      string `json:"logo,omitempty"`
	Framework string `json:"framework,omitempty"`
}

// ProblemSection describes the pain points being addressed.
type ProblemSection struct {
	Title      string   `json:"title"`
	Narrative  string   `json:"narrative"`
	BulletPoints []string `json:"bullet_points"`
	Stat       string   `json:"stat,omitempty"`
	StatLabel  string   `json:"stat_label,omitempty"`
}

// Metric is a single result data point with a 0-1 value.
type Metric struct {
	Label string  `json:"label"`
	Value float64 `json:"value"`
	Color string  `json:"color,omitempty"`
}

// SummaryCard is a small stat card (e.g. "19/21 Metrics passing").
type SummaryCard struct {
	Value string `json:"value"`
	Label string `json:"label"`
	Color string `json:"color,omitempty"`
}

// ResultsSection presents calibration or benchmark outcomes.
type ResultsSection struct {
	Title       string        `json:"title"`
	Description string        `json:"description,omitempty"`
	Metrics     []Metric      `json:"metrics"`
	Summary     []SummaryCard `json:"summary,omitempty"`
}

// Competitor is one row in a competitive comparison table.
type Competitor struct {
	Name      string            `json:"name"`
	Fields    map[string]string `json:"fields"`
	Highlight bool              `json:"highlight,omitempty"`
}

// ArchitectureSection describes the system architecture.
type ArchitectureSection struct {
	Title      string          `json:"title"`
	Components []ArchComponent `json:"components"`
	Footer     string          `json:"footer,omitempty"`
}

// ArchComponent is a box in the architecture diagram.
type ArchComponent struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Color       string `json:"color,omitempty"`
}

// Milestone is a point on the roadmap timeline.
type Milestone struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Status string `json:"status"` // "done", "current", "future"
}

// ClosingSection is the final slide.
type ClosingSection struct {
	Headline string   `json:"headline"`
	Tagline  string   `json:"tagline,omitempty"`
	Lines    []string `json:"lines,omitempty"`
}

// presentationPayload is the JSON envelope for /api/presentation.
type presentationPayload struct {
	Hero           *HeroSection         `json:"hero,omitempty"`
	Problem        *ProblemSection      `json:"problem,omitempty"`
	Results        *ResultsSection      `json:"results,omitempty"`
	Competitive    []Competitor         `json:"competitive,omitempty"`
	Architecture   *ArchitectureSection `json:"architecture,omitempty"`
	Roadmap        []Milestone          `json:"roadmap,omitempty"`
	Closing        *ClosingSection      `json:"closing,omitempty"`
	TransitionLine string               `json:"transition_line,omitempty"`
}

// themePayload is the JSON envelope for /api/theme.
type themePayload struct {
	Name               string            `json:"name"`
	AgentIntros        []AgentIntro      `json:"agent_intros"`
	NodeDescriptions   map[string]string `json:"node_descriptions"`
	CostumeAssets      map[string]string `json:"costume_assets"`
	CooperationDialogs []Dialog          `json:"cooperation_dialogs"`
}

// pipelinePayload is the JSON envelope for /api/pipeline.
type pipelinePayload struct {
	Nodes map[string]string `json:"nodes"`
}

func (s *Server) handleThemeAPI(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.cfg.Theme == nil {
		json.NewEncoder(w).Encode(themePayload{})
		return
	}
	json.NewEncoder(w).Encode(themePayload{
		Name:               s.cfg.Theme.Name(),
		AgentIntros:        s.cfg.Theme.AgentIntros(),
		NodeDescriptions:   s.cfg.Theme.NodeDescriptions(),
		CostumeAssets:      s.cfg.Theme.CostumeAssets(),
		CooperationDialogs: s.cfg.Theme.CooperationDialogs(),
	})
}

func (s *Server) handlePipelineAPI(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.cfg.Theme == nil {
		json.NewEncoder(w).Encode(pipelinePayload{})
		return
	}
	json.NewEncoder(w).Encode(pipelinePayload{
		Nodes: s.cfg.Theme.NodeDescriptions(),
	})
}

func (s *Server) handlePresentationAPI(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.cfg.Presentation == nil {
		json.NewEncoder(w).Encode(presentationPayload{})
		return
	}
	p := s.cfg.Presentation
	json.NewEncoder(w).Encode(presentationPayload{
		Hero:           p.Hero(),
		Problem:        p.Problem(),
		Results:        p.Results(),
		Competitive:    p.Competitive(),
		Architecture:   p.Architecture(),
		Roadmap:        p.Roadmap(),
		Closing:        p.Closing(),
		TransitionLine: p.TransitionLine(),
	})
}
