package rca

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
)

var updateGolden = flag.Bool("update-golden", false, "overwrite golden files with current output")

// goldenFixtureParams returns a TemplateParams with every field populated.
// Used by golden render tests and coverage tests.
func goldenFixtureParams() *TemplateParams {
	return &TemplateParams{
		SourceID: "launch-42",
		CaseID:   7,
		StepName: "", // set per-test
		Envelope: &EnvelopeParams{
			Name:   "ptp-ci-nightly",
			RunID:  "run-123",
			Status: "FAILED",
		},
		Env: map[string]string{
			"OCP_VERSION":      "4.21",
			"OPERATOR_VERSION": "4.21.0-rc1",
		},
		Git: &GitParams{
			Branch: "release-4.21",
			Commit: "abc1234def5678",
		},
		Failure: &FailureParams{
			TestName:     "[T-TSC] PTP Recovery after grandmaster clock switchover",
			ErrorMessage: "Expected clock class 6 but got 248 after 300s holdover timeout",
			LogSnippet:   "level=error msg=\"holdover timeout exceeded\" class=248 expected=6\nts2phc[123]: DPLL not locked",
			LogTruncated: true,
			Status:       "FAILED",
			Path:         "test/e2e/ptp_recovery_test.go",
		},
		Siblings: []SiblingParams{
			{ID: "1", Name: "[T-TSC] PTP Sync test", Status: "FAILED"},
			{ID: "2", Name: "[T-TSC] PTP Clock accuracy", Status: "PASSED"},
			{ID: "3", Name: "[T-TSC] PTP DPLL tracking", Status: "FAILED"},
		},
		Workspace: &WorkspaceParams{
			Repos: []RepoParams{
				{Name: "ptp-operator", Path: "/repos/ptp-operator", Purpose: "PTP operator reconciliation", Branch: "release-4.21"},
				{Name: "linuxptp-daemon", Path: "/repos/linuxptp-daemon", Purpose: "PTP sync logic", Branch: "main"},
				{Name: "cnf-gotests", Path: "/repos/cnf-gotests", Purpose: "test framework", Branch: "release-4.21"},
			},
			LaunchAttributes: []AttributeParams{
				{Key: "ocp_version", Value: "4.21.3", System: false},
				{Key: "cluster", Value: "lab-sno-01", System: false},
				{Key: "agent", Value: "internal", System: true},
			},
			JiraLinks: []JiraLinkParams{
				{TicketID: "OCPBUGS-12345", URL: "https://issues.redhat.com/browse/OCPBUGS-12345"},
			},
			AttrsStatus: Resolved,
			JiraStatus:  Resolved,
			ReposStatus: Resolved,
		},
		URLs: &URLParams{
			SourceDashboard: "https://rp.example.com/launches/42",
			SourceItem:      "https://rp.example.com/items/7",
		},
		AlwaysReadSources: []AlwaysReadSource{
			{Name: "PTP Domain Knowledge", Purpose: "PTP protocol reference", Content: "PTP uses Best Master Clock Algorithm (BMCA) to select the grandmaster."},
		},
		Prior: &PriorParams{
			RecallResult: &RecallResult{
				Match:        true,
				PriorRCAID:   42,
				SymptomID:    7,
				Confidence:   0.85,
				Reasoning:    "Same holdover timeout pattern as RCA #42",
				IsRegression: true,
			},
			TriageResult: &TriageResult{
				SymptomCategory:      "product",
				Severity:             "high",
				DefectTypeHypothesis: "pb001",
				CandidateRepos:       []string{"ptp-operator", "linuxptp-daemon"},
				SkipInvestigation:    false,
				ClockSkewSuspected:   true,
				CascadeSuspected:     false,
				DataQualityNotes:     "Log truncated at 4KB",
			},
			ResolveResult: &ResolveResult{
				SelectedRepos: []RepoSelection{
					{
						Name:       "linuxptp-daemon",
						Path:       "/repos/linuxptp-daemon",
						FocusPaths: []string{"pkg/daemon/", "api/v1/"},
						Branch:     "main",
						Reason:     "Triage indicates product bug in PTP sync daemon code",
					},
				},
				CrossRefStrategy: "Check test assertion in cnf-gotests, then verify SUT behavior in linuxptp-daemon.",
			},
			InvestigateResult: &InvestigateArtifact{
				RunID:            "launch-42",
				CaseIDs:          []string{"7"},
				RCAMessage:       "Holdover timeout changed from 300s to 60s in commit abc1234, causing premature clock class transition to 248.",
				DefectType:       "pb001",
				Component:        "linuxptp-daemon",
				ConvergenceScore: 0.85,
				EvidenceRefs:     []string{"linuxptp-daemon:pkg/daemon/config.go:abc1234", "cnf-gotests:test/e2e/ptp_recovery_test.go:TestRecovery"},
				GapBrief: &GapBrief{
					Verdict: VerdictLowConfidence,
					GapItems: []EvidenceGap{
						{Category: GapLogDepth, Description: "Log truncated at 4KB", WouldHelp: "Full log would show complete error chain", Source: "CI console"},
					},
				},
			},
			CorrelateResult: &CorrelateResult{
				IsDuplicate:       false,
				LinkedRCAID:       0,
				Confidence:        0.3,
				Reasoning:         "Different error patterns despite similar test names",
				CrossVersionMatch: true,
				AffectedVersions:  []string{"4.20", "4.21"},
			},
		},
		History: &HistoryParams{
			SymptomInfo: &SymptomInfoParams{
				Name:                  "PTP holdover timeout",
				OccurrenceCount:       5,
				FirstSeen:             "2025-11-01",
				LastSeen:              "2026-02-28",
				Status:                "active",
				IsDormantReactivation: true,
			},
			PriorRCAs: []PriorRCAParams{
				{
					ID:                42,
					Title:             "Holdover timeout regression",
					DefectType:        "pb001",
					Status:            "resolved",
					AffectedVersions:  "4.20, 4.21",
					JiraLink:          "OCPBUGS-12345",
					ResolvedAt:        "2026-01-15",
					DaysSinceResolved: 47,
				},
			},
		},
		RecallDigest: []RecallDigestEntry{
			{ID: 42, Component: "linuxptp-daemon", DefectType: "pb001", Summary: "Holdover timeout regression"},
			{ID: 38, Component: "cloud-event-proxy", DefectType: "pb001", Summary: "GNSS sync state mapping error"},
		},
		Taxonomy:   DefaultTaxonomy(),
		Timestamps: &TimestampParams{
			ClockPlaneNote:   "Timestamps are in UTC. CI cluster uses chrony for NTP sync.",
			ClockSkewWarning: "Detected 2.3s clock skew between worker nodes.",
		},
	}
}

func TestPromptGolden(t *testing.T) {
	steps := []struct {
		step   CircuitStep
		family string
	}{
		{StepF0Recall, "recall"},
		{StepF1Triage, "triage"},
		{StepF2Resolve, "resolve"},
		{StepF3Invest, "investigate"},
		{StepF4Correlate, "correlate"},
		{StepF5Review, "review"},
		{StepF6Report, "report"},
	}

	for _, tt := range steps {
		t.Run(tt.family, func(t *testing.T) {
			params := goldenFixtureParams()
			params.StepName = string(tt.step)

			templatePath := TemplatePathForStep(tt.step)
			if templatePath == "" {
				t.Fatalf("no template path for step %s", tt.step)
			}

			got, err := FillTemplateFS(DefaultPromptFS, templatePath, params)
			if err != nil {
				t.Fatalf("FillTemplateFS(%s): %v", templatePath, err)
			}

			goldenFile := filepath.Join("testdata", "golden", "prompt-"+tt.family+".md")

			if *updateGolden {
				if err := os.MkdirAll(filepath.Dir(goldenFile), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(goldenFile, []byte(got), 0644); err != nil {
					t.Fatal(err)
				}
				t.Logf("updated %s", goldenFile)
				return
			}

			want, err := os.ReadFile(goldenFile)
			if err != nil {
				t.Fatalf("read golden file (run with -update-golden to create): %v", err)
			}
			if got != string(want) {
				t.Errorf("output differs from golden file %s.\nRun with -update-golden to update.\n\nGot (first 500 chars):\n%s", goldenFile, truncate(got, 500))
			}
		})
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "...(truncated)"
}
