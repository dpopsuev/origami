package lint

import (
	"fmt"
	"strings"

	framework "github.com/dpopsuev/origami"
)

// --- G1: orphan-node ---

type OrphanNode struct{}

func (r *OrphanNode) ID() string          { return "G1/orphan-node" }
func (r *OrphanNode) Description() string { return "node not reachable from start via any edge path" }
func (r *OrphanNode) Severity() Severity   { return SeverityWarning }
func (r *OrphanNode) Tags() []string       { return []string{"semantic"} }

func (r *OrphanNode) Check(ctx *LintContext) []Finding {
	reachable := reachableNodes(ctx.Def)
	var out []Finding
	for _, nd := range ctx.Def.Nodes {
		if !reachable[nd.Name] {
			out = append(out, Finding{
				RuleID:   r.ID(),
				Severity: r.Severity(),
				Message:  fmt.Sprintf("node %q is not reachable from start node %q", nd.Name, ctx.Def.Start),
				File:     ctx.File,
				Line:     ctx.NodeLine(nd.Name),
			})
		}
	}
	return out
}

// --- G2: unreachable-done ---

type UnreachableDone struct{}

func (r *UnreachableDone) ID() string          { return "G2/unreachable-done" }
func (r *UnreachableDone) Description() string { return "no edge path from start reaches done" }
func (r *UnreachableDone) Severity() Severity   { return SeverityError }
func (r *UnreachableDone) Tags() []string       { return []string{"semantic"} }

func (r *UnreachableDone) Check(ctx *LintContext) []Finding {
	if ctx.Def.Done == "" || ctx.Def.Start == "" {
		return nil
	}
	adj := buildAdjacency(ctx.Def)
	visited := bfs(ctx.Def.Start, adj)
	if !visited[ctx.Def.Done] {
		return []Finding{{
			RuleID:   r.ID(),
			Severity: r.Severity(),
			Message:  fmt.Sprintf("no path from start %q reaches done node %q", ctx.Def.Start, ctx.Def.Done),
			File:     ctx.File,
			Line:     ctx.TopLevelLine("done"),
		}}
	}
	return nil
}

// --- G3: dead-edge ---

type DeadEdge struct{}

func (r *DeadEdge) ID() string          { return "G3/dead-edge" }
func (r *DeadEdge) Description() string { return "edge from unreachable node is dead" }
func (r *DeadEdge) Severity() Severity   { return SeverityWarning }
func (r *DeadEdge) Tags() []string       { return []string{"semantic"} }

func (r *DeadEdge) Check(ctx *LintContext) []Finding {
	reachable := reachableNodes(ctx.Def)
	var out []Finding
	for _, ed := range ctx.Def.Edges {
		if !reachable[ed.From] {
			out = append(out, Finding{
				RuleID:   r.ID(),
				Severity: r.Severity(),
				Message:  fmt.Sprintf("edge %q originates from unreachable node %q", ed.ID, ed.From),
				File:     ctx.File,
				Line:     ctx.EdgeLine(ed.ID),
			})
		}
	}
	return out
}

// --- G4: shortcut-bypasses-required ---

type ShortcutBypassesRequired struct{}

func (r *ShortcutBypassesRequired) ID() string        { return "G4/shortcut-bypasses-required" }
func (r *ShortcutBypassesRequired) Description() string { return "shortcut edge skips a node with a schema" }
func (r *ShortcutBypassesRequired) Severity() Severity { return SeverityWarning }
func (r *ShortcutBypassesRequired) Tags() []string     { return []string{"semantic"} }

func (r *ShortcutBypassesRequired) Check(ctx *LintContext) []Finding {
	schemaNodes := make(map[string]bool)
	for _, nd := range ctx.Def.Nodes {
		if nd.Schema != nil {
			schemaNodes[nd.Name] = true
		}
	}
	if len(schemaNodes) == 0 {
		return nil
	}

	normalAdj := make(map[string][]string)
	for _, ed := range ctx.Def.Edges {
		if !ed.Shortcut {
			normalAdj[ed.From] = append(normalAdj[ed.From], ed.To)
		}
	}

	var out []Finding
	for _, ed := range ctx.Def.Edges {
		if !ed.Shortcut {
			continue
		}
		skipped := nodesOnPath(ed.From, ed.To, normalAdj)
		for name := range skipped {
			if schemaNodes[name] {
				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.Severity(),
					Message:  fmt.Sprintf("shortcut edge %q bypasses schema-bearing node %q", ed.ID, name),
					File:     ctx.File,
					Line:     ctx.EdgeLine(ed.ID),
				})
			}
		}
	}
	return out
}

// --- G5: zone-element-mismatch ---

type ZoneElementMismatch struct{}

func (r *ZoneElementMismatch) ID() string          { return "G5/zone-element-mismatch" }
func (r *ZoneElementMismatch) Description() string { return "zone element differs from contained node element" }
func (r *ZoneElementMismatch) Severity() Severity   { return SeverityInfo }
func (r *ZoneElementMismatch) Tags() []string       { return []string{"semantic"} }

func (r *ZoneElementMismatch) Check(ctx *LintContext) []Finding {
	nodeElems := make(map[string]string)
	for _, nd := range ctx.Def.Nodes {
		if nd.Element != "" {
			nodeElems[nd.Name] = strings.ToLower(nd.Element)
		}
	}

	var out []Finding
	for zoneName, z := range ctx.Def.Zones {
		if z.Element == "" || len(z.Nodes) == 0 {
			continue
		}
		zoneElem := strings.ToLower(z.Element)

		// Only flag when NO nodes in the zone match the zone element.
		// Mixed elements within a zone is a valid design pattern.
		anyMatch := false
		for _, nodeName := range z.Nodes {
			if ne, ok := nodeElems[nodeName]; ok && ne == zoneElem {
				anyMatch = true
				break
			}
		}
		if !anyMatch {
			out = append(out, Finding{
				RuleID:   r.ID(),
				Severity: r.Severity(),
				Message:  fmt.Sprintf("zone %q declares element %q but none of its nodes use that element", zoneName, z.Element),
				File:     ctx.File,
			})
		}
	}
	return out
}

// --- G6: expression-compile-error ---

type ExpressionCompileError struct{}

func (r *ExpressionCompileError) ID() string          { return "G6/expression-compile-error" }
func (r *ExpressionCompileError) Description() string { return "when expression does not compile" }
func (r *ExpressionCompileError) Severity() Severity   { return SeverityError }
func (r *ExpressionCompileError) Tags() []string       { return []string{"semantic"} }

func (r *ExpressionCompileError) Check(ctx *LintContext) []Finding {
	var out []Finding
	for _, ed := range ctx.Def.Edges {
		if ed.When == "" {
			continue
		}
		if _, err := framework.CompileExpressionEdge(ed, ctx.Def.Vars); err != nil {
			out = append(out, Finding{
				RuleID:   r.ID(),
				Severity: r.Severity(),
				Message:  fmt.Sprintf("edge %q: %v", ed.ID, err),
				File:     ctx.File,
				Line:     ctx.EdgeLine(ed.ID),
			})
		}
	}
	return out
}

// --- G7: fan-in-without-merge ---

type FanInWithoutMerge struct{}

func (r *FanInWithoutMerge) ID() string          { return "G7/fan-in-without-merge" }
func (r *FanInWithoutMerge) Description() string { return "multiple edges converge on a node without merge strategy" }
func (r *FanInWithoutMerge) Severity() Severity   { return SeverityWarning }
func (r *FanInWithoutMerge) Tags() []string       { return []string{"semantic"} }

func (r *FanInWithoutMerge) Check(ctx *LintContext) []Finding {
	type edgeInfo struct {
		id          string
		conditional bool
	}
	inbound := make(map[string][]edgeInfo)
	for _, ed := range ctx.Def.Edges {
		conditional := ed.When != "" || ed.Condition != "" || ed.Shortcut || ed.Parallel
		inbound[ed.To] = append(inbound[ed.To], edgeInfo{id: ed.ID, conditional: conditional})
	}

	hasMerge := make(map[string]bool)
	for _, ed := range ctx.Def.Edges {
		if ed.Merge != "" {
			hasMerge[ed.To] = true
		}
	}

	var out []Finding
	for node, edges := range inbound {
		if len(edges) <= 1 || hasMerge[node] || node == ctx.Def.Done {
			continue
		}
		// Only flag when at least two inbound edges could fire simultaneously
		// (unconditional edges). If all inbound edges are conditional/parallel/shortcut,
		// the fan-in is guarded by routing logic and merge is unnecessary.
		unconditional := 0
		for _, e := range edges {
			if !e.conditional {
				unconditional++
			}
		}
		if unconditional >= 2 {
			out = append(out, Finding{
				RuleID:   r.ID(),
				Severity: r.Severity(),
				Message:  fmt.Sprintf("node %q has %d unconditional inbound edges but no merge strategy", node, unconditional),
				File:     ctx.File,
				Line:     ctx.NodeLine(node),
			})
		}
	}
	return out
}

// --- Graph helpers ---

func buildAdjacency(def *framework.CircuitDef) map[string][]string {
	adj := make(map[string][]string)
	for _, ed := range def.Edges {
		adj[ed.From] = append(adj[ed.From], ed.To)
	}
	return adj
}

func bfs(start string, adj map[string][]string) map[string]bool {
	visited := map[string]bool{start: true}
	queue := []string{start}
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		for _, next := range adj[curr] {
			if !visited[next] {
				visited[next] = true
				queue = append(queue, next)
			}
		}
	}
	return visited
}

func reachableNodes(def *framework.CircuitDef) map[string]bool {
	if def.Start == "" {
		return nil
	}
	return bfs(def.Start, buildAdjacency(def))
}

// nodesOnPath returns nodes between from and to (exclusive of both) reachable
// via BFS through the adjacency map. Used by shortcut-bypasses-required.
func nodesOnPath(from, to string, adj map[string][]string) map[string]bool {
	visited := bfs(from, adj)
	skipped := make(map[string]bool)
	for node := range visited {
		if node != from && node != to {
			skipped[node] = true
		}
	}
	return skipped
}
