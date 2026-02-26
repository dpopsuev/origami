package knowledge

// RouteRequest describes what the pipeline is looking for — a component under
// investigation, a hypothesis being tested, the current step, and any
// additional tag constraints.
type RouteRequest struct {
	Component  string
	Hypothesis string
	Step       string
	Tags       map[string]string
}

// RouteRule decides whether a source is relevant for a given request.
type RouteRule interface {
	Match(source Source, req RouteRequest) bool
}

// KnowledgeSourceRouter selects sources from a catalog based on configurable
// rules. If no rule matches any source, all sources are returned (safe default).
type KnowledgeSourceRouter struct {
	catalog *KnowledgeSourceCatalog
	rules   []RouteRule
}

// NewRouter creates a router over the given catalog with the provided rules.
func NewRouter(catalog *KnowledgeSourceCatalog, rules ...RouteRule) *KnowledgeSourceRouter {
	return &KnowledgeSourceRouter{catalog: catalog, rules: rules}
}

// Route returns sources where at least one rule matches. Sources with
// ReadPolicy == ReadAlways are always included regardless of rule matching.
// If no rules are configured or no conditional rule matches any source,
// all sources are returned.
func (r *KnowledgeSourceRouter) Route(req RouteRequest) []Source {
	if len(r.rules) == 0 || r.catalog == nil {
		return r.allSources()
	}

	seen := make(map[string]bool, len(r.catalog.Sources))
	matched := make([]Source, 0, len(r.catalog.Sources))

	for _, src := range r.catalog.Sources {
		if src.IsAlwaysRead() {
			seen[src.Name] = true
			matched = append(matched, src)
			continue
		}
		for _, rule := range r.rules {
			if rule.Match(src, req) {
				if !seen[src.Name] {
					seen[src.Name] = true
					matched = append(matched, src)
				}
				break
			}
		}
	}

	alwaysCount := 0
	for _, src := range matched {
		if src.IsAlwaysRead() {
			alwaysCount++
		}
	}

	if len(matched) == alwaysCount {
		return r.allSources()
	}
	return matched
}

func (r *KnowledgeSourceRouter) allSources() []Source {
	if r.catalog == nil {
		return nil
	}
	out := make([]Source, len(r.catalog.Sources))
	copy(out, r.catalog.Sources)
	return out
}

// TagMatchRule matches sources whose Tags contain all the key-value pairs
// specified in the rule's Required map. This is the batteries-included rule
// for tag-based source selection.
type TagMatchRule struct {
	Required map[string]string
}

// Match returns true if source.Tags[k] == v for every (k, v) in Required.
func (r TagMatchRule) Match(src Source, _ RouteRequest) bool {
	for k, v := range r.Required {
		if src.Tags[k] != v {
			return false
		}
	}
	return true
}

// RequestTagMatchRule matches sources whose Tags overlap with the request's
// Tags. For every key present in both source.Tags and req.Tags, the values
// must be equal.
type RequestTagMatchRule struct{}

// Match returns true if all overlapping tag keys have equal values.
// Sources with no tags always match (no constraints to violate).
func (RequestTagMatchRule) Match(src Source, req RouteRequest) bool {
	if len(src.Tags) == 0 || len(req.Tags) == 0 {
		return true
	}
	for k, rv := range req.Tags {
		if sv, ok := src.Tags[k]; ok && sv != rv {
			return false
		}
	}
	return true
}
