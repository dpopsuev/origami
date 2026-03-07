package gateway_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/dpopsuev/origami/gateway"
)

func newTestBackend(t *testing.T, tools map[string]func(ctx context.Context, req *sdkmcp.CallToolRequest) (*sdkmcp.CallToolResult, error)) *httptest.Server {
	t.Helper()
	server := sdkmcp.NewServer(
		&sdkmcp.Implementation{Name: "test-backend", Version: "v0.1.0"},
		nil,
	)
	for name, handler := range tools {
		server.AddTool(
			&sdkmcp.Tool{
				Name:        name,
				InputSchema: json.RawMessage(`{"type":"object"}`),
			},
			handler,
		)
	}
	h := sdkmcp.NewStreamableHTTPHandler(
		func(_ *http.Request) *sdkmcp.Server { return server },
		&sdkmcp.StreamableHTTPOptions{Stateless: true},
	)
	ts := httptest.NewServer(h)
	t.Cleanup(ts.Close)
	return ts
}

func echoHandler(_ context.Context, req *sdkmcp.CallToolRequest) (*sdkmcp.CallToolResult, error) {
	var args struct{ Message string }
	json.Unmarshal(req.Params.Arguments, &args)
	return &sdkmcp.CallToolResult{
		Content: []sdkmcp.Content{&sdkmcp.TextContent{Text: "echo: " + args.Message}},
	}, nil
}

func addHandler(_ context.Context, req *sdkmcp.CallToolRequest) (*sdkmcp.CallToolResult, error) {
	var args struct {
		A int `json:"a"`
		B int `json:"b"`
	}
	json.Unmarshal(req.Params.Arguments, &args)
	return &sdkmcp.CallToolResult{
		Content: []sdkmcp.Content{&sdkmcp.TextContent{Text: fmt.Sprintf("%d", args.A+args.B)}},
	}, nil
}

func connectGateway(t *testing.T, ts *httptest.Server) *sdkmcp.ClientSession {
	t.Helper()
	ctx := t.Context()
	transport := &sdkmcp.StreamableClientTransport{Endpoint: ts.URL + "/mcp"}
	client := sdkmcp.NewClient(
		&sdkmcp.Implementation{Name: "test-client", Version: "v0.1.0"},
		nil,
	)
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		t.Fatalf("connect to gateway: %v", err)
	}
	t.Cleanup(func() { session.Close() })
	return session
}

func extractText(t *testing.T, result *sdkmcp.CallToolResult) string {
	t.Helper()
	for _, c := range result.Content {
		if tc, ok := c.(*sdkmcp.TextContent); ok {
			return tc.Text
		}
	}
	t.Fatal("no text content in result")
	return ""
}

func TestGateway_RoutesToCorrectBackend(t *testing.T) {
	backend1 := newTestBackend(t, map[string]func(context.Context, *sdkmcp.CallToolRequest) (*sdkmcp.CallToolResult, error){
		"echo": echoHandler,
	})
	backend2 := newTestBackend(t, map[string]func(context.Context, *sdkmcp.CallToolRequest) (*sdkmcp.CallToolResult, error){
		"add": addHandler,
	})

	gw := gateway.New([]gateway.BackendConfig{
		{Name: "svc1", Endpoint: backend1.URL + "/mcp"},
		{Name: "svc2", Endpoint: backend2.URL + "/mcp"},
	})
	ctx := t.Context()
	if err := gw.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer gw.Stop(context.Background())

	ts := httptest.NewServer(gw.Handler())
	defer ts.Close()

	session := connectGateway(t, ts)

	result, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
		Name:      "echo",
		Arguments: mustJSON(map[string]any{"message": "hello"}),
	})
	if err != nil {
		t.Fatalf("CallTool echo: %v", err)
	}
	if got := extractText(t, result); got != "echo: hello" {
		t.Errorf("echo = %q, want %q", got, "echo: hello")
	}

	result, err = session.CallTool(ctx, &sdkmcp.CallToolParams{
		Name:      "add",
		Arguments: mustJSON(map[string]any{"a": 3, "b": 7}),
	})
	if err != nil {
		t.Fatalf("CallTool add: %v", err)
	}
	if got := extractText(t, result); got != "10" {
		t.Errorf("add = %q, want %q", got, "10")
	}
}

func TestGateway_ListToolsAggregation(t *testing.T) {
	backend1 := newTestBackend(t, map[string]func(context.Context, *sdkmcp.CallToolRequest) (*sdkmcp.CallToolResult, error){
		"tool_a": echoHandler,
		"tool_b": echoHandler,
	})
	backend2 := newTestBackend(t, map[string]func(context.Context, *sdkmcp.CallToolRequest) (*sdkmcp.CallToolResult, error){
		"tool_c": echoHandler,
	})

	gw := gateway.New([]gateway.BackendConfig{
		{Name: "svc1", Endpoint: backend1.URL + "/mcp"},
		{Name: "svc2", Endpoint: backend2.URL + "/mcp"},
	})
	ctx := t.Context()
	if err := gw.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer gw.Stop(context.Background())

	ts := httptest.NewServer(gw.Handler())
	defer ts.Close()

	session := connectGateway(t, ts)

	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}

	found := make(map[string]bool)
	for _, tool := range tools.Tools {
		found[tool.Name] = true
	}

	for _, name := range []string{"tool_a", "tool_b", "tool_c"} {
		if !found[name] {
			t.Errorf("tool %q not found in aggregated list", name)
		}
	}
}

func TestGateway_Healthz(t *testing.T) {
	backend := newTestBackend(t, map[string]func(context.Context, *sdkmcp.CallToolRequest) (*sdkmcp.CallToolResult, error){
		"echo": echoHandler,
	})

	gw := gateway.New([]gateway.BackendConfig{
		{Name: "svc1", Endpoint: backend.URL + "/mcp"},
	})
	ctx := t.Context()
	if err := gw.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer gw.Stop(context.Background())

	ts := httptest.NewServer(gw.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("healthz = %d, want 200", resp.StatusCode)
	}
}

func TestGateway_Readyz_AllHealthy(t *testing.T) {
	backend := newTestBackend(t, map[string]func(context.Context, *sdkmcp.CallToolRequest) (*sdkmcp.CallToolResult, error){
		"echo": echoHandler,
	})

	gw := gateway.New([]gateway.BackendConfig{
		{Name: "svc1", Endpoint: backend.URL + "/mcp"},
	})
	ctx := t.Context()
	if err := gw.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer gw.Stop(context.Background())

	ts := httptest.NewServer(gw.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/readyz")
	if err != nil {
		t.Fatalf("GET /readyz: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("readyz = %d, want 200", resp.StatusCode)
	}
}

func TestGateway_UnknownTool_ReturnsError(t *testing.T) {
	backend := newTestBackend(t, map[string]func(context.Context, *sdkmcp.CallToolRequest) (*sdkmcp.CallToolResult, error){
		"echo": echoHandler,
	})

	gw := gateway.New([]gateway.BackendConfig{
		{Name: "svc1", Endpoint: backend.URL + "/mcp"},
	})
	ctx := t.Context()
	if err := gw.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer gw.Stop(context.Background())

	result, err := gw.CallTool(ctx, "nonexistent", nil)
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError for unknown tool")
	}
}

func mustJSON(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}
