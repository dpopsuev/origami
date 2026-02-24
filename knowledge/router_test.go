package knowledge

import "testing"

var testCatalog = &KnowledgeSourceCatalog{
	Sources: []Source{
		{Name: "operator", Kind: SourceKindRepo, URI: "/op", Tags: map[string]string{"component": "ptp", "role": "operator"}},
		{Name: "tests", Kind: SourceKindRepo, URI: "/tests", Tags: map[string]string{"component": "ptp", "role": "tests"}},
		{Name: "cloud-events", Kind: SourceKindRepo, URI: "/ce", Tags: map[string]string{"component": "cep", "role": "proxy"}},
		{Name: "runbook", Kind: SourceKindDoc, URI: "/docs/runbook.md"},
	},
}

func TestTagMatchRule_Match(t *testing.T) {
	rule := TagMatchRule{Required: map[string]string{"component": "ptp"}}
	router := NewRouter(testCatalog, rule)
	got := router.Route(RouteRequest{})

	if len(got) != 2 {
		t.Fatalf("want 2 ptp sources, got %d", len(got))
	}
	for _, s := range got {
		if s.Tags["component"] != "ptp" {
			t.Errorf("unexpected source: %s", s.Name)
		}
	}
}

func TestTagMatchRule_MultipleRequired(t *testing.T) {
	rule := TagMatchRule{Required: map[string]string{"component": "ptp", "role": "operator"}}
	router := NewRouter(testCatalog, rule)
	got := router.Route(RouteRequest{})

	if len(got) != 1 || got[0].Name != "operator" {
		t.Errorf("want only 'operator', got %v", got)
	}
}

func TestRouter_NoRules_ReturnsAll(t *testing.T) {
	router := NewRouter(testCatalog)
	got := router.Route(RouteRequest{})

	if len(got) != 4 {
		t.Errorf("want all 4, got %d", len(got))
	}
}

func TestRouter_NoMatch_ReturnsAll(t *testing.T) {
	rule := TagMatchRule{Required: map[string]string{"component": "nonexistent"}}
	router := NewRouter(testCatalog, rule)
	got := router.Route(RouteRequest{})

	if len(got) != 4 {
		t.Errorf("no match should return all 4, got %d", len(got))
	}
}

func TestRouter_NilCatalog(t *testing.T) {
	router := NewRouter(nil, TagMatchRule{Required: map[string]string{"x": "y"}})
	got := router.Route(RouteRequest{})

	if got != nil {
		t.Errorf("nil catalog should return nil, got %v", got)
	}
}

func TestRouter_EmptyCatalog(t *testing.T) {
	empty := &KnowledgeSourceCatalog{}
	router := NewRouter(empty, TagMatchRule{Required: map[string]string{"x": "y"}})
	got := router.Route(RouteRequest{})

	if len(got) != 0 {
		t.Errorf("empty catalog should return empty, got %d", len(got))
	}
}

func TestRouter_MultipleRules_AnyMatch(t *testing.T) {
	rule1 := TagMatchRule{Required: map[string]string{"component": "ptp", "role": "operator"}}
	rule2 := TagMatchRule{Required: map[string]string{"component": "cep"}}
	router := NewRouter(testCatalog, rule1, rule2)
	got := router.Route(RouteRequest{})

	if len(got) != 2 {
		t.Fatalf("want 2 (operator + cloud-events), got %d", len(got))
	}
	names := map[string]bool{}
	for _, s := range got {
		names[s.Name] = true
	}
	if !names["operator"] || !names["cloud-events"] {
		t.Errorf("expected operator and cloud-events, got %v", names)
	}
}

func TestRequestTagMatchRule_OverlappingTags(t *testing.T) {
	rule := RequestTagMatchRule{}
	router := NewRouter(testCatalog, rule)

	got := router.Route(RouteRequest{Tags: map[string]string{"component": "ptp"}})
	names := map[string]bool{}
	for _, s := range got {
		names[s.Name] = true
	}
	if !names["operator"] || !names["tests"] || !names["runbook"] {
		t.Errorf("expected operator, tests, runbook; got %v", names)
	}
	if names["cloud-events"] {
		t.Errorf("cloud-events has component=cep, should not match component=ptp")
	}
}

func TestRequestTagMatchRule_NoReqTags_MatchesAll(t *testing.T) {
	rule := RequestTagMatchRule{}
	router := NewRouter(testCatalog, rule)

	got := router.Route(RouteRequest{})
	if len(got) != 4 {
		t.Errorf("empty request tags should match all, got %d", len(got))
	}
}

func TestRouter_ReturnsDefensiveCopy(t *testing.T) {
	router := NewRouter(testCatalog)
	got := router.Route(RouteRequest{})
	got[0].Name = "mutated"

	if testCatalog.Sources[0].Name == "mutated" {
		t.Error("Route() should return a copy, not a reference to catalog sources")
	}
}
