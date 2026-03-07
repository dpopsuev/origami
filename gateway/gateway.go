// Package gateway implements an MCP proxy that routes tool calls to
// backend MCP services based on tool name. It aggregates tool lists
// from all backends and dispatches CallTool requests to the backend
// that registered each tool.
package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/dpopsuev/origami/subprocess"
)

// BackendConfig describes a named backend MCP service.
type BackendConfig struct {
	Name     string
	Endpoint string
}

// Gateway proxies MCP tool calls to backend services.
type Gateway struct {
	mu         sync.RWMutex
	backends   map[string]*subprocess.RemoteBackend
	sessions   map[string]*sdkmcp.ClientSession
	toolRoutes map[string]string // tool name -> backend name
}

// New creates a Gateway that will connect to the given backends.
func New(configs []BackendConfig) *Gateway {
	gw := &Gateway{
		backends:   make(map[string]*subprocess.RemoteBackend, len(configs)),
		sessions:   make(map[string]*sdkmcp.ClientSession, len(configs)),
		toolRoutes: make(map[string]string),
	}
	for _, cfg := range configs {
		gw.backends[cfg.Name] = &subprocess.RemoteBackend{Endpoint: cfg.Endpoint}
	}
	return gw
}

// Start connects to all backends and builds the tool routing table.
func (gw *Gateway) Start(ctx context.Context) error {
	for name, rb := range gw.backends {
		if err := rb.Start(ctx); err != nil {
			return fmt.Errorf("backend %q: %w", name, err)
		}
	}

	for name, rb := range gw.backends {
		transport := &sdkmcp.StreamableClientTransport{Endpoint: rb.Endpoint}
		client := sdkmcp.NewClient(
			&sdkmcp.Implementation{Name: "origami-gateway", Version: "v0.1.0"},
			nil,
		)
		session, err := client.Connect(ctx, transport, nil)
		if err != nil {
			return fmt.Errorf("discover tools for %q: %w", name, err)
		}
		gw.sessions[name] = session

		tools, err := session.ListTools(ctx, nil)
		if err != nil {
			session.Close()
			return fmt.Errorf("list tools for %q: %w", name, err)
		}

		gw.mu.Lock()
		for _, tool := range tools.Tools {
			gw.toolRoutes[tool.Name] = name
		}
		gw.mu.Unlock()
	}

	return nil
}

// Stop closes all discovery sessions and backend connections.
func (gw *Gateway) Stop(ctx context.Context) {
	for _, s := range gw.sessions {
		s.Close()
	}
	for _, rb := range gw.backends {
		rb.Stop(ctx)
	}
}

// CallTool routes a tool call to the backend that registered the tool.
func (gw *Gateway) CallTool(ctx context.Context, name string, args map[string]any) (*sdkmcp.CallToolResult, error) {
	gw.mu.RLock()
	backendName, ok := gw.toolRoutes[name]
	gw.mu.RUnlock()

	if !ok {
		return &sdkmcp.CallToolResult{
			IsError: true,
			Content: []sdkmcp.Content{&sdkmcp.TextContent{Text: fmt.Sprintf("unknown tool %q", name)}},
		}, nil
	}

	return gw.backends[backendName].CallTool(ctx, name, args)
}

// ListTools returns the merged tool list from all backends.
func (gw *Gateway) ListTools() []sdkmcp.Tool {
	gw.mu.RLock()
	defer gw.mu.RUnlock()

	// Re-discover would be better, but for now return from routing table.
	var tools []sdkmcp.Tool
	for name := range gw.toolRoutes {
		tools = append(tools, sdkmcp.Tool{
			Name:        name,
			InputSchema: json.RawMessage(`{"type":"object"}`),
		})
	}
	return tools
}

// Healthy returns true if all backends respond to ping.
func (gw *Gateway) Healthy(ctx context.Context) bool {
	for _, rb := range gw.backends {
		if !rb.Healthy(ctx) {
			return false
		}
	}
	return true
}

// UnhealthyBackends returns the names of backends that fail health checks.
func (gw *Gateway) UnhealthyBackends(ctx context.Context) []string {
	var unhealthy []string
	for name, rb := range gw.backends {
		if !rb.Healthy(ctx) {
			unhealthy = append(unhealthy, name)
		}
	}
	return unhealthy
}

// MCPServer creates an sdkmcp.Server that proxies all tool calls through the Gateway.
func (gw *Gateway) MCPServer() *sdkmcp.Server {
	server := sdkmcp.NewServer(
		&sdkmcp.Implementation{Name: "origami-gateway", Version: "v0.1.0"},
		nil,
	)

	gw.mu.RLock()
	defer gw.mu.RUnlock()

	for toolName := range gw.toolRoutes {
		tn := toolName
		server.AddTool(
			&sdkmcp.Tool{
				Name:        tn,
				InputSchema: json.RawMessage(`{"type":"object"}`),
			},
			func(ctx context.Context, req *sdkmcp.CallToolRequest) (*sdkmcp.CallToolResult, error) {
				var args map[string]any
				if req.Params.Arguments != nil {
					json.Unmarshal(req.Params.Arguments, &args)
				}
				return gw.CallTool(ctx, tn, args)
			},
		)
	}

	return server
}

// Handler returns an http.Handler with MCP, health, and readiness endpoints.
func (gw *Gateway) Handler() http.Handler {
	mcpServer := gw.MCPServer()

	mcpHandler := sdkmcp.NewStreamableHTTPHandler(
		func(_ *http.Request) *sdkmcp.Server { return mcpServer },
		&sdkmcp.StreamableHTTPOptions{Stateless: true},
	)

	mux := http.NewServeMux()
	mux.Handle("/mcp", mcpHandler)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if gw.Healthy(r.Context()) {
			w.WriteHeader(http.StatusOK)
			return
		}
		unhealthy := gw.UnhealthyBackends(r.Context())
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, "unhealthy backends: %v", unhealthy)
	})
	return mux
}
