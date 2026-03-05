package subprocess

import (
	"context"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// SchematicBackend is the lifecycle and tool-call interface that all
// schematic backends (subprocess, container) implement. Orchestrator
// accepts this interface for polymorphic dispatch.
type SchematicBackend interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	CallTool(ctx context.Context, name string, args map[string]any) (*sdkmcp.CallToolResult, error)
	Healthy(ctx context.Context) bool
}

// ToolCaller is the minimal interface for making MCP tool calls.
// Schematic adapters (e.g. knowledge.MCPReader) depend on this
// rather than the full SchematicBackend.
type ToolCaller interface {
	CallTool(ctx context.Context, name string, args map[string]any) (*sdkmcp.CallToolResult, error)
}

// Compile-time checks.
var (
	_ SchematicBackend = (*Server)(nil)
	_ SchematicBackend = (*ContainerBackend)(nil)
	_ ToolCaller       = (*Server)(nil)
	_ ToolCaller       = (*ContainerBackend)(nil)
)
