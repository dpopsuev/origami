package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	framework "github.com/dpopsuev/origami"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
)

type elementInfo struct {
	Description string
	Traits      string
	Color       string
}

var elementDocs = map[string]elementInfo{
	"fire":      {Description: "Bold, fast, confident. First to declare a verdict.", Traits: "speed: high | max_loops: 0 | shortcut: 0.9 | failure: premature conclusions", Color: "#DC143C (Crimson)"},
	"water":     {Description: "Methodical, thorough, evidence-first. Examines every log.", Traits: "speed: low | max_loops: 3 | shortcut: 0.1 | failure: analysis paralysis", Color: "#007BA7 (Cerulean)"},
	"earth":     {Description: "Pragmatic, categorical, infrastructure-focused.", Traits: "speed: medium | max_loops: 1 | shortcut: 0.3 | failure: oversimplification", Color: "#0047AB (Cobalt)"},
	"air":       {Description: "Creative, lateral thinker, cross-domain correlator.", Traits: "speed: medium | max_loops: 2 | shortcut: 0.5 | failure: tangential thinking", Color: "#FFBF00 (Amber)"},
	"diamond":   {Description: "Skeptical, evidence-demanding. The final quality gate.", Traits: "speed: low | max_loops: 2 | shortcut: 0.0 | failure: paralyzing perfectionism", Color: "#0F52BA (Sapphire)"},
	"lightning": {Description: "Dispatcher, orchestrator. Manages the circuit queue.", Traits: "speed: highest | max_loops: 0 | shortcut: 1.0 | failure: sacrifices quality", Color: "#DC143C (Crimson)"},
	"iron":      {Description: "Structural, load-bearing. Foundation node type.", Traits: "speed: lowest | max_loops: 0 | shortcut: 0.0 | failure: rigidity", Color: "#48494B (Iron)"},
}

var personaDocs = map[string]string{
	"herald":   "Fire persona. Bold, fast classifier. \"I saw the error. I already know what happened.\"",
	"seeker":   "Water persona. Deep evidence gatherer. \"Let's not jump to conclusions.\"",
	"sentinel": "Earth persona. Infrastructure specialist. \"I've filed this under infrastructure.\"",
	"weaver":   "Air persona. Cross-repo correlator. \"What if the bug isn't in the code?\"",
	"arbiter":  "Diamond persona. Adversarial reviewer. \"The evidence is inconclusive.\"",
	"catalyst": "Lightning persona. Circuit orchestrator. \"New failure incoming! All units respond!\"",
	"oracle":   "Void persona. Pattern recognizer across time. Sees trends invisible to others.",
	"phantom":  "Antithesis persona. The adversarial counterpart used in the Dialectic system.",
}

var exprContextDocs = map[string]string{
	"output": "The artifact produced by the source node. Fields depend on the node family.",
	"state":  "Walker state: `state.loops.<node>` (loop count), `state.visited` (set of visited nodes).",
	"config": "Circuit vars from the `vars:` section. Access as `config.<var_name>`.",
}

func (s *Server) handleHover(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	var params protocol.HoverParams
	if err := json.Unmarshal(req.Params(), &params); err != nil {
		return reply(ctx, nil, err)
	}

	doc := s.getDocument(params.TextDocument.URI)
	if doc == nil {
		return reply(ctx, nil, nil)
	}

	hover := computeHover(doc, params.Position, s.vocab)
	if hover == nil {
		return reply(ctx, nil, nil)
	}
	return reply(ctx, hover, nil)
}

func computeHover(doc *document, pos protocol.Position, vocab framework.RichVocabulary) *protocol.Hover {
	lines := strings.Split(doc.Content, "\n")
	if int(pos.Line) >= len(lines) {
		return nil
	}

	line := lines[pos.Line]
	trimmed := strings.TrimSpace(line)

	// Element hover
	if strings.HasPrefix(trimmed, "element:") {
		val := strings.TrimSpace(strings.TrimPrefix(trimmed, "element:"))
		if info, ok := elementDocs[val]; ok {
			md := fmt.Sprintf("### Element: %s\n\n%s\n\n**Traits:** %s\n\n**Color:** %s",
				val, info.Description, info.Traits, info.Color)
			return &protocol.Hover{
				Contents: protocol.MarkupContent{Kind: protocol.Markdown, Value: md},
			}
		}
	}

	// Persona hover
	if strings.HasPrefix(trimmed, "persona:") {
		val := strings.TrimSpace(strings.TrimPrefix(trimmed, "persona:"))
		if desc, ok := personaDocs[val]; ok {
			md := fmt.Sprintf("### Persona: %s\n\n%s", val, desc)
			return &protocol.Hover{
				Contents: protocol.MarkupContent{Kind: protocol.Markdown, Value: md},
			}
		}
	}

	// Expression context hover
	if strings.HasPrefix(trimmed, "when:") {
		md := "### Edge Expression Context\n\n"
		for k, v := range exprContextDocs {
			md += fmt.Sprintf("- **%s** — %s\n", k, v)
		}
		return &protocol.Hover{
			Contents: protocol.MarkupContent{Kind: protocol.Markdown, Value: md},
		}
	}

	// Node name hover (in from:/to:/start:)
	for _, prefix := range []string{"from:", "to:", "start:"} {
		if strings.HasPrefix(trimmed, prefix) && doc.Def != nil {
			nodeName := strings.TrimSpace(strings.TrimPrefix(trimmed, prefix))
			for _, n := range doc.Def.Nodes {
				if n.Name == nodeName {
					md := fmt.Sprintf("### Node: %s\n\n", n.Name)
					if n.Family != "" {
						md += fmt.Sprintf("**Family:** %s\n\n", n.Family)
					}
					if n.Element != "" {
						md += fmt.Sprintf("**Element:** %s\n\n", n.Element)
					}
					if vocab != nil {
						if d := vocab.Description(n.Name); d != "" {
							md += fmt.Sprintf("---\n\n%s\n", d)
						}
					}
					return &protocol.Hover{
						Contents: protocol.MarkupContent{Kind: protocol.Markdown, Value: md},
					}
				}
			}
		}
	}

	// Top-level key hover
	topLevelDocs := map[string]string{
		"circuit":    "Circuit name identifier. Used in logs, reports, and MCP routing.",
		"description": "Human-readable circuit description.",
		"imports":     "List of external circuit files to import and merge.",
		"vars":        "Circuit variables. Accessible in edge expressions as `config.<name>`.",
		"zones":       "Logical node groupings. Map zone name to `{ nodes: [...], element: ..., stickiness: N }`.",
		"nodes":       "Circuit nodes. Each node has a name, family, and optional element/extractor/transformer.",
		"edges":       "Conditional transitions between nodes. Each edge has `from`, `to`, and `when` expression.",
		"walkers":     "Walker definitions. Each walker has a name, element, persona, and optional step affinity.",
		"start":       "The starting node for circuit execution.",
		"done":        "The terminal sentinel node. Reaching this node completes the walk.",
	}
	for key, desc := range topLevelDocs {
		if strings.HasPrefix(trimmed, key+":") {
			md := fmt.Sprintf("### %s\n\n%s", key, desc)
			return &protocol.Hover{
				Contents: protocol.MarkupContent{Kind: protocol.Markdown, Value: md},
			}
		}
	}

	return nil
}
