package lint

import (
	"fmt"
	"strings"

	framework "github.com/dpopsuev/origami"
)

// --- B1: prefer-when-over-condition ---

type PreferWhenOverCondition struct{}

func (r *PreferWhenOverCondition) ID() string          { return "B1/prefer-when-over-condition" }
func (r *PreferWhenOverCondition) Description() string { return "use when instead of deprecated condition field" }
func (r *PreferWhenOverCondition) Severity() Severity   { return SeverityWarning }
func (r *PreferWhenOverCondition) Tags() []string       { return []string{"best-practice"} }

func (r *PreferWhenOverCondition) Check(ctx *LintContext) []Finding {
	var out []Finding
	for _, ed := range ctx.Def.Edges {
		if ed.Condition != "" && ed.When == "" && looksLikeExpression(ed.Condition) {
			out = append(out, Finding{
				RuleID:       r.ID(),
				Severity:     r.Severity(),
				Message:      fmt.Sprintf("edge %q: condition %q looks like an expression; use when for evaluated conditions", ed.ID, ed.Condition),
				File:         ctx.File,
				Line:         ctx.EdgeLine(ed.ID),
				FixAvailable: true,
			})
		}
	}
	return out
}

// looksLikeExpression returns true if the string contains operators or
// patterns typical of expr-lang expressions rather than human comments.
// looksLikeExpression returns true only when the string contains tokens
// that are clearly programmatic (expr-lang references or boolean operators).
// Comparison operators like ==, >= are excluded because they commonly appear
// in human-readable descriptions ("confidence >= 0.9", "verdict == affirm").
func looksLikeExpression(s string) bool {
	for _, op := range []string{"&&", "||", "output.", "state.", "config."} {
		if strings.Contains(s, op) {
			return true
		}
	}
	return false
}

// --- B2: name-your-edges ---

type NameYourEdges struct{}

func (r *NameYourEdges) ID() string          { return "B2/name-your-edges" }
func (r *NameYourEdges) Description() string { return "circuits with many edges should name them for readability" }
func (r *NameYourEdges) Severity() Severity   { return SeverityInfo }
func (r *NameYourEdges) Tags() []string       { return []string{"best-practice"} }

func (r *NameYourEdges) Check(ctx *LintContext) []Finding {
	unnamed := 0
	for _, ed := range ctx.Def.Edges {
		if ed.Name == "" {
			unnamed++
		}
	}
	if unnamed > 3 {
		return []Finding{{
			RuleID:   r.ID(),
			Severity: r.Severity(),
			Message:  fmt.Sprintf("circuit has %d unnamed edges; consider adding name fields for readability", unnamed),
			File:     ctx.File,
			Line:     ctx.TopLevelLine("edges"),
		}}
	}
	return nil
}

// --- B3: terminal-edge-to-done ---

type TerminalEdgeToDone struct{}

func (r *TerminalEdgeToDone) ID() string          { return "B3/terminal-edge-to-done" }
func (r *TerminalEdgeToDone) Description() string { return "terminal nodes should have an edge to done" }
func (r *TerminalEdgeToDone) Severity() Severity   { return SeverityWarning }
func (r *TerminalEdgeToDone) Tags() []string       { return []string{"best-practice"} }

func (r *TerminalEdgeToDone) Check(ctx *LintContext) []Finding {
	hasOutgoing := make(map[string]bool)
	for _, ed := range ctx.Def.Edges {
		hasOutgoing[ed.From] = true
	}

	var out []Finding
	for _, nd := range ctx.Def.Nodes {
		if !hasOutgoing[nd.Name] {
			out = append(out, Finding{
				RuleID:   r.ID(),
				Severity: r.Severity(),
				Message:  fmt.Sprintf("node %q has no outgoing edges; add an edge to %q", nd.Name, ctx.Def.Done),
				File:     ctx.File,
				Line:     ctx.NodeLine(nd.Name),
			})
		}
	}
	return out
}

// --- B4: zone-stickiness-without-provider ---

type ZoneStickinessWithoutProvider struct{}

func (r *ZoneStickinessWithoutProvider) ID() string        { return "B4/zone-stickiness-without-provider" }
func (r *ZoneStickinessWithoutProvider) Description() string { return "zone with stickiness but no providers in its nodes" }
func (r *ZoneStickinessWithoutProvider) Severity() Severity { return SeverityInfo }
func (r *ZoneStickinessWithoutProvider) Tags() []string     { return []string{"best-practice"} }

func (r *ZoneStickinessWithoutProvider) Check(ctx *LintContext) []Finding {
	// Providers are often runtime-configured and stickiness is valid
	// as declarative intent even when YAML-level provider fields are absent.
	// Only flag when stickiness is "exclusive" (3) and zero providers exist
	// anywhere in the circuit, suggesting a configuration oversight.
	anyProvider := false
	for _, nd := range ctx.Def.Nodes {
		if nd.Provider != "" {
			anyProvider = true
			break
		}
	}
	if anyProvider {
		return nil
	}

	var out []Finding
	for zoneName, z := range ctx.Def.Zones {
		if z.Stickiness < 3 {
			continue
		}
		out = append(out, Finding{
			RuleID:   r.ID(),
			Severity: r.Severity(),
			Message:  fmt.Sprintf("zone %q has exclusive stickiness=%d but no nodes in the entire circuit declare a provider", zoneName, z.Stickiness),
			File:     ctx.File,
		})
	}
	return out
}

// --- B5: large-circuit-no-zones ---

type LargeCircuitNoZones struct{}

func (r *LargeCircuitNoZones) ID() string          { return "B5/large-circuit-no-zones" }
func (r *LargeCircuitNoZones) Description() string { return "large circuits should define zones for organization" }
func (r *LargeCircuitNoZones) Severity() Severity   { return SeverityInfo }
func (r *LargeCircuitNoZones) Tags() []string       { return []string{"best-practice"} }

func (r *LargeCircuitNoZones) Check(ctx *LintContext) []Finding {
	if len(ctx.Def.Nodes) > 6 && len(ctx.Def.Zones) == 0 {
		return []Finding{{
			RuleID:   r.ID(),
			Severity: r.Severity(),
			Message:  fmt.Sprintf("circuit has %d nodes but no zones; consider adding zones for organization", len(ctx.Def.Nodes)),
			File:     ctx.File,
			Line:     ctx.TopLevelLine("nodes"),
		}}
	}
	return nil
}

// --- B6: element-affinity-chain ---

type ElementAffinityChain struct{}

func (r *ElementAffinityChain) ID() string          { return "B6/element-affinity-chain" }
func (r *ElementAffinityChain) Description() string { return "three or more consecutive nodes with the same element" }
func (r *ElementAffinityChain) Severity() Severity   { return SeverityInfo }
func (r *ElementAffinityChain) Tags() []string       { return []string{"best-practice"} }

func (r *ElementAffinityChain) Check(ctx *LintContext) []Finding {
	nodeElems := make(map[string]string)
	for _, nd := range ctx.Def.Nodes {
		if nd.Element != "" {
			nodeElems[nd.Name] = strings.ToLower(nd.Element)
		}
	}

	adj := make(map[string][]string)
	for _, ed := range ctx.Def.Edges {
		if !ed.Shortcut && !ed.Loop {
			adj[ed.From] = append(adj[ed.From], ed.To)
		}
	}

	var out []Finding
	reported := make(map[string]bool)
	for _, nd := range ctx.Def.Nodes {
		elem := nodeElems[nd.Name]
		if elem == "" {
			continue
		}
		chain := findElementChain(nd.Name, elem, nodeElems, adj)
		if len(chain) >= 3 && !reported[elem+":"+chain[0]] {
			reported[elem+":"+chain[0]] = true
			out = append(out, Finding{
				RuleID:   r.ID(),
				Severity: r.Severity(),
				Message:  fmt.Sprintf("%d consecutive %s-element nodes: %s; consider varying elements for balance", len(chain), elem, strings.Join(chain, " → ")),
				File:     ctx.File,
				Line:     ctx.NodeLine(chain[0]),
			})
		}
	}
	return out
}

func findElementChain(start, elem string, nodeElems map[string]string, adj map[string][]string) []string {
	chain := []string{start}
	curr := start
	visited := map[string]bool{start: true}
	for {
		nexts := adj[curr]
		extended := false
		for _, next := range nexts {
			if !visited[next] && nodeElems[next] == elem {
				chain = append(chain, next)
				visited[next] = true
				curr = next
				extended = true
				break
			}
		}
		if !extended {
			break
		}
	}
	return chain
}

// --- B7: stochastic-transformer ---

// knownStochasticTransformers is the fallback list used when no
// TransformerRegistry is available at lint time (static YAML analysis).
var knownStochasticTransformers = map[string]bool{
	"core.llm": true,
	"llm":      true,
}

type StochasticTransformer struct{}

func (r *StochasticTransformer) ID() string          { return "B7/stochastic-transformer" }
func (r *StochasticTransformer) Description() string { return "node uses a stochastic (non-deterministic) transformer" }
func (r *StochasticTransformer) Severity() Severity   { return SeverityInfo }
func (r *StochasticTransformer) Tags() []string       { return []string{"best-practice", "determinism"} }

func (r *StochasticTransformer) Check(ctx *LintContext) []Finding {
	var reg framework.TransformerRegistry
	if ctx.Registries != nil {
		reg = ctx.Registries.Transformers
	}

	var out []Finding
	for _, nd := range ctx.Def.Nodes {
		if nd.Transformer == "" {
			continue
		}
		if isStochastic(nd.Transformer, reg) {
			out = append(out, Finding{
				RuleID:   r.ID(),
				Severity: r.Severity(),
				Message:  fmt.Sprintf("node %q uses stochastic transformer %q", nd.Name, nd.Transformer),
				File:     ctx.File,
				Line:     ctx.NodeLine(nd.Name),
			})
		}
	}
	return out
}

func isStochastic(name string, reg framework.TransformerRegistry) bool {
	if reg != nil {
		if t, err := reg.Get(name); err == nil {
			return !framework.IsDeterministic(t)
		}
	}
	return knownStochasticTransformers[name]
}

// --- B7s: stochastic-summary ---

type StochasticSummary struct{}

func (r *StochasticSummary) ID() string          { return "B7s/stochastic-summary" }
func (r *StochasticSummary) Description() string { return "aggregate count of stochastic nodes in circuit" }
func (r *StochasticSummary) Severity() Severity   { return SeverityInfo }
func (r *StochasticSummary) Tags() []string       { return []string{"best-practice", "determinism"} }

func (r *StochasticSummary) Check(ctx *LintContext) []Finding {
	var reg framework.TransformerRegistry
	if ctx.Registries != nil {
		reg = ctx.Registries.Transformers
	}

	totalWithTransformer := 0
	var stochasticNames []string
	for _, nd := range ctx.Def.Nodes {
		if nd.Transformer == "" {
			continue
		}
		totalWithTransformer++
		if isStochastic(nd.Transformer, reg) {
			stochasticNames = append(stochasticNames, nd.Name)
		}
	}

	if len(stochasticNames) == 0 {
		return nil
	}

	return []Finding{{
		RuleID:   r.ID(),
		Severity: r.Severity(),
		Message:  fmt.Sprintf("circuit has %d stochastic node(s) out of %d transformer-bound: %s", len(stochasticNames), totalWithTransformer, strings.Join(stochasticNames, ", ")),
		File:     ctx.File,
	}}
}
