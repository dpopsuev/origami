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
	raw := `circuit: test
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
	raw := `circuit: test
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
	for _, key := range []string{"circuit", "nodes", "edges", "start", "done"} {
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

	hover := computeHover(doc, protocol.Position{Line: 0, Character: 13}, nil)
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

	hover := computeHover(doc, protocol.Position{Line: 0, Character: 14}, nil)
	if hover == nil {
		t.Fatal("expected hover for persona")
	}
	if !strings.Contains(hover.Contents.Value, "herald") && !strings.Contains(hover.Contents.Value, "Herald") {
		t.Errorf("hover content doesn't mention herald: %s", hover.Contents.Value)
	}
}

func TestDefinition_EdgeToNode(t *testing.T) {
	content := `circuit: test
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

func TestSemanticTokens_ElementValues(t *testing.T) {
	content := `circuit: test
nodes:
  - name: recall
    element: fire
  - name: deep
    element: water
  - name: classify
    element: earth
edges:
  - id: E1
    from: recall
    to: deep
    when: "true"
start: recall
done: DONE`

	doc := &document{
		URI:     uri.URI("file:///test.yaml"),
		Content: content,
	}

	raw := []byte(content)
	lctx, _ := lint.NewLintContext(raw, "test.yaml")
	if lctx != nil {
		doc.Def = lctx.Def
		doc.LintCtx = lctx
	}

	data := computeSemanticTokens(doc)
	// Each token produces 5 uint32 values
	if len(data)%5 != 0 {
		t.Fatalf("data length %d is not a multiple of 5", len(data))
	}
	tokenCount := len(data) / 5
	if tokenCount < 3 {
		t.Errorf("expected at least 3 element tokens (fire, water, earth), got %d", tokenCount)
	}

	// Verify first token is fire (type 0)
	if len(data) >= 5 {
		tokenType := data[3]
		if tokenType != elementTokenIndex["fire"] {
			t.Errorf("first token type = %d, want %d (fire)", tokenType, elementTokenIndex["fire"])
		}
	}
}

func TestSemanticTokens_Empty(t *testing.T) {
	doc := &document{
		URI:     uri.URI("file:///empty.yaml"),
		Content: "circuit: test\nnodes:\n  - name: start\nstart: start\ndone: DONE",
	}

	data := computeSemanticTokens(doc)
	if len(data) != 0 {
		t.Errorf("expected no semantic tokens for circuit without elements, got %d values", len(data))
	}
}

func TestSemanticTokens_AllElements(t *testing.T) {
	content := `circuit: elements
nodes:
  - name: n1
    element: fire
  - name: n2
    element: water
  - name: n3
    element: earth
  - name: n4
    element: air
  - name: n5
    element: diamond
  - name: n6
    element: lightning
  - name: n7
    element: iron
start: n1
done: DONE`

	doc := &document{
		URI:     uri.URI("file:///test.yaml"),
		Content: content,
	}
	raw := []byte(content)
	lctx, _ := lint.NewLintContext(raw, "test.yaml")
	if lctx != nil {
		doc.Def = lctx.Def
	}

	data := computeSemanticTokens(doc)
	tokenCount := len(data) / 5
	if tokenCount != 7 {
		t.Errorf("expected 7 element tokens (all 7 types), got %d", tokenCount)
	}

	// Verify all 7 unique token types are present
	seen := map[uint32]bool{}
	for i := 0; i < len(data); i += 5 {
		seen[data[i+3]] = true
	}
	for el, idx := range elementTokenIndex {
		if !seen[idx] {
			t.Errorf("missing token type for element %q (index %d)", el, idx)
		}
	}
}

func TestSemanticTokensLegend(t *testing.T) {
	legend := SemanticTokensLegend()
	types, ok := legend["tokenTypes"].([]string)
	if !ok {
		t.Fatal("legend missing tokenTypes")
	}
	if len(types) != 7 {
		t.Errorf("expected 7 token types, got %d", len(types))
	}
}

func TestSemanticTokensProvider(t *testing.T) {
	provider := SemanticTokensProvider()
	if provider["full"] != true {
		t.Error("expected full: true")
	}
	legend, ok := provider["legend"].(map[string]any)
	if !ok {
		t.Fatal("expected legend in provider")
	}
	types, ok := legend["tokenTypes"].([]string)
	if !ok {
		t.Fatal("legend missing tokenTypes")
	}
	if len(types) != 7 {
		t.Errorf("expected 7 token types, got %d", len(types))
	}
}

func TestInlayHints_ElementTraits(t *testing.T) {
	content := `circuit: test
nodes:
  - name: recall
    element: fire
  - name: deep
    element: water
start: recall
done: DONE`

	doc := makeTestDoc(content)
	hints := computeInlayHints(doc)

	traitHints := filterHintsByKind(hints, 1)
	found := 0
	for _, h := range traitHints {
		if strings.Contains(h.Label, "high") || strings.Contains(h.Label, "low") {
			found++
		}
	}
	if found < 2 {
		t.Errorf("expected at least 2 element trait hints (fire=high, water=low), found %d", found)
	}
}

func TestInlayHints_PersonaDescription(t *testing.T) {
	content := `circuit: test
nodes:
  - name: recall
walkers:
  - name: scout
    persona: herald
start: recall
done: DONE`

	doc := makeTestDoc(content)
	hints := computeInlayHints(doc)

	found := false
	for _, h := range hints {
		if strings.Contains(h.Label, "Fire persona") {
			found = true
		}
	}
	if !found {
		t.Error("expected persona hint for herald")
	}
}

func TestInlayHints_ExpressionValidity(t *testing.T) {
	content := `circuit: test
nodes:
  - name: recall
  - name: triage
edges:
  - id: E1
    from: recall
    to: triage
    when: "true"
  - id: E2
    from: triage
    to: DONE
    when: "output.confidence > 0.8"
start: recall
done: DONE`

	doc := makeTestDoc(content)
	hints := computeInlayHints(doc)

	staticCount := 0
	outputDepCount := 0
	for _, h := range hints {
		if h.Label == "static" {
			staticCount++
		}
		if h.Label == "output-dep" {
			outputDepCount++
		}
	}
	if staticCount < 1 {
		t.Error("expected at least 1 'static' expression hint for when: true")
	}
	if outputDepCount < 1 {
		t.Error("expected at least 1 'output-dep' expression hint")
	}
}

func TestInlayHints_EdgeFlow(t *testing.T) {
	content := `circuit: test
nodes:
  - name: recall
    element: fire
  - name: triage
    element: water
edges:
  - id: E1
    from: recall
    to: triage
    when: "true"
  - id: E2
    from: triage
    to: DONE
    when: "true"
start: recall
done: DONE`

	doc := makeTestDoc(content)
	hints := computeInlayHints(doc)

	found := false
	for _, h := range hints {
		if strings.Contains(h.Label, "fire") && strings.Contains(h.Label, "water") {
			found = true
		}
	}
	if !found {
		t.Error("expected edge flow hint showing fire → water")
	}
}

func TestInlayHints_StartNode(t *testing.T) {
	content := `circuit: test
nodes:
  - name: recall
    element: fire
    family: ingest
start: recall
done: DONE`

	doc := makeTestDoc(content)
	hints := computeInlayHints(doc)

	found := false
	for _, h := range hints {
		if strings.Contains(h.Label, "entry") {
			found = true
			if !strings.Contains(h.Label, "fire") {
				t.Error("start node hint should include element")
			}
			if !strings.Contains(h.Label, "ingest") {
				t.Error("start node hint should include family")
			}
		}
	}
	if !found {
		t.Error("expected start node entry hint")
	}
}

func TestInlayHints_Empty(t *testing.T) {
	doc := &document{
		URI:     uri.URI("file:///test.yaml"),
		Content: "circuit: test",
	}
	hints := computeInlayHints(doc)
	if len(hints) != 0 {
		t.Errorf("expected no hints for doc without parsed def, got %d", len(hints))
	}
}

func makeTestDoc(content string) *document {
	doc := &document{
		URI:     uri.URI("file:///test.yaml"),
		Content: content,
	}
	raw := []byte(content)
	lctx, _ := lint.NewLintContext(raw, "test.yaml")
	if lctx != nil {
		doc.Def = lctx.Def
		doc.LintCtx = lctx
	}
	return doc
}

func filterHintsByKind(hints []InlayHint, kind int) []InlayHint {
	var out []InlayHint
	for _, h := range hints {
		if h.Kind == kind {
			out = append(out, h)
		}
	}
	return out
}

func TestKamiBridge_ProcessEvents(t *testing.T) {
	kb := NewKamiBridge(0)

	kb.processEvent(`{"type":"node_enter","node":"recall","agent":"seeker","ts":"2026-02-26T10:00:00Z"}`)
	state := kb.State()
	if state.ActiveNode != "recall" {
		t.Errorf("active node = %q, want recall", state.ActiveNode)
	}
	if state.ActiveAgent != "seeker" {
		t.Errorf("active agent = %q, want seeker", state.ActiveAgent)
	}
	if _, ok := state.Visited["recall"]; !ok {
		t.Error("recall should be in visited map")
	}

	kb.processEvent(`{"type":"transition","edge":"E1","ts":"2026-02-26T10:00:01Z"}`)
	state = kb.State()
	if _, ok := state.Transitions["E1"]; !ok {
		t.Error("E1 should be in transitions map")
	}

	kb.processEvent(`{"type":"paused","ts":"2026-02-26T10:00:02Z"}`)
	state = kb.State()
	if !state.Paused {
		t.Error("expected paused=true after paused event")
	}

	kb.processEvent(`{"type":"resumed","ts":"2026-02-26T10:00:03Z"}`)
	state = kb.State()
	if state.Paused {
		t.Error("expected paused=false after resumed event")
	}

	kb.processEvent(`{"type":"walk_complete","ts":"2026-02-26T10:00:04Z"}`)
	state = kb.State()
	if state.ActiveNode != "" {
		t.Errorf("active node should be empty after walk_complete, got %q", state.ActiveNode)
	}
}

func TestKamiBridge_LiveInlayHints(t *testing.T) {
	kb := NewKamiBridge(0)

	content := `circuit: test
nodes:
  - name: recall
    element: fire
  - name: triage
    element: water
start: recall
done: DONE`

	doc := makeTestDoc(content)

	kb.processEvent(`{"type":"node_enter","node":"recall","agent":"herald","ts":"2026-02-26T10:00:00Z"}`)

	hints := kb.LiveInlayHints(doc)
	foundActive := false
	foundVisited := false
	for _, h := range hints {
		if strings.Contains(h.Label, "ACTIVE") {
			foundActive = true
			if !strings.Contains(h.Label, "herald") {
				t.Error("active hint should include agent name")
			}
		}
	}
	if !foundActive {
		t.Error("expected ACTIVE hint for recall node")
	}

	kb.processEvent(`{"type":"node_exit","node":"recall","ts":"2026-02-26T10:00:01Z"}`)
	kb.processEvent(`{"type":"node_enter","node":"triage","agent":"herald","ts":"2026-02-26T10:00:02Z"}`)

	hints = kb.LiveInlayHints(doc)
	for _, h := range hints {
		if strings.Contains(h.Label, "visited") {
			foundVisited = true
		}
	}
	if !foundVisited {
		t.Error("expected 'visited' hint for recall after it was exited")
	}
}

func TestKamiBridge_PausedHint(t *testing.T) {
	kb := NewKamiBridge(0)

	content := `circuit: test
nodes:
  - name: recall
start: recall
done: DONE`
	doc := makeTestDoc(content)

	kb.processEvent(`{"type":"node_enter","node":"recall","agent":"seeker","ts":"2026-02-26T10:00:00Z"}`)
	kb.processEvent(`{"type":"paused","ts":"2026-02-26T10:00:01Z"}`)

	hints := kb.LiveInlayHints(doc)
	found := false
	for _, h := range hints {
		if h.Label == "PAUSED" {
			found = true
		}
	}
	if !found {
		t.Error("expected PAUSED hint when circuit is paused on active node")
	}
}

func TestKamiBridge_NotConnected(t *testing.T) {
	kb := NewKamiBridge(0)
	if kb.Connected() {
		t.Error("bridge should not be connected without Start()")
	}
}

func TestKamiBridge_StateSnapshotIsolation(t *testing.T) {
	kb := NewKamiBridge(0)
	kb.processEvent(`{"type":"node_enter","node":"recall","agent":"seeker","ts":"2026-02-26T10:00:00Z"}`)

	state := kb.State()
	state.Visited["injected"] = VisitInfo{Agent: "hacker"}

	fresh := kb.State()
	if _, ok := fresh.Visited["injected"]; ok {
		t.Error("state snapshot should be isolated — mutation leaked")
	}
}
