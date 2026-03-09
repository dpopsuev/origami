package gateway_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/dpopsuev/origami/dispatch"
	"github.com/dpopsuev/origami/gateway"
	"github.com/dpopsuev/origami/mcp"
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

// --- Gap 2: MCP routing gap — tools reachable via HTTP ---

func newCircuitBackend(t *testing.T) (*httptest.Server, *mcp.CircuitServer) {
	t.Helper()
	cfg := mcp.CircuitConfig{
		Name:        "test-circuit",
		Version:     "dev",
		StepSchemas: []mcp.StepSchema{
			{
				Name:   "STEP_A",
				Fields: map[string]string{"value": "string", "score": "float"},
				Defs: []mcp.FieldDef{
					{Name: "value", Type: "string", Required: true},
					{Name: "score", Type: "float", Required: true},
				},
			},
		},
		DefaultGetNextStepTimeout: 5000,
		DefaultSessionTTL:         300000,
		CreateSession: func(ctx context.Context, params mcp.StartParams, disp *dispatch.MuxDispatcher, bus *dispatch.SignalBus) (mcp.RunFunc, mcp.SessionMeta, error) {
			nCases := 2
			return func(ctx context.Context) (any, error) {
				for i := 0; i < nCases; i++ {
					caseID := fmt.Sprintf("C%02d", i+1)
					if _, err := disp.Dispatch(ctx, dispatch.DispatchContext{
						CaseID: caseID, Step: "STEP_A",
					}); err != nil {
						return nil, err
					}
				}
				return map[string]any{"cases": nCases}, nil
			}, mcp.SessionMeta{TotalCases: nCases, Scenario: "http-test"}, nil
		},
		FormatReport: func(result any) (string, any, error) {
			return "Processed 2 cases", result, nil
		},
	}

	srv := mcp.NewCircuitServer(cfg)
	t.Cleanup(srv.Shutdown)

	h := sdkmcp.NewStreamableHTTPHandler(
		func(_ *http.Request) *sdkmcp.Server { return srv.MCPServer },
		&sdkmcp.StreamableHTTPOptions{Stateless: true},
	)
	ts := httptest.NewServer(h)
	t.Cleanup(ts.Close)
	return ts, srv
}

func TestGateway_MCPToolsReachableViaHTTP(t *testing.T) {
	backend, _ := newCircuitBackend(t)

	gw := gateway.New([]gateway.BackendConfig{
		{Name: "rca", Endpoint: backend.URL + "/mcp"},
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

	wantTools := map[string]bool{
		"start_circuit": false,
		"get_next_step": false,
		"submit_step":   false,
		"get_report":    false,
	}
	for _, tool := range tools.Tools {
		if _, ok := wantTools[tool.Name]; ok {
			wantTools[tool.Name] = true
		}
	}
	for name, found := range wantTools {
		if !found {
			t.Errorf("circuit tool %q not found via HTTP gateway", name)
		}
	}
}

func TestWorker_CanCallToolsViaHTTPTransport(t *testing.T) {
	backend, _ := newCircuitBackend(t)

	gw := gateway.New([]gateway.BackendConfig{
		{Name: "rca", Endpoint: backend.URL + "/mcp"},
	})
	ctx := t.Context()
	if err := gw.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer gw.Stop(context.Background())

	ts := httptest.NewServer(gw.Handler())
	defer ts.Close()

	session := connectGateway(t, ts)

	// Start circuit
	startRes, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
		Name:      "start_circuit",
		Arguments: mustJSON(map[string]any{"parallel": 1}),
	})
	if err != nil {
		t.Fatalf("start_circuit: %v", err)
	}
	var startOut map[string]any
	for _, c := range startRes.Content {
		if tc, ok := c.(*sdkmcp.TextContent); ok {
			json.Unmarshal([]byte(tc.Text), &startOut)
		}
	}
	sessionID, _ := startOut["session_id"].(string)
	if sessionID == "" {
		t.Fatal("no session_id in start_circuit response")
	}

	// Worker loop via HTTP
	stepsProcessed := 0
	for {
		res, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
			Name:      "get_next_step",
			Arguments: mustJSON(map[string]any{"session_id": sessionID, "timeout_ms": 5000}),
		})
		if err != nil {
			t.Fatalf("get_next_step: %v", err)
		}
		var out map[string]any
		for _, c := range res.Content {
			if tc, ok := c.(*sdkmcp.TextContent); ok {
				json.Unmarshal([]byte(tc.Text), &out)
			}
		}

		if done, _ := out["done"].(bool); done {
			break
		}
		if avail, _ := out["available"].(bool); !avail {
			continue
		}

		dispatchID := int64(out["dispatch_id"].(float64))
		step, _ := out["step"].(string)

		_, err = session.CallTool(ctx, &sdkmcp.CallToolParams{
			Name: "submit_step",
			Arguments: mustJSON(map[string]any{
				"session_id":  sessionID,
				"dispatch_id": dispatchID,
				"step":        step,
				"fields":      map[string]any{"value": "http-worker", "score": 0.95},
			}),
		})
		if err != nil {
			t.Fatalf("submit_step: %v", err)
		}
		stepsProcessed++
	}

	if stepsProcessed != 2 {
		t.Fatalf("processed %d steps via HTTP, want 2", stepsProcessed)
	}
}
