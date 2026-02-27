package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/uri"
)

// InlayHint represents an LSP 3.17 inlay hint (not in go.lsp.dev/protocol v0.12).
type InlayHint struct {
	Position    Position     `json:"position"`
	Label       string       `json:"label"`
	Kind        int          `json:"kind"` // 1=Type, 2=Parameter
	PaddingLeft bool         `json:"paddingLeft,omitempty"`
	Tooltip     *HintTooltip `json:"tooltip,omitempty"`
}

// Position is a minimal position for inlay hint JSON serialization.
type Position struct {
	Line      uint32 `json:"line"`
	Character uint32 `json:"character"`
}

// HintTooltip wraps markdown content for hover-over detail.
type HintTooltip struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

func markdownTooltip(md string) *HintTooltip {
	return &HintTooltip{Kind: "markdown", Value: md}
}

// inlayHintParams mirrors the LSP InlayHintParams (not in protocol v0.12).
type inlayHintParams struct {
	TextDocument struct {
		URI string `json:"uri"`
	} `json:"textDocument"`
}

func (s *Server) handleInlayHint(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	var params inlayHintParams
	if err := json.Unmarshal(req.Params(), &params); err != nil {
		return reply(ctx, nil, err)
	}

	doc := s.getDocument(uri.URI(params.TextDocument.URI))
	if doc == nil {
		return reply(ctx, []InlayHint{}, nil)
	}

	hints := computeInlayHints(doc)

	s.mu.Lock()
	bridge := s.kamiBridge
	s.mu.Unlock()
	if bridge != nil && bridge.Connected() {
		hints = append(hints, bridge.LiveInlayHints(doc)...)
	}

	return reply(ctx, hints, nil)
}

func computeInlayHints(doc *document) []InlayHint {
	if doc.Def == nil {
		return nil
	}

	lines := strings.Split(doc.Content, "\n")
	var hints []InlayHint

	hints = append(hints, elementTraitHints(doc, lines)...)
	hints = append(hints, personaHints(doc, lines)...)
	hints = append(hints, expressionHints(doc, lines)...)
	hints = append(hints, edgeFlowHints(doc, lines)...)
	hints = append(hints, startNodeHint(doc, lines)...)

	return hints
}

func elementTraitHints(doc *document, lines []string) []InlayHint {
	var hints []InlayHint
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "element:") {
			continue
		}
		val := strings.TrimSpace(strings.TrimPrefix(trimmed, "element:"))
		info, ok := elementDocs[val]
		if !ok {
			continue
		}

		summary := compactTraits(info.Traits)
		hints = append(hints, InlayHint{
			Position:    Position{Line: uint32(i), Character: uint32(len(line))},
			Label:       summary,
			Kind:        1,
			PaddingLeft: true,
			Tooltip:     markdownTooltip(fmt.Sprintf("### %s\n\n%s\n\n**Traits:** %s", val, info.Description, info.Traits)),
		})
	}
	return hints
}

func personaHints(doc *document, lines []string) []InlayHint {
	var hints []InlayHint
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "persona:") {
			continue
		}
		val := strings.TrimSpace(strings.TrimPrefix(trimmed, "persona:"))
		desc, ok := personaDocs[val]
		if !ok {
			continue
		}

		short := desc
		if idx := strings.Index(desc, "."); idx > 0 && idx < 60 {
			short = desc[:idx+1]
		}
		hints = append(hints, InlayHint{
			Position:    Position{Line: uint32(i), Character: uint32(len(line))},
			Label:       short,
			Kind:        1,
			PaddingLeft: true,
			Tooltip:     markdownTooltip(fmt.Sprintf("### %s\n\n%s", val, desc)),
		})
	}
	return hints
}

func expressionHints(doc *document, lines []string) []InlayHint {
	var hints []InlayHint
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "when:") {
			continue
		}
		expr := strings.TrimSpace(strings.TrimPrefix(trimmed, "when:"))
		expr = strings.Trim(expr, `"'`)

		label := "expr"
		if expr == "true" || expr == "false" {
			label = "static"
		} else if strings.Contains(expr, "output.") {
			label = "output-dep"
		} else if strings.Contains(expr, "state.") {
			label = "state-dep"
		}

		hints = append(hints, InlayHint{
			Position:    Position{Line: uint32(i), Character: uint32(len(line))},
			Label:       label,
			Kind:        2,
			PaddingLeft: true,
		})
	}
	return hints
}

func edgeFlowHints(doc *document, lines []string) []InlayHint {
	if doc.Def == nil {
		return nil
	}

	nodeElements := make(map[string]string, len(doc.Def.Nodes))
	for _, n := range doc.Def.Nodes {
		if n.Element != "" {
			nodeElements[n.Name] = n.Element
		}
	}

	var hints []InlayHint
	for _, edge := range doc.Def.Edges {
		fromEl := nodeElements[edge.From]
		toEl := nodeElements[edge.To]
		if fromEl == "" && toEl == "" {
			continue
		}

		flow := buildFlowLabel(fromEl, toEl)
		line := findEdgeIDLine(lines, edge.ID)
		if line < 0 {
			continue
		}

		hints = append(hints, InlayHint{
			Position:    Position{Line: uint32(line), Character: uint32(len(lines[line]))},
			Label:       flow,
			Kind:        1,
			PaddingLeft: true,
		})
	}
	return hints
}

func startNodeHint(doc *document, lines []string) []InlayHint {
	if doc.Def == nil || doc.Def.Start == "" {
		return nil
	}

	var hints []InlayHint
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "start:") {
			nodeName := strings.TrimSpace(strings.TrimPrefix(trimmed, "start:"))
			if nodeName == doc.Def.Start {
				for _, n := range doc.Def.Nodes {
					if n.Name == nodeName {
						label := "entry"
						if n.Element != "" {
							label = fmt.Sprintf("entry [%s]", n.Element)
						}
						if n.Family != "" {
							label += fmt.Sprintf(" family=%s", n.Family)
						}
						hints = append(hints, InlayHint{
							Position:    Position{Line: uint32(i), Character: uint32(len(line))},
							Label:       label,
							Kind:        1,
							PaddingLeft: true,
						})
						break
					}
				}
			}
			break
		}
	}
	return hints
}

// compactTraits turns "speed: high | max_loops: 0 | shortcut: 0.9 | failure: premature conclusions"
// into "high | 0 loops | 0.9 shortcut"
func compactTraits(traits string) string {
	parts := strings.Split(traits, "|")
	var compact []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		kv := strings.SplitN(p, ":", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.TrimSpace(kv[0])
		val := strings.TrimSpace(kv[1])
		switch key {
		case "speed":
			compact = append(compact, val)
		case "max_loops":
			compact = append(compact, val+" loops")
		case "shortcut":
			compact = append(compact, val+" shortcut")
		}
	}
	return strings.Join(compact, " | ")
}

func buildFlowLabel(fromEl, toEl string) string {
	if fromEl != "" && toEl != "" {
		return fromEl + " → " + toEl
	}
	if fromEl != "" {
		return fromEl + " →"
	}
	return "→ " + toEl
}

func findEdgeIDLine(lines []string, edgeID string) int {
	target := "id: " + edgeID
	for i, line := range lines {
		if strings.TrimSpace(line) == "- "+target || strings.TrimSpace(line) == target {
			return i
		}
	}
	return -1
}
