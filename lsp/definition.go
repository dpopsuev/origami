package lsp

import (
	"context"
	"encoding/json"
	"strings"

	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
)

func (s *Server) handleDefinition(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	var params protocol.DefinitionParams
	if err := json.Unmarshal(req.Params(), &params); err != nil {
		return reply(ctx, nil, err)
	}

	doc := s.getDocument(params.TextDocument.URI)
	if doc == nil || doc.Def == nil || doc.LintCtx == nil {
		return reply(ctx, nil, nil)
	}

	loc := computeDefinition(doc, params.Position)
	if loc == nil {
		return reply(ctx, nil, nil)
	}
	return reply(ctx, loc, nil)
}

func computeDefinition(doc *document, pos protocol.Position) *protocol.Location {
	lines := strings.Split(doc.Content, "\n")
	if int(pos.Line) >= len(lines) {
		return nil
	}

	line := lines[pos.Line]
	trimmed := strings.TrimSpace(line)

	// Navigate from edge from/to or start to node definition
	var targetNode string
	for _, prefix := range []string{"from:", "to:", "start:"} {
		if strings.HasPrefix(trimmed, prefix) {
			targetNode = strings.TrimSpace(strings.TrimPrefix(trimmed, prefix))
			break
		}
	}

	if targetNode == "" || doc.LintCtx == nil {
		return nil
	}

	nodeLine := doc.LintCtx.NodeLine(targetNode)
	if nodeLine <= 0 {
		return nil
	}

	return &protocol.Location{
		URI: doc.URI,
		Range: protocol.Range{
			Start: protocol.Position{Line: uint32(nodeLine - 1), Character: 0},
			End:   protocol.Position{Line: uint32(nodeLine - 1), Character: 0},
		},
	}
}
