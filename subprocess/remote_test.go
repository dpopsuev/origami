package subprocess_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/dpopsuev/origami/subprocess"
)

func newTestMCPServer(t *testing.T) *httptest.Server {
	t.Helper()

	server := sdkmcp.NewServer(
		&sdkmcp.Implementation{Name: "test-remote", Version: "v0.1.0"},
		nil,
	)
	server.AddTool(
		&sdkmcp.Tool{
			Name:        "echo",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"message":{"type":"string"}}}`),
		},
		func(_ context.Context, req *sdkmcp.CallToolRequest) (*sdkmcp.CallToolResult, error) {
			var args struct{ Message string }
			json.Unmarshal(req.Params.Arguments, &args)
			return &sdkmcp.CallToolResult{
				Content: []sdkmcp.Content{&sdkmcp.TextContent{Text: "echo: " + args.Message}},
			}, nil
		},
	)
	server.AddTool(
		&sdkmcp.Tool{
			Name:        "add",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"a":{"type":"integer"},"b":{"type":"integer"}}}`),
		},
		func(_ context.Context, req *sdkmcp.CallToolRequest) (*sdkmcp.CallToolResult, error) {
			var args struct {
				A int `json:"a"`
				B int `json:"b"`
			}
			json.Unmarshal(req.Params.Arguments, &args)
			return &sdkmcp.CallToolResult{
				Content: []sdkmcp.Content{&sdkmcp.TextContent{Text: fmt.Sprintf("%d", args.A+args.B)}},
			}, nil
		},
	)

	handler := sdkmcp.NewStreamableHTTPHandler(
		func(_ *http.Request) *sdkmcp.Server { return server },
		&sdkmcp.StreamableHTTPOptions{Stateless: true},
	)
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)
	return ts
}

func newRemoteBackend(endpoint string) *subprocess.RemoteBackend {
	return &subprocess.RemoteBackend{
		Endpoint:  endpoint,
		Connector: subprocess.DefaultConnector(),
	}
}

func TestRemoteBackend_ToolCallRoundTrip(t *testing.T) {
	ts := newTestMCPServer(t)
	ctx := t.Context()

	rb := newRemoteBackend(ts.URL)
	if err := rb.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { rb.Stop(context.Background()) })

	result, err := rb.CallTool(ctx, "echo", map[string]any{"message": "hello"})
	if err != nil {
		t.Fatalf("CallTool echo: %v", err)
	}
	got := extractText(t, result)
	if got != "echo: hello" {
		t.Errorf("echo returned %q, want %q", got, "echo: hello")
	}

	result, err = rb.CallTool(ctx, "add", map[string]any{"a": 3, "b": 7})
	if err != nil {
		t.Fatalf("CallTool add: %v", err)
	}
	got = extractText(t, result)
	if got != "10" {
		t.Errorf("add returned %q, want %q", got, "10")
	}
}

func TestRemoteBackend_Healthy(t *testing.T) {
	ts := newTestMCPServer(t)
	ctx := t.Context()

	rb := newRemoteBackend(ts.URL)

	if rb.Healthy(ctx) {
		t.Error("expected unhealthy before Start")
	}

	if err := rb.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { rb.Stop(context.Background()) })

	if !rb.Healthy(ctx) {
		t.Error("expected healthy after Start")
	}
}

func TestRemoteBackend_StopIdempotent(t *testing.T) {
	ts := newTestMCPServer(t)
	ctx := t.Context()

	rb := newRemoteBackend(ts.URL)
	if err := rb.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	if err := rb.Stop(ctx); err != nil {
		t.Fatalf("first Stop: %v", err)
	}

	if err := rb.Stop(ctx); err != nil {
		t.Fatalf("second Stop: %v", err)
	}

	if rb.Healthy(ctx) {
		t.Error("expected unhealthy after Stop")
	}
}

func TestRemoteBackend_CallToolAfterStop(t *testing.T) {
	ts := newTestMCPServer(t)
	ctx := t.Context()

	rb := newRemoteBackend(ts.URL)
	if err := rb.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := rb.Stop(ctx); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	_, err := rb.CallTool(ctx, "echo", map[string]any{"message": "hello"})
	if err == nil {
		t.Fatal("expected error calling tool after Stop")
	}
}

func TestRemoteBackend_DoubleStart(t *testing.T) {
	ts := newTestMCPServer(t)
	ctx := t.Context()

	rb := newRemoteBackend(ts.URL)
	if err := rb.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { rb.Stop(context.Background()) })

	err := rb.Start(ctx)
	if err == nil {
		t.Fatal("expected error on double Start")
	}
}

func TestRemoteBackend_CallToolBeforeStart(t *testing.T) {
	rb := newRemoteBackend("http://127.0.0.1:19999")

	_, err := rb.CallTool(context.Background(), "echo", map[string]any{"message": "hello"})
	if err == nil {
		t.Fatal("expected error calling tool before Start")
	}
}

func TestRemoteBackend_MultipleConcurrentCalls(t *testing.T) {
	ts := newTestMCPServer(t)
	ctx := t.Context()

	rb := newRemoteBackend(ts.URL)
	if err := rb.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { rb.Stop(context.Background()) })

	const n = 10
	errs := make(chan error, n)
	for i := range n {
		go func(i int) {
			result, err := rb.CallTool(ctx, "add", map[string]any{"a": i, "b": 1})
			if err != nil {
				errs <- fmt.Errorf("call %d: %w", i, err)
				return
			}
			want := fmt.Sprintf("%d", i+1)
			if got := extractTextNoFail(result); got != want {
				errs <- fmt.Errorf("call %d: got %q, want %q", i, got, want)
				return
			}
			errs <- nil
		}(i)
	}

	for range n {
		if err := <-errs; err != nil {
			t.Error(err)
		}
	}
}

func TestRemoteBackend_StartFailsOnBadEndpoint(t *testing.T) {
	rb := newRemoteBackend("http://127.0.0.1:19999/mcp")
	rb.HTTPTimeout = 1 * time.Second

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := rb.Start(ctx)
	if err == nil {
		t.Fatal("expected error connecting to unreachable endpoint")
	}
}
