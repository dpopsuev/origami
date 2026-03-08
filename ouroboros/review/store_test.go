package review

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dpopsuev/origami/ouroboros"
)

func testTranscript(runID, seed, model string) *ouroboros.ProbeTranscript {
	return &ouroboros.ProbeTranscript{
		RunID:      runID,
		SeedName:   seed,
		Model:      model,
		Difficulty: "medium",
		Exchanges: []ouroboros.Exchange{
			{Role: "generate", Prompt: "q", Response: "a", Elapsed: time.Second},
		},
		Timestamp: time.Now(),
	}
}

func TestTranscriptStore_SaveAndGet(t *testing.T) {
	dir := t.TempDir()
	store, err := NewTranscriptStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	tr := testTranscript("run-1", "debug-skill", "gpt-4")
	if err := store.Save(tr); err != nil {
		t.Fatal(err)
	}

	got, err := store.Get("run-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.RunID != "run-1" {
		t.Errorf("got RunID %q, want run-1", got.RunID)
	}
	if got.SeedName != "debug-skill" {
		t.Errorf("got SeedName %q, want debug-skill", got.SeedName)
	}
	if len(got.Exchanges) != 1 {
		t.Errorf("got %d exchanges, want 1", len(got.Exchanges))
	}
}

func TestTranscriptStore_InvalidID(t *testing.T) {
	dir := t.TempDir()
	store, err := NewTranscriptStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	tr := testTranscript("../escape", "debug-skill", "gpt-4")
	if err := store.Save(tr); err == nil {
		t.Error("expected error for path-traversal ID")
	}

	if _, err := store.Get("../escape"); err == nil {
		t.Error("expected error for path-traversal ID")
	}
}

func TestTranscriptStore_Score(t *testing.T) {
	dir := t.TempDir()
	store, err := NewTranscriptStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	tr := testTranscript("run-2", "debug-skill", "gpt-4")
	if err := store.Save(tr); err != nil {
		t.Fatal(err)
	}

	review := &ouroboros.HumanReview{
		ReviewerID: "alice",
		Ratings:    map[string]int{"generate": 4, "subject": 3, "judge": 5},
		Notes:      "Good probe run.",
		Timestamp:  time.Now(),
	}
	if err := store.Score("run-2", review); err != nil {
		t.Fatal(err)
	}

	got, err := store.Get("run-2")
	if err != nil {
		t.Fatal(err)
	}
	if got.Review == nil {
		t.Fatal("expected review to be attached")
	}
	if got.Review.ReviewerID != "alice" {
		t.Errorf("got ReviewerID %q, want alice", got.Review.ReviewerID)
	}
	if got.Review.Ratings["judge"] != 5 {
		t.Errorf("got judge rating %d, want 5", got.Review.Ratings["judge"])
	}
}

func TestTranscriptStore_Score_InvalidRating(t *testing.T) {
	dir := t.TempDir()
	store, err := NewTranscriptStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	tr := testTranscript("run-3", "debug-skill", "gpt-4")
	if err := store.Save(tr); err != nil {
		t.Fatal(err)
	}

	review := &ouroboros.HumanReview{
		ReviewerID: "bob",
		Ratings:    map[string]int{"subject": 6},
		Timestamp:  time.Now(),
	}
	if err := store.Score("run-3", review); err == nil {
		t.Error("expected error for rating > 5")
	}
}

func TestTranscriptStore_List(t *testing.T) {
	dir := t.TempDir()
	store, err := NewTranscriptStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct{ id, seed, model string }{
		{"run-a", "debug-skill", "gpt-4"},
		{"run-b", "refactor-skill", "gpt-4"},
		{"run-c", "debug-skill", "claude-3"},
	} {
		if err := store.Save(testTranscript(tc.id, tc.seed, tc.model)); err != nil {
			t.Fatal(err)
		}
	}

	grouped, err := store.List()
	if err != nil {
		t.Fatal(err)
	}

	if len(grouped["gpt-4"]) != 2 {
		t.Errorf("gpt-4: got %d, want 2", len(grouped["gpt-4"]))
	}
	if len(grouped["claude-3"]) != 1 {
		t.Errorf("claude-3: got %d, want 1", len(grouped["claude-3"]))
	}
}

func TestTranscriptStore_Models(t *testing.T) {
	dir := t.TempDir()
	store, err := NewTranscriptStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct{ id, model string }{
		{"r1", "gpt-4"},
		{"r2", "claude-3"},
		{"r3", "gpt-4"},
	} {
		if err := store.Save(testTranscript(tc.id, "seed", tc.model)); err != nil {
			t.Fatal(err)
		}
	}

	models, err := store.Models()
	if err != nil {
		t.Fatal(err)
	}

	if len(models) != 2 {
		t.Fatalf("got %d models, want 2", len(models))
	}
	if models[0] != "claude-3" || models[1] != "gpt-4" {
		t.Errorf("got models %v, want [claude-3 gpt-4]", models)
	}
}

func TestTranscriptStore_GetNotFound(t *testing.T) {
	dir := t.TempDir()
	store, err := NewTranscriptStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := store.Get("nonexistent"); err == nil {
		t.Error("expected error for missing transcript")
	}
}

func TestTranscriptStore_FileOnDisk(t *testing.T) {
	dir := t.TempDir()
	store, err := NewTranscriptStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	tr := testTranscript("run-disk", "debug-skill", "gpt-4")
	if err := store.Save(tr); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(dir, "run-disk.json")
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected file at %s: %v", path, err)
	}
}
