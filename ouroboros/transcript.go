package ouroboros

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Exchange records a single prompt/response interaction during a probe walk.
type Exchange struct {
	Role     string        `json:"role"`
	Prompt   string        `json:"prompt"`
	Response string        `json:"response"`
	Elapsed  time.Duration `json:"elapsed_ns"`
}

// HumanReview records a human reviewer's assessment of a probe transcript.
type HumanReview struct {
	ReviewerID   string            `json:"reviewer_id"`
	Ratings      map[string]int    `json:"ratings"`
	Notes        string            `json:"notes"`
	PromptTuning map[string]string `json:"prompt_tuning,omitempty"`
	Timestamp    time.Time         `json:"timestamp"`
}

// ProbeTranscript captures the full record of a single probe walk,
// including all exchanges, the final result, and optional human review.
type ProbeTranscript struct {
	RunID      string       `json:"run_id"`
	SeedName   string       `json:"seed_name"`
	Model      string       `json:"model"`
	Difficulty string       `json:"difficulty"`
	Exchanges  []Exchange   `json:"exchanges"`
	PoleResult *PoleResult  `json:"pole_result,omitempty"`
	Review     *HumanReview `json:"review,omitempty"`
	Timestamp  time.Time    `json:"timestamp"`
}

// Validate checks required fields on ProbeTranscript.
func (t *ProbeTranscript) Validate() error {
	if t.RunID == "" {
		return fmt.Errorf("transcript: run_id is required")
	}
	if t.SeedName == "" {
		return fmt.Errorf("transcript: seed_name is required")
	}
	return nil
}

// ValidateReview checks HumanReview invariants.
func (r *HumanReview) ValidateReview() error {
	if r.ReviewerID == "" {
		return fmt.Errorf("review: reviewer_id is required")
	}
	for role, rating := range r.Ratings {
		if rating < 1 || rating > 5 {
			return fmt.Errorf("review: rating for %q must be 1-5, got %d", role, rating)
		}
	}
	if len(r.Notes) > 10240 {
		return fmt.Errorf("review: notes exceed 10KB limit")
	}
	return nil
}

// SaveTranscript writes a transcript as JSON to the given path.
func SaveTranscript(t *ProbeTranscript, path string) error {
	if err := t.Validate(); err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("transcript save: mkdir %s: %w", dir, err)
	}

	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return fmt.Errorf("transcript save: marshal: %w", err)
	}

	return os.WriteFile(path, data, 0o644)
}

// LoadTranscript reads a transcript from the given JSON file.
func LoadTranscript(path string) (*ProbeTranscript, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("transcript load: %w", err)
	}

	var t ProbeTranscript
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("transcript load: unmarshal: %w", err)
	}

	return &t, nil
}

// NewTranscriptRecorder returns a TranscriptRecorder that appends exchanges
// to the provided slice. Thread-unsafe — intended for single-walk use.
func NewTranscriptRecorder(exchanges *[]Exchange) TranscriptRecorder {
	return func(role string, prompt string, response string, elapsed time.Duration) {
		*exchanges = append(*exchanges, Exchange{
			Role:     role,
			Prompt:   prompt,
			Response: response,
			Elapsed:  elapsed,
		})
	}
}
