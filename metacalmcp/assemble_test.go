package metacalmcp

import (
	"log/slog"
	"testing"
	"time"

	"github.com/dpopsuev/origami"
	"github.com/dpopsuev/origami/metacal"
)

func TestAssembleProfiles_MergesAcrossRuns(t *testing.T) {
	dir := t.TempDir()
	store, err := metacal.NewFileRunStore(dir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}

	gpt4o := framework.ModelIdentity{ModelName: "gpt-4o", Provider: "OpenAI", Version: "2025-01"}
	claude := framework.ModelIdentity{ModelName: "claude-sonnet-4", Provider: "Anthropic", Version: "20250514"}

	run1 := metacal.RunReport{
		RunID:     "run-refactor",
		StartTime: time.Now(),
		EndTime:   time.Now(),
		Config:    metacal.DiscoveryConfig{ProbeID: "refactor-v1"},
		Results: []metacal.DiscoveryResult{
			{
				Model: gpt4o,
				Probe: metacal.ProbeResult{
					ProbeID:         "refactor-v1",
					DimensionScores: map[metacal.Dimension]float64{metacal.DimSpeed: 0.8, metacal.DimShortcutAffinity: 0.7, metacal.DimEvidenceDepth: 0.3},
				},
			},
			{
				Model: claude,
				Probe: metacal.ProbeResult{
					ProbeID:         "refactor-v1",
					DimensionScores: map[metacal.Dimension]float64{metacal.DimSpeed: 0.3, metacal.DimShortcutAffinity: 0.2, metacal.DimEvidenceDepth: 0.9},
				},
			},
		},
		UniqueModels: []framework.ModelIdentity{gpt4o, claude},
		TermReason:   "max_iterations_reached",
	}

	run2 := metacal.RunReport{
		RunID:     "run-debug",
		StartTime: time.Now(),
		EndTime:   time.Now(),
		Config:    metacal.DiscoveryConfig{ProbeID: "debug-v1"},
		Results: []metacal.DiscoveryResult{
			{
				Model: gpt4o,
				Probe: metacal.ProbeResult{
					ProbeID:         "debug-v1",
					DimensionScores: map[metacal.Dimension]float64{metacal.DimSpeed: 0.9, metacal.DimConvergenceThreshold: 0.6},
				},
			},
			{
				Model: claude,
				Probe: metacal.ProbeResult{
					ProbeID:         "debug-v1",
					DimensionScores: map[metacal.Dimension]float64{metacal.DimSpeed: 0.4, metacal.DimConvergenceThreshold: 0.9},
				},
			},
		},
		UniqueModels: []framework.ModelIdentity{gpt4o, claude},
		TermReason:   "max_iterations_reached",
	}

	if err := store.SaveRun(run1); err != nil {
		t.Fatalf("save run1: %v", err)
	}
	if err := store.SaveRun(run2); err != nil {
		t.Fatalf("save run2: %v", err)
	}

	srv := &Server{RunsDir: dir, log: defaultTestLogger()}
	out, err := srv.assembleProfilesFromStore()
	if err != nil {
		t.Fatalf("assembleProfilesFromStore: %v", err)
	}

	if out.ModelCount != 2 {
		t.Errorf("model count = %d, want 2", out.ModelCount)
	}
	if out.RunsUsed != 2 {
		t.Errorf("runs used = %d, want 2", out.RunsUsed)
	}

	for _, p := range out.Profiles {
		key := metacal.ModelKey(p.Model)
		if len(p.RawResults) != 2 {
			t.Errorf("model %s: raw results = %d, want 2 (one per probe)", key, len(p.RawResults))
		}

		if p.ElementMatch == "" {
			t.Errorf("model %s: element match is empty", key)
		}
		if len(p.SuggestedPersonas) == 0 {
			t.Errorf("model %s: no suggested personas", key)
		}
		if len(p.ElementScores) == 0 {
			t.Errorf("model %s: no element scores", key)
		}

		switch key {
		case "gpt-4o":
			speed := p.Dimensions[metacal.DimSpeed]
			if !approxEqual(speed, 0.85, 0.01) {
				t.Errorf("gpt-4o DimSpeed = %.2f, want ~0.85 (avg of 0.8 and 0.9)", speed)
			}
		case "claude-sonnet-4":
			speed := p.Dimensions[metacal.DimSpeed]
			if !approxEqual(speed, 0.35, 0.01) {
				t.Errorf("claude DimSpeed = %.2f, want ~0.35 (avg of 0.3 and 0.4)", speed)
			}
			conv := p.Dimensions[metacal.DimConvergenceThreshold]
			if !approxEqual(conv, 0.9, 0.01) {
				t.Errorf("claude DimConvergenceThreshold = %.2f, want 0.9 (from debug only)", conv)
			}
		}
	}
}

func TestAssembleProfiles_EmptyStore(t *testing.T) {
	dir := t.TempDir()
	srv := &Server{RunsDir: dir, log: defaultTestLogger()}

	out, err := srv.assembleProfilesFromStore()
	if err != nil {
		t.Fatalf("assembleProfilesFromStore: %v", err)
	}
	if out.ModelCount != 0 {
		t.Errorf("expected 0 profiles from empty store, got %d", out.ModelCount)
	}
}

func TestAssembleProfiles_NoRunsDir(t *testing.T) {
	srv := &Server{RunsDir: "", log: defaultTestLogger()}
	_, err := srv.assembleProfilesFromStore()
	if err == nil {
		t.Fatal("expected error when RunsDir is empty")
	}
}

func TestAssembleProfiles_SkipsResultsWithoutDimensionScores(t *testing.T) {
	dir := t.TempDir()
	store, _ := metacal.NewFileRunStore(dir)

	report := metacal.RunReport{
		RunID:  "run-legacy",
		Config: metacal.DiscoveryConfig{ProbeID: "refactor-v1"},
		Results: []metacal.DiscoveryResult{
			{
				Model: framework.ModelIdentity{ModelName: "gpt-4o", Provider: "OpenAI"},
				Probe: metacal.ProbeResult{
					ProbeID: "refactor-v1",
					Score:   metacal.ProbeScore{TotalScore: 0.5},
				},
			},
		},
		UniqueModels: []framework.ModelIdentity{{ModelName: "gpt-4o", Provider: "OpenAI"}},
		TermReason:   "done",
	}
	store.SaveRun(report)

	srv := &Server{RunsDir: dir, log: defaultTestLogger()}
	out, err := srv.assembleProfilesFromStore()
	if err != nil {
		t.Fatalf("assembleProfilesFromStore: %v", err)
	}

	for _, p := range out.Profiles {
		if len(p.RawResults) > 0 {
			t.Errorf("expected no raw results for legacy run without DimensionScores")
		}
	}
}

func approxEqual(a, b, epsilon float64) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d < epsilon
}

func defaultTestLogger() *slog.Logger {
	return slog.Default()
}
