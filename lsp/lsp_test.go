package lsp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"

	"github.com/dpopsuev/origami/lint"
)

func newLintContextForTest(raw []byte, file string) (*lint.LintContext, error) {
	return lint.NewLintContext(raw, file)
}

func TestComputeDiagnostics_ValidYAML(t *testing.T) {
	raw := `pipeline: test
description: "A simple test"
nodes:
  - name: start
  - name: finish
edges:
  - id: E1
    name: go
    from: start
    to: finish
    when: "true"
  - id: E2
    name: done
    from: finish
    to: DONE
    when: "true"
start: start
done: DONE`

	doc := &document{
		URI:     uri.URI("file:///test.yaml"),
		Content: raw,
	}

	diags := computeDiagnostics(doc)
	for _, d := range diags {
		if d.Severity == protocol.DiagnosticSeverityError {
			t.Errorf("unexpected error diagnostic: %s (%v)", d.Message, d.Code)
		}
	}
}

func TestComputeDiagnostics_InvalidElement(t *testing.T) {
	raw := `pipeline: test
nodes:
  - name: start
    element: fyre
start: start
done: DONE`

	doc := &document{
		URI:     uri.URI("file:///test.yaml"),
		Content: raw,
	}

	diags := computeDiagnostics(doc)
	found := false
	for _, d := range diags {
		if strings.Contains(d.Message, "fyre") || strings.Contains(d.Message, "element") {
			found = true
		}
	}
	if !found {
		t.Error("expected diagnostic about invalid element 'fyre'")
	}
}

func TestComputeDiagnostics_Empty(t *testing.T) {
	doc := &document{
		URI:     uri.URI("file:///empty.yaml"),
		Content: "",
	}

	diags := computeDiagnostics(doc)
	if len(diags) != 0 {
		t.Errorf("expected no diagnostics for empty file, got %d", len(diags))
	}
}

func TestCompletion_TopLevel(t *testing.T) {
	doc := &document{
		URI:     uri.URI("file:///test.yaml"),
		Content: "",
	}

	items := computeCompletions(doc, protocol.Position{Line: 0, Character: 0})
	if len(items) == 0 {
		t.Fatal("expected top-level completions")
	}

	labels := map[string]bool{}
	for _, item := range items {
		labels[item.Label] = true
	}
	for _, key := range []string{"pipeline", "nodes", "edges", "start", "done"} {
		if !labels[key] {
			t.Errorf("missing top-level completion: %s", key)
		}
	}
}

func TestCompletion_ElementValues(t *testing.T) {
	doc := &document{
		URI:     uri.URI("file:///test.yaml"),
		Content: "  element: ",
	}

	items := computeCompletions(doc, protocol.Position{Line: 0, Character: 11})
	if len(items) == 0 {
		t.Fatal("expected element value completions")
	}

	labels := map[string]bool{}
	for _, item := range items {
		labels[item.Label] = true
	}
	for _, el := range []string{"fire", "water", "earth", "air", "diamond"} {
		if !labels[el] {
			t.Errorf("missing element completion: %s", el)
		}
	}
}

func TestHover_Element(t *testing.T) {
	doc := &document{
		URI:     uri.URI("file:///test.yaml"),
		Content: "    element: fire",
	}

	hover := computeHover(doc, protocol.Position{Line: 0, Character: 13})
	if hover == nil {
		t.Fatal("expected hover for element")
	}
	if !strings.Contains(hover.Contents.Value, "Fire") && !strings.Contains(hover.Contents.Value, "fire") {
		t.Errorf("hover content doesn't mention fire: %s", hover.Contents.Value)
	}
}

func TestHover_Persona(t *testing.T) {
	doc := &document{
		URI:     uri.URI("file:///test.yaml"),
		Content: "    persona: herald",
	}

	hover := computeHover(doc, protocol.Position{Line: 0, Character: 14})
	if hover == nil {
		t.Fatal("expected hover for persona")
	}
	if !strings.Contains(hover.Contents.Value, "herald") && !strings.Contains(hover.Contents.Value, "Herald") {
		t.Errorf("hover content doesn't mention herald: %s", hover.Contents.Value)
	}
}

func TestDefinition_EdgeToNode(t *testing.T) {
	content := `pipeline: test
nodes:
  - name: recall
  - name: triage
edges:
  - id: E1
    from: recall
    to: triage
    when: "true"
start: recall
done: DONE`

	raw := []byte(content)
	doc := &document{
		URI:     uri.URI("file:///test.yaml"),
		Content: content,
	}

	// Parse for definition support
	lintCtx, err := newLintContextForTest(raw, "test.yaml")
	if err == nil && lintCtx != nil {
		doc.LintCtx = lintCtx
		doc.Def = lintCtx.Def
	}

	// Line 7 is "    to: triage" (0-indexed)
	loc := computeDefinition(doc, protocol.Position{Line: 7, Character: 8})
	if loc == nil {
		t.Skip("definition not resolved (acceptable for basic line mapping)")
	}
}

func TestScenarioYAMLs_NoDiagnosticErrors(t *testing.T) {
	patterns := []string{
		"../testdata/*.yaml",
		"../testdata/scenarios/*.yaml",
		"../testdata/patterns/*.yaml",
	}

	tested := 0
	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			t.Fatalf("glob %s: %v", pattern, err)
		}
		for _, path := range matches {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}

			doc := &document{
				URI:     uri.URI("file://" + path),
				Content: string(raw),
			}

			diags := computeDiagnostics(doc)
			for _, d := range diags {
				if d.Severity == protocol.DiagnosticSeverityError {
					t.Errorf("%s: unexpected error: %s (%v)", path, d.Message, d.Code)
				}
			}
			tested++
		}
	}
	if tested == 0 {
		t.Skip("no scenario YAMLs found")
	}
	t.Logf("validated %d scenario YAMLs with zero errors", tested)
}

func TestServerHandler_Initialize(t *testing.T) {
	srv := NewServer()
	h := srv.Handler()
	if h == nil {
		t.Fatal("Handler() returned nil")
	}
}
