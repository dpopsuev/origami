package lsp

import (
	"context"
	"encoding/json"
	"strings"

	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
)

var topLevelKeys = []string{
	"pipeline", "description", "imports", "vars", "zones",
	"nodes", "edges", "walkers", "start", "done",
}

var nodeFieldKeys = []string{
	"name", "element", "family", "extractor", "transformer",
	"provider", "prompt", "input", "after", "schema", "cache", "marble",
}

var edgeFieldKeys = []string{
	"id", "name", "from", "to", "shortcut", "loop",
	"parallel", "condition", "when", "merge",
}

var walkerFieldKeys = []string{
	"name", "element", "persona", "preamble", "step_affinity",
}

var elementValues = []string{
	"fire", "water", "earth", "air", "diamond", "lightning", "iron",
}

var personaValues = []string{
	"herald", "seeker", "sentinel", "weaver",
	"arbiter", "catalyst", "oracle", "phantom",
}

func (s *Server) handleCompletion(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	var params protocol.CompletionParams
	if err := json.Unmarshal(req.Params(), &params); err != nil {
		return reply(ctx, nil, err)
	}

	doc := s.getDocument(params.TextDocument.URI)
	if doc == nil {
		return reply(ctx, &protocol.CompletionList{}, nil)
	}

	items := computeCompletions(doc, params.Position)
	return reply(ctx, &protocol.CompletionList{Items: items}, nil)
}

func computeCompletions(doc *document, pos protocol.Position) []protocol.CompletionItem {
	lines := strings.Split(doc.Content, "\n")
	if int(pos.Line) >= len(lines) {
		return nil
	}

	line := lines[pos.Line]
	trimmed := strings.TrimSpace(line)
	indent := len(line) - len(strings.TrimLeft(line, " "))

	// Top-level key completion (indent 0)
	if indent == 0 && (trimmed == "" || !strings.Contains(trimmed, ":")) {
		return keyCompletions(topLevelKeys, protocol.CompletionItemKindField)
	}

	// After "element:" suggest element values
	if strings.HasPrefix(trimmed, "element:") || strings.HasPrefix(trimmed, "element: ") {
		return valueCompletions(elementValues, protocol.CompletionItemKindEnum)
	}

	// After "persona:" suggest persona values
	if strings.HasPrefix(trimmed, "persona:") || strings.HasPrefix(trimmed, "persona: ") {
		return valueCompletions(personaValues, protocol.CompletionItemKindEnum)
	}

	// Node references in "from:", "to:", "start:"
	if (strings.HasPrefix(trimmed, "from:") || strings.HasPrefix(trimmed, "to:") ||
		strings.HasPrefix(trimmed, "start:")) && doc.Def != nil {
		names := make([]string, 0, len(doc.Def.Nodes))
		for _, n := range doc.Def.Nodes {
			names = append(names, n.Name)
		}
		return valueCompletions(names, protocol.CompletionItemKindReference)
	}

	// Node field completion (indent ~4-6, inside nodes list)
	ctx := guessContext(lines, int(pos.Line))
	switch ctx {
	case "nodes":
		return keyCompletions(nodeFieldKeys, protocol.CompletionItemKindField)
	case "edges":
		return keyCompletions(edgeFieldKeys, protocol.CompletionItemKindField)
	case "walkers":
		return keyCompletions(walkerFieldKeys, protocol.CompletionItemKindField)
	}

	return nil
}

func guessContext(lines []string, curLine int) string {
	for i := curLine; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		switch {
		case line == "nodes:" || strings.HasPrefix(line, "nodes:"):
			return "nodes"
		case line == "edges:" || strings.HasPrefix(line, "edges:"):
			return "edges"
		case line == "walkers:" || strings.HasPrefix(line, "walkers:"):
			return "walkers"
		case line == "zones:" || strings.HasPrefix(line, "zones:"):
			return "zones"
		}
	}
	return ""
}

func keyCompletions(keys []string, kind protocol.CompletionItemKind) []protocol.CompletionItem {
	items := make([]protocol.CompletionItem, 0, len(keys))
	for _, k := range keys {
		items = append(items, protocol.CompletionItem{
			Label:      k,
			Kind:       kind,
			InsertText: k + ": ",
		})
	}
	return items
}

func valueCompletions(values []string, kind protocol.CompletionItemKind) []protocol.CompletionItem {
	items := make([]protocol.CompletionItem, 0, len(values))
	for _, v := range values {
		items = append(items, protocol.CompletionItem{
			Label: v,
			Kind:  kind,
		})
	}
	return items
}

