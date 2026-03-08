// Package review provides a directory-backed store for probe transcripts
// and human reviews. Each transcript is a JSON file named {run_id}.json.
package review

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/dpopsuev/origami/ouroboros"
)

var validID = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// ValidIDPattern returns the compiled regex used for transcript ID validation.
func ValidIDPattern() *regexp.Regexp { return validID }

// TranscriptStore manages probe transcripts on disk.
type TranscriptStore struct {
	dir string
}

// NewTranscriptStore creates a store rooted at dir, creating it if needed.
func NewTranscriptStore(dir string) (*TranscriptStore, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("review store: mkdir %s: %w", dir, err)
	}
	return &TranscriptStore{dir: dir}, nil
}

// Dir returns the store's root directory.
func (s *TranscriptStore) Dir() string { return s.dir }

// Save persists a transcript. The file is named {run_id}.json.
func (s *TranscriptStore) Save(t *ouroboros.ProbeTranscript) error {
	if err := t.Validate(); err != nil {
		return err
	}
	if !validID.MatchString(t.RunID) {
		return fmt.Errorf("review store: invalid run_id %q (alphanumeric, dash, underscore only)", t.RunID)
	}

	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return fmt.Errorf("review store: marshal: %w", err)
	}

	path := filepath.Join(s.dir, t.RunID+".json")
	return os.WriteFile(path, data, 0o644)
}

// Get loads a single transcript by run ID.
func (s *TranscriptStore) Get(runID string) (*ouroboros.ProbeTranscript, error) {
	if !validID.MatchString(runID) {
		return nil, fmt.Errorf("review store: invalid run_id %q", runID)
	}

	path := filepath.Join(s.dir, runID+".json")
	return ouroboros.LoadTranscript(path)
}

// Score attaches a HumanReview to an existing transcript.
func (s *TranscriptStore) Score(runID string, review *ouroboros.HumanReview) error {
	if err := review.ValidateReview(); err != nil {
		return err
	}

	t, err := s.Get(runID)
	if err != nil {
		return err
	}

	t.Review = review
	return s.Save(t)
}

// TranscriptSummary is a lightweight listing entry.
type TranscriptSummary struct {
	RunID      string `json:"run_id"`
	SeedName   string `json:"seed_name"`
	Model      string `json:"model"`
	Difficulty string `json:"difficulty"`
	HasReview  bool   `json:"has_review"`
}

// List returns summaries of all transcripts, grouped by model.
func (s *TranscriptStore) List() (map[string][]TranscriptSummary, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, fmt.Errorf("review store: list: %w", err)
	}

	result := make(map[string][]TranscriptSummary)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		path := filepath.Join(s.dir, entry.Name())
		t, err := ouroboros.LoadTranscript(path)
		if err != nil {
			continue
		}

		summary := TranscriptSummary{
			RunID:      t.RunID,
			SeedName:   t.SeedName,
			Model:      t.Model,
			Difficulty: t.Difficulty,
			HasReview:  t.Review != nil,
		}
		result[t.Model] = append(result[t.Model], summary)
	}

	for model := range result {
		sort.Slice(result[model], func(i, j int) bool {
			return result[model][i].RunID < result[model][j].RunID
		})
	}

	return result, nil
}

// Models returns a sorted list of distinct model names that have transcripts.
func (s *TranscriptStore) Models() ([]string, error) {
	grouped, err := s.List()
	if err != nil {
		return nil, err
	}

	models := make([]string, 0, len(grouped))
	for m := range grouped {
		models = append(models, m)
	}
	sort.Strings(models)
	return models, nil
}
