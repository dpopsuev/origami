package ouroboros

import (
	"context"
	"time"

	"github.com/dpopsuev/origami"
)

// ---------------------------------------------------------------------------
// Ouroboros — behavioral dimensions and probe pipeline
// ---------------------------------------------------------------------------

// Dimension represents a behavioral axis measured by Ouroboros probes.
// Each maps 1:1 to an ElementTraits field, normalized to 0.0-1.0.
type Dimension string

const (
	DimSpeed                Dimension = "speed"
	DimPersistence          Dimension = "persistence"
	DimConvergenceThreshold Dimension = "convergence_threshold"
	DimShortcutAffinity     Dimension = "shortcut_affinity"
	DimEvidenceDepth        Dimension = "evidence_depth"
	DimFailureMode          Dimension = "failure_mode"
)

// AllDimensions returns the six behavioral dimensions in canonical order.
func AllDimensions() []Dimension {
	return []Dimension{
		DimSpeed, DimPersistence, DimConvergenceThreshold,
		DimShortcutAffinity, DimEvidenceDepth, DimFailureMode,
	}
}

// ProbeStep identifies a step in the Ouroboros probe pipeline,
// analogous to orchestrate.PipelineStep but framework-level.
type ProbeStep string

const (
	StepRefactor    ProbeStep = "O0_REFACTOR"
	StepDebug       ProbeStep = "O1_DEBUG"
	StepSummarize   ProbeStep = "O2_SUMMARIZE"
	StepAmbiguity   ProbeStep = "O3_AMBIGUITY"
	StepPersistence ProbeStep = "O4_PERSISTENCE"
)

// ProbeSpec defines a behavioral probe: its identity, what it measures,
// and the stimulus material. The agent does NOT know which dimensions
// are being measured — it just does the task.
type ProbeSpec struct {
	ID                string      `json:"id"`
	Name              string      `json:"name"`
	Description       string      `json:"description"`
	Step              ProbeStep   `json:"step"`
	Dimensions        []Dimension `json:"dimensions"`
	Input             string      `json:"input"`
	ExpectedBehaviors []string    `json:"expected_behaviors,omitempty"`
}

// ModelProfile is the empirical output of one Ouroboros cycle for one model.
// Append-only: historical profiles are never overwritten.
type ModelProfile struct {
	Model            framework.ModelIdentity        `json:"model"`
	BatteryVersion   string                         `json:"battery_version"`
	Timestamp        time.Time                      `json:"timestamp"`
	Dimensions       map[Dimension]float64          `json:"dimensions"`
	ElementMatch     framework.Element              `json:"element_match"`
	ElementScores    map[framework.Element]float64  `json:"element_scores"`
	SuggestedPersonas []string                      `json:"suggested_personas,omitempty"`
	CostProfile      framework.CostProfile          `json:"cost_profile"`
	RawResults       []ProbeResult                  `json:"raw_results"`
}

// Dispatcher sends a probe prompt to a model and returns the raw text response.
// The ProbeStep parameter gives routing context (which probe is running),
// enabling per-probe model selection by domain adapters.
type Dispatcher func(ctx context.Context, step ProbeStep, prompt string) (string, error)

// DiscoveryConfig controls the recursive discovery loop.
type DiscoveryConfig struct {
	MaxIterations      int    `json:"max_iterations"`
	ProbeID            string `json:"probe_id"`
	TerminateOnRepeat  bool   `json:"terminate_on_repeat"`
}

// DefaultConfig returns a sensible starting configuration.
func DefaultConfig() DiscoveryConfig {
	return DiscoveryConfig{
		MaxIterations:     15,
		ProbeID:           "refactor-v1",
		TerminateOnRepeat: true,
	}
}

// ProbeScore holds the scored dimensions from a refactoring probe.
type ProbeScore struct {
	Renames           int     `json:"renames"`
	FunctionSplits    int     `json:"function_splits"`
	CommentsAdded     int     `json:"comments_added"`
	StructuralChanges int     `json:"structural_changes"`
	TotalScore        float64 `json:"total_score"`
}

// ProbeResult captures the raw output and scored result of a single probe.
// Legacy discovery probes populate Score; Ouroboros probes populate DimensionScores.
type ProbeResult struct {
	ProbeID         string               `json:"probe_id"`
	RawOutput       string               `json:"raw_output"`
	Score           ProbeScore           `json:"score"`
	DimensionScores map[Dimension]float64 `json:"dimension_scores,omitempty"`
	Elapsed         time.Duration        `json:"elapsed_ns"`
	TokensUsed      int                  `json:"tokens_used,omitempty"`
}

// DiscoveryResult records one iteration of the negation discovery loop.
type DiscoveryResult struct {
	Iteration       int                    `json:"iteration"`
	Model           framework.ModelIdentity `json:"model"`
	ExclusionPrompt string                 `json:"exclusion_prompt"`
	Probe           ProbeResult            `json:"probe"`
	Timestamp       time.Time              `json:"timestamp"`
}

// RunReport is the complete output of a discovery run. Persisted as
// append-only JSON — each run gets its own file, never overwritten.
type RunReport struct {
	RunID        string                    `json:"run_id"`
	StartTime    time.Time                 `json:"start_time"`
	EndTime      time.Time                 `json:"end_time"`
	Config       DiscoveryConfig           `json:"config"`
	Results      []DiscoveryResult         `json:"results"`
	UniqueModels []framework.ModelIdentity `json:"unique_models"`
	TermReason   string                    `json:"termination_reason"`
}

// ModelNames returns a sorted list of unique model names from the report.
func (r *RunReport) ModelNames() []string {
	names := make([]string, len(r.UniqueModels))
	for i, m := range r.UniqueModels {
		names[i] = m.String()
	}
	return names
}
