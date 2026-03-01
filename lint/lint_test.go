package lint

import (
	"strings"
	"testing"

	framework "github.com/dpopsuev/origami"
)

func minimalYAML() []byte {
	return []byte(`
pipeline: test
description: a test pipeline
nodes:
  - name: recall
    element: fire
    transformer: llm
    prompt: "recall items"
  - name: triage
    element: earth
    transformer: llm
    prompt: "triage items"
edges:
  - id: e1
    name: recall to triage
    from: recall
    to: triage
  - id: e2
    name: triage to done
    from: triage
    to: _done
start: recall
done: _done
`)
}

func TestRun_CleanPipeline_ZeroFindings(t *testing.T) {
	findings, err := Run(minimalYAML(), "test.yaml", WithProfile(ProfileStrict))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(findings) != 0 {
		for _, f := range findings {
			t.Logf("  %s", f)
		}
		t.Fatalf("expected 0 findings on clean pipeline, got %d", len(findings))
	}
}

func TestRun_InvalidElement(t *testing.T) {
	yml := []byte(`
pipeline: test
description: test
nodes:
  - name: recall
    family: recall
    element: fyre
edges:
  - id: e1
    name: e1
    from: recall
    to: _done
start: recall
done: _done
`)
	findings, err := Run(yml, "test.yaml")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	found := false
	for _, f := range findings {
		if f.RuleID == "S2/invalid-element" {
			found = true
			if !strings.Contains(f.Message, "fyre") {
				t.Errorf("expected message to contain 'fyre', got %q", f.Message)
			}
			if f.Suggestion == "" {
				t.Error("expected a suggestion for 'fyre'")
			}
		}
	}
	if !found {
		t.Error("expected S2/invalid-element finding")
	}
}

func TestRun_InvalidMergeStrategy(t *testing.T) {
	yml := []byte(`
pipeline: test
description: test
nodes:
  - name: a
    element: fire
  - name: b
    element: earth
edges:
  - id: e1
    name: e1
    from: a
    to: b
    merge: squash
  - id: e2
    name: e2
    from: b
    to: _done
start: a
done: _done
`)
	findings, err := Run(yml, "test.yaml")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	found := false
	for _, f := range findings {
		if f.RuleID == "S3/invalid-merge-strategy" {
			found = true
			if !strings.Contains(f.Message, "squash") {
				t.Errorf("expected message to contain 'squash', got %q", f.Message)
			}
		}
	}
	if !found {
		t.Error("expected S3/invalid-merge-strategy finding")
	}
}

func TestRun_MissingEdgeName(t *testing.T) {
	yml := []byte(`
pipeline: test
description: test
nodes:
  - name: a
    element: fire
edges:
  - id: e1
    from: a
    to: _done
start: a
done: _done
`)
	findings, err := Run(yml, "test.yaml", WithProfile(ProfileStrict))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	found := false
	for _, f := range findings {
		if f.RuleID == "S4/missing-edge-name" {
			found = true
		}
	}
	if !found {
		t.Error("expected S4/missing-edge-name finding at strict profile")
	}
}

func TestRun_InvalidCacheTTL(t *testing.T) {
	yml := []byte(`
pipeline: test
description: test
nodes:
  - name: a
    element: fire
    cache:
      ttl: "not-a-duration"
edges:
  - id: e1
    name: e1
    from: a
    to: _done
start: a
done: _done
`)
	findings, err := Run(yml, "test.yaml")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	found := false
	for _, f := range findings {
		if f.RuleID == "S7/invalid-cache-ttl" {
			found = true
		}
	}
	if !found {
		t.Error("expected S7/invalid-cache-ttl finding")
	}
}

func TestRun_MissingDescription(t *testing.T) {
	yml := []byte(`
pipeline: test
nodes:
  - name: a
    element: fire
edges:
  - id: e1
    name: e1
    from: a
    to: _done
start: a
done: _done
`)
	findings, err := Run(yml, "test.yaml", WithProfile(ProfileStrict))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	found := false
	for _, f := range findings {
		if f.RuleID == "S8/missing-pipeline-description" {
			found = true
		}
	}
	if !found {
		t.Error("expected S8/missing-pipeline-description finding")
	}
}

func TestRun_OrphanNode(t *testing.T) {
	yml := []byte(`
pipeline: test
description: test
nodes:
  - name: start_node
    element: fire
  - name: orphan
    element: water
edges:
  - id: e1
    name: e1
    from: start_node
    to: _done
start: start_node
done: _done
`)
	findings, err := Run(yml, "test.yaml")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	found := false
	for _, f := range findings {
		if f.RuleID == "G1/orphan-node" {
			found = true
			if !strings.Contains(f.Message, "orphan") {
				t.Errorf("expected message to mention 'orphan', got %q", f.Message)
			}
		}
	}
	if !found {
		t.Error("expected G1/orphan-node finding")
	}
}

func TestRun_UnreachableDone(t *testing.T) {
	yml := []byte(`
pipeline: test
description: test
nodes:
  - name: a
    element: fire
  - name: b
    element: earth
edges:
  - id: e1
    name: e1
    from: a
    to: b
  - id: e2
    name: e2
    from: b
    to: a
start: a
done: _done
`)
	findings, err := Run(yml, "test.yaml")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	found := false
	for _, f := range findings {
		if f.RuleID == "G2/unreachable-done" {
			found = true
		}
	}
	if !found {
		t.Error("expected G2/unreachable-done finding")
	}
}

func TestRun_PreferWhenOverCondition(t *testing.T) {
	yml := []byte(`
pipeline: test
description: test
nodes:
  - name: a
    element: fire
edges:
  - id: e1
    name: e1
    from: a
    to: _done
    condition: "output.confidence >= 0.9 && state.ready"
start: a
done: _done
`)
	findings, err := Run(yml, "test.yaml")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	found := false
	for _, f := range findings {
		if f.RuleID == "B1/prefer-when-over-condition" {
			found = true
		}
	}
	if !found {
		t.Error("expected B1/prefer-when-over-condition finding")
	}
}

func TestRun_ProfileMin_OnlyErrors(t *testing.T) {
	yml := []byte(`
pipeline: test
nodes:
  - name: a
    element: fyre
edges:
  - id: e1
    from: a
    to: _done
    condition: "always"
start: a
done: _done
`)
	findings, err := Run(yml, "test.yaml", WithProfile(ProfileMin))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	for _, f := range findings {
		if f.Severity != SeverityError {
			t.Errorf("profile=min should only return errors, got %s: %s", f.Severity, f.RuleID)
		}
	}
	if !HasErrors(findings) {
		t.Error("expected at least one error finding (invalid element)")
	}
}

func TestRun_InvalidWalkerPersona(t *testing.T) {
	yml := []byte(`
pipeline: test
description: test
nodes:
  - name: a
    element: fire
edges:
  - id: e1
    name: e1
    from: a
    to: _done
walkers:
  - name: agent1
    element: fire
    persona: "NonExistentPersona"
start: a
done: _done
`)
	findings, err := Run(yml, "test.yaml")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	found := false
	for _, f := range findings {
		if f.RuleID == "S11/invalid-walker-persona" {
			found = true
		}
	}
	if !found {
		t.Error("expected S11/invalid-walker-persona finding")
	}
}

func TestRun_FanInWithoutMerge(t *testing.T) {
	yml := []byte(`
pipeline: test
description: test
nodes:
  - name: a
    element: fire
  - name: b
    element: earth
  - name: c
    element: water
edges:
  - id: e1
    name: e1
    from: a
    to: c
  - id: e2
    name: e2
    from: b
    to: c
  - id: e3
    name: e3
    from: a
    to: b
  - id: e4
    name: e4
    from: c
    to: _done
start: a
done: _done
`)
	findings, err := Run(yml, "test.yaml")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	found := false
	for _, f := range findings {
		if f.RuleID == "G7/fan-in-without-merge" {
			found = true
		}
	}
	if !found {
		t.Error("expected G7/fan-in-without-merge finding")
	}
}

func TestLintContext_LineNumbers(t *testing.T) {
	ctx, err := NewLintContext(minimalYAML(), "test.yaml")
	if err != nil {
		t.Fatalf("NewLintContext: %v", err)
	}
	if line := ctx.NodeLine("recall"); line == 0 {
		t.Error("expected non-zero line for node 'recall'")
	}
	if line := ctx.EdgeLine("e1"); line == 0 {
		t.Error("expected non-zero line for edge 'e1'")
	}
	if line := ctx.TopLevelLine("pipeline"); line == 0 {
		t.Error("expected non-zero line for top-level 'pipeline'")
	}
}

func TestNewLintContextFromDef(t *testing.T) {
	def := &framework.PipelineDef{
		Pipeline: "test",
		Nodes:    []framework.NodeDef{{Name: "a", Element: "fire"}},
		Edges:    []framework.EdgeDef{{ID: "e1", Name: "e1", From: "a", To: "_done"}},
		Start:    "a",
		Done:     "_done",
	}
	ctx := NewLintContextFromDef(def, "inline")
	runner := DefaultRunner()
	findings := runner.Run(ctx, WithProfile(ProfileStrict))
	// Should not crash; line numbers will be 0
	for _, f := range findings {
		if f.RuleID == "S8/missing-pipeline-description" && f.Line != 0 {
			t.Error("expected line=0 for def-only context")
		}
	}
}

func TestFinding_String(t *testing.T) {
	f := Finding{
		RuleID:   "S2/invalid-element",
		Severity: SeverityError,
		Message:  `unknown element "fyre"`,
		File:     "pipeline.yaml",
		Line:     12,
	}
	s := f.String()
	if !strings.Contains(s, "pipeline.yaml:12") {
		t.Errorf("expected file:line, got %q", s)
	}
	if !strings.Contains(s, "error") {
		t.Errorf("expected severity, got %q", s)
	}
}

func TestAllRules_Count(t *testing.T) {
	rules := AllRules()
	// 13 structural + 7 semantic + 6 best-practice = 26
	if len(rules) != 26 {
		t.Errorf("expected 26 rules, got %d", len(rules))
	}

	ids := make(map[string]bool)
	for _, r := range rules {
		if ids[r.ID()] {
			t.Errorf("duplicate rule ID: %s", r.ID())
		}
		ids[r.ID()] = true
	}
}

func TestHasErrors(t *testing.T) {
	if HasErrors(nil) {
		t.Error("nil should not have errors")
	}
	if HasErrors([]Finding{{Severity: SeverityWarning}}) {
		t.Error("warnings should not count as errors")
	}
	if !HasErrors([]Finding{{Severity: SeverityError}}) {
		t.Error("errors should be detected")
	}
}

func TestApplyFixes_InvalidElement(t *testing.T) {
	yml := []byte(`pipeline: test
description: test
nodes:
  - name: a
    element: fyre
edges:
  - id: e1
    name: e1
    from: a
    to: _done
start: a
done: _done
`)
	fixed, fixes, err := ApplyFixes(yml, "test.yaml", WithProfile(ProfileStrict))
	if err != nil {
		t.Fatalf("ApplyFixes: %v", err)
	}
	if len(fixes) == 0 {
		t.Fatal("expected at least one fix")
	}
	if !strings.Contains(string(fixed), "element: fire") {
		t.Errorf("expected fix to replace 'fyre' with 'fire', got:\n%s", string(fixed))
	}
}

func TestApplyFixes_ConditionToWhen(t *testing.T) {
	yml := []byte(`pipeline: test
description: test
nodes:
  - name: a
    element: fire
edges:
  - id: e1
    name: e1
    from: a
    to: _done
    condition: "output.confidence >= 0.9 && state.ready"
start: a
done: _done
`)
	fixed, fixes, err := ApplyFixes(yml, "test.yaml", WithProfile(ProfileStrict))
	if err != nil {
		t.Fatalf("ApplyFixes: %v", err)
	}
	if len(fixes) == 0 {
		t.Fatal("expected at least one fix")
	}
	if !strings.Contains(string(fixed), "when:") {
		t.Errorf("expected 'condition:' to be renamed to 'when:', got:\n%s", string(fixed))
	}
	if strings.Contains(string(fixed), "condition:") {
		t.Errorf("expected 'condition:' to be removed, got:\n%s", string(fixed))
	}
}

func TestApplyFixes_NoFixNeeded(t *testing.T) {
	fixed, fixes, err := ApplyFixes(minimalYAML(), "test.yaml", WithProfile(ProfileStrict))
	if err != nil {
		t.Fatalf("ApplyFixes: %v", err)
	}
	if len(fixes) != 0 {
		for _, f := range fixes {
			t.Logf("  fix: %s at line %d: %s", f.Finding.RuleID, f.StartLine, f.Finding.Message)
		}
		t.Errorf("expected 0 fixes on clean YAML, got %d", len(fixes))
	}
	if fixed != nil {
		t.Error("expected nil bytes when no fixes applied")
	}
}

func TestLevenshtein(t *testing.T) {
	tests := []struct{ a, b string; want int }{
		{"fire", "fire", 0},
		{"fyre", "fire", 1},
		{"", "abc", 3},
		{"abc", "", 3},
		{"kitten", "sitting", 3},
	}
	for _, tt := range tests {
		if got := levenshtein(tt.a, tt.b); got != tt.want {
			t.Errorf("levenshtein(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}
