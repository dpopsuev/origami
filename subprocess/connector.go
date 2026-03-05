package subprocess

import (
	"context"
	"fmt"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// MCPConnector creates MCP client sessions over arbitrary transports.
// It extracts the duplicated client-creation logic from Server and
// ContainerBackend into a single reusable component.
type MCPConnector struct {
	Name    string // client implementation name, e.g. "origami-subprocess-client"
	Version string // client implementation version
}

// DefaultConnector returns an MCPConnector with standard Origami naming.
func DefaultConnector() *MCPConnector {
	return &MCPConnector{
		Name:    "origami-client",
		Version: "v0.1.0",
	}
}

// Connect creates an MCP client and establishes a session over the given transport.
func (mc *MCPConnector) Connect(ctx context.Context, transport sdkmcp.Transport) (*sdkmcp.ClientSession, error) {
	client := sdkmcp.NewClient(
		&sdkmcp.Implementation{Name: mc.Name, Version: mc.Version},
		nil,
	)

	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("MCP connect (%s): %w", mc.Name, err)
	}
	return session, nil
}
