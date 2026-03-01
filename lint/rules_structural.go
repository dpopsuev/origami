package lint

import (
	"fmt"
	"strings"
	"time"

	framework "github.com/dpopsuev/origami"
)

var validElements = map[string]bool{
	"fire": true, "lightning": true, "earth": true,
	"diamond": true, "water": true, "air": true, "iron": true,
}

var validMergeStrategies = map[string]bool{
	framework.MergeAppend: true,
	framework.MergeLatest: true,
	framework.MergeCustom: true,
}

func knownPersonas() map[string]bool {
	m := make(map[string]bool)
	for _, p := range framework.AllPersonas() {
		m[strings.ToLower(p.Identity.PersonaName)] = true
	}
	return m
}

func elementSuggestion(val string) string {
	best, bestDist := "", 100
	for e := range validElements {
		d := levenshtein(strings.ToLower(val), e)
		if d < bestDist {
			bestDist = d
			best = e
		}
	}
	if bestDist <= 3 {
		return fmt.Sprintf("did you mean %q?", best)
	}
	return ""
}

// --- S1: missing-node-element ---

type MissingNodeElement struct{}

func (r *MissingNodeElement) ID() string          { return "S1/missing-node-element" }
func (r *MissingNodeElement) Description() string { return "every node should declare an element" }
func (r *MissingNodeElement) Severity() Severity   { return SeverityWarning }
func (r *MissingNodeElement) Tags() []string       { return []string{"structural"} }

func (r *MissingNodeElement) Check(ctx *LintContext) []Finding {
	var out []Finding
	for _, nd := range ctx.Def.Nodes {
		if nd.Element == "" {
			out = append(out, Finding{
				RuleID:   r.ID(),
				Severity: r.Severity(),
				Message:  fmt.Sprintf("node %q has no element", nd.Name),
				File:     ctx.File,
				Line:     ctx.NodeLine(nd.Name),
			})
		}
	}
	return out
}

// --- S2: invalid-element ---

type InvalidElement struct{}

func (r *InvalidElement) ID() string          { return "S2/invalid-element" }
func (r *InvalidElement) Description() string { return "element value must be a known element" }
func (r *InvalidElement) Severity() Severity   { return SeverityError }
func (r *InvalidElement) Tags() []string       { return []string{"structural"} }

func (r *InvalidElement) Check(ctx *LintContext) []Finding {
	var out []Finding
	for _, nd := range ctx.Def.Nodes {
		if nd.Element != "" && !validElements[strings.ToLower(nd.Element)] {
			f := Finding{
				RuleID:   r.ID(),
				Severity: r.Severity(),
				Message:  fmt.Sprintf("node %q: unknown element %q (valid: fire, water, earth, air, diamond, lightning, iron)", nd.Name, nd.Element),
				File:     ctx.File,
				Line:     ctx.NodeLine(nd.Name),
			}
			if s := elementSuggestion(nd.Element); s != "" {
				f.Suggestion = s
				f.FixAvailable = true
			}
			out = append(out, f)
		}
	}
	return out
}

// --- S3: invalid-merge-strategy ---

type InvalidMergeStrategy struct{}

func (r *InvalidMergeStrategy) ID() string          { return "S3/invalid-merge-strategy" }
func (r *InvalidMergeStrategy) Description() string { return "merge strategy must be append, latest, or custom" }
func (r *InvalidMergeStrategy) Severity() Severity   { return SeverityError }
func (r *InvalidMergeStrategy) Tags() []string       { return []string{"structural"} }

func (r *InvalidMergeStrategy) Check(ctx *LintContext) []Finding {
	var out []Finding
	for _, ed := range ctx.Def.Edges {
		if ed.Merge != "" && !validMergeStrategies[ed.Merge] {
			out = append(out, Finding{
				RuleID:   r.ID(),
				Severity: r.Severity(),
				Message:  fmt.Sprintf("edge %q: unknown merge strategy %q (valid: append, latest, custom)", ed.ID, ed.Merge),
				File:     ctx.File,
				Line:     ctx.EdgeLine(ed.ID),
			})
		}
	}
	return out
}

// --- S4: missing-edge-name ---

type MissingEdgeName struct{}

func (r *MissingEdgeName) ID() string          { return "S4/missing-edge-name" }
func (r *MissingEdgeName) Description() string { return "edges should have a human-readable name" }
func (r *MissingEdgeName) Severity() Severity   { return SeverityInfo }
func (r *MissingEdgeName) Tags() []string       { return []string{"structural"} }

func (r *MissingEdgeName) Check(ctx *LintContext) []Finding {
	var out []Finding
	for _, ed := range ctx.Def.Edges {
		if ed.Name == "" {
			out = append(out, Finding{
				RuleID:       r.ID(),
				Severity:     r.Severity(),
				Message:      fmt.Sprintf("edge %q has no name", ed.ID),
				File:         ctx.File,
				Line:         ctx.EdgeLine(ed.ID),
				FixAvailable: true,
			})
		}
	}
	return out
}

// --- S5: duplicate-edge-condition ---

type DuplicateEdgeCondition struct{}

func (r *DuplicateEdgeCondition) ID() string        { return "S5/duplicate-edge-condition" }
func (r *DuplicateEdgeCondition) Description() string { return "edges from the same node should not have identical when expressions" }
func (r *DuplicateEdgeCondition) Severity() Severity { return SeverityWarning }
func (r *DuplicateEdgeCondition) Tags() []string     { return []string{"structural"} }

func (r *DuplicateEdgeCondition) Check(ctx *LintContext) []Finding {
	type key struct{ from, when string }
	seen := make(map[key]string)
	var out []Finding
	for _, ed := range ctx.Def.Edges {
		if ed.When == "" || ed.Parallel {
			continue
		}
		k := key{ed.From, ed.When}
		if prev, ok := seen[k]; ok {
			out = append(out, Finding{
				RuleID:   r.ID(),
				Severity: r.Severity(),
				Message:  fmt.Sprintf("edge %q has same when expression as edge %q (from node %q)", ed.ID, prev, ed.From),
				File:     ctx.File,
				Line:     ctx.EdgeLine(ed.ID),
			})
		} else {
			seen[k] = ed.ID
		}
	}
	return out
}

// --- S6: empty-prompt ---

type EmptyPrompt struct{}

func (r *EmptyPrompt) ID() string          { return "S6/empty-prompt" }
func (r *EmptyPrompt) Description() string { return "node with no prompt, transformer, extractor, renderer, or marble may produce empty output" }
func (r *EmptyPrompt) Severity() Severity   { return SeverityWarning }
func (r *EmptyPrompt) Tags() []string       { return []string{"structural"} }

func (r *EmptyPrompt) Check(ctx *LintContext) []Finding {
	var out []Finding
	for _, nd := range ctx.Def.Nodes {
		// family-based nodes are resolved by NodeRegistry — the Go implementation
		// provides its own prompting logic, so missing prompt/transformer is fine.
		if nd.Family != "" {
			continue
		}
		if nd.Prompt == "" && nd.Transformer == "" && nd.Extractor == "" && nd.Renderer == "" && nd.Marble == "" {
			out = append(out, Finding{
				RuleID:   r.ID(),
				Severity: r.Severity(),
				Message:  fmt.Sprintf("node %q has no prompt, transformer, extractor, or marble", nd.Name),
				File:     ctx.File,
				Line:     ctx.NodeLine(nd.Name),
			})
		}
	}
	return out
}

// --- S7: invalid-cache-ttl ---

type InvalidCacheTTL struct{}

func (r *InvalidCacheTTL) ID() string          { return "S7/invalid-cache-ttl" }
func (r *InvalidCacheTTL) Description() string { return "cache TTL must be a valid Go duration" }
func (r *InvalidCacheTTL) Severity() Severity   { return SeverityError }
func (r *InvalidCacheTTL) Tags() []string       { return []string{"structural"} }

func (r *InvalidCacheTTL) Check(ctx *LintContext) []Finding {
	var out []Finding
	for _, nd := range ctx.Def.Nodes {
		if nd.Cache != nil && nd.Cache.TTL != "" {
			if _, err := time.ParseDuration(nd.Cache.TTL); err != nil {
				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.Severity(),
					Message:  fmt.Sprintf("node %q: invalid cache TTL %q: %v", nd.Name, nd.Cache.TTL, err),
					File:     ctx.File,
					Line:     ctx.NodeLine(nd.Name),
				})
			}
		}
	}
	return out
}

// --- S8: missing-pipeline-description ---

type MissingPipelineDescription struct{}

func (r *MissingPipelineDescription) ID() string        { return "S8/missing-pipeline-description" }
func (r *MissingPipelineDescription) Description() string { return "pipeline should have a description" }
func (r *MissingPipelineDescription) Severity() Severity { return SeverityInfo }
func (r *MissingPipelineDescription) Tags() []string     { return []string{"structural"} }

func (r *MissingPipelineDescription) Check(ctx *LintContext) []Finding {
	if ctx.Def.Description == "" {
		return []Finding{{
			RuleID:       r.ID(),
			Severity:     r.Severity(),
			Message:      "pipeline has no description",
			File:         ctx.File,
			Line:         ctx.TopLevelLine("pipeline"),
			FixAvailable: true,
		}}
	}
	return nil
}

// --- S9: unnamed-node ---

type UnnamedNode struct{}

func (r *UnnamedNode) ID() string          { return "S9/unnamed-node" }
func (r *UnnamedNode) Description() string { return "every node must have a name" }
func (r *UnnamedNode) Severity() Severity   { return SeverityError }
func (r *UnnamedNode) Tags() []string       { return []string{"structural"} }

func (r *UnnamedNode) Check(ctx *LintContext) []Finding {
	var out []Finding
	for i, nd := range ctx.Def.Nodes {
		if nd.Name == "" {
			out = append(out, Finding{
				RuleID:   r.ID(),
				Severity: r.Severity(),
				Message:  fmt.Sprintf("node at index %d has no name", i),
				File:     ctx.File,
			})
		}
	}
	return out
}

// --- S10: invalid-walker-element ---

type InvalidWalkerElement struct{}

func (r *InvalidWalkerElement) ID() string          { return "S10/invalid-walker-element" }
func (r *InvalidWalkerElement) Description() string { return "walker element must be a known element" }
func (r *InvalidWalkerElement) Severity() Severity   { return SeverityError }
func (r *InvalidWalkerElement) Tags() []string       { return []string{"structural"} }

func (r *InvalidWalkerElement) Check(ctx *LintContext) []Finding {
	var out []Finding
	for _, w := range ctx.Def.Walkers {
		if w.Element != "" && !validElements[strings.ToLower(w.Element)] {
			f := Finding{
				RuleID:   r.ID(),
				Severity: r.Severity(),
				Message:  fmt.Sprintf("walker %q: unknown element %q", w.Name, w.Element),
				File:     ctx.File,
				Line:     ctx.WalkerLine(w.Name),
			}
			if s := elementSuggestion(w.Element); s != "" {
				f.Suggestion = s
				f.FixAvailable = true
			}
			out = append(out, f)
		}
	}
	return out
}

// --- S11: invalid-walker-persona ---

type InvalidWalkerPersona struct{}

func (r *InvalidWalkerPersona) ID() string          { return "S11/invalid-walker-persona" }
func (r *InvalidWalkerPersona) Description() string { return "walker persona must be a known persona" }
func (r *InvalidWalkerPersona) Severity() Severity   { return SeverityError }
func (r *InvalidWalkerPersona) Tags() []string       { return []string{"structural"} }

func (r *InvalidWalkerPersona) Check(ctx *LintContext) []Finding {
	personas := knownPersonas()
	var out []Finding
	for _, w := range ctx.Def.Walkers {
		if w.Persona != "" && !personas[strings.ToLower(w.Persona)] {
			out = append(out, Finding{
				RuleID:   r.ID(),
				Severity: r.Severity(),
				Message:  fmt.Sprintf("walker %q: unknown persona %q", w.Name, w.Persona),
				File:     ctx.File,
				Line:     ctx.WalkerLine(w.Name),
			})
		}
	}
	return out
}

// Valid zone domain values.
var validDomains = map[string]bool{
	"unstructured": true,
	"structured":   true,
	"hybrid":       true,
}

// --- S12: schema-in-unstructured-zone ---

type SchemaInUnstructuredZone struct{}

func (r *SchemaInUnstructuredZone) ID() string          { return "S12/schema-in-unstructured-zone" }
func (r *SchemaInUnstructuredZone) Description() string { return "nodes with schema should not be in unstructured zones" }
func (r *SchemaInUnstructuredZone) Severity() Severity   { return SeverityWarning }
func (r *SchemaInUnstructuredZone) Tags() []string       { return []string{"structural"} }

func (r *SchemaInUnstructuredZone) Check(ctx *LintContext) []Finding {
	var out []Finding
	for zoneName, zd := range ctx.Def.Zones {
		if strings.ToLower(zd.Domain) != "unstructured" {
			continue
		}
		nodeSet := make(map[string]bool, len(zd.Nodes))
		for _, n := range zd.Nodes {
			nodeSet[n] = true
		}
		for _, nd := range ctx.Def.Nodes {
			if nodeSet[nd.Name] && nd.Schema != nil {
				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.Severity(),
					Message:  fmt.Sprintf("node %q has schema but is in unstructured zone %q", nd.Name, zoneName),
					File:     ctx.File,
					Line:     ctx.NodeLine(nd.Name),
				})
			}
		}
	}
	return out
}

// --- S13: invalid-zone-domain ---

type InvalidZoneDomain struct{}

func (r *InvalidZoneDomain) ID() string          { return "S13/invalid-zone-domain" }
func (r *InvalidZoneDomain) Description() string { return "zone domain must be unstructured, structured, or hybrid" }
func (r *InvalidZoneDomain) Severity() Severity   { return SeverityError }
func (r *InvalidZoneDomain) Tags() []string       { return []string{"structural"} }

func (r *InvalidZoneDomain) Check(ctx *LintContext) []Finding {
	var out []Finding
	for zoneName, zd := range ctx.Def.Zones {
		if zd.Domain != "" && !validDomains[strings.ToLower(zd.Domain)] {
			out = append(out, Finding{
				RuleID:   r.ID(),
				Severity: r.Severity(),
				Message:  fmt.Sprintf("zone %q: unknown domain %q (valid: unstructured, structured, hybrid)", zoneName, zd.Domain),
				File:     ctx.File,
				Line:     ctx.TopLevelLine("zones"),
			})
		}
	}
	return out
}

// levenshtein computes the edit distance between two strings.
func levenshtein(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}
	prev := make([]int, lb+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= la; i++ {
		curr := make([]int, lb+1)
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min(curr[j-1]+1, min(prev[j]+1, prev[j-1]+cost))
		}
		prev = curr
	}
	return prev[lb]
}
