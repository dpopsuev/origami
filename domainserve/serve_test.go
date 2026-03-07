package domainserve_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/dpopsuev/origami/domainserve"
)

func setup(t *testing.T, fsys fstest.MapFS) *sdkmcp.ClientSession {
	t.Helper()
	handler := domainserve.New(fsys, domainserve.Config{
		Name:    "test-domain",
		Version: "v0.1.0",
	})

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	ctx := t.Context()
	transport := &sdkmcp.StreamableClientTransport{Endpoint: srv.URL + "/mcp"}
	client := sdkmcp.NewClient(
		&sdkmcp.Implementation{Name: "test-client", Version: "v0.1.0"},
		nil,
	)
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { session.Close() })
	return session
}

func callTool(t *testing.T, session *sdkmcp.ClientSession, name string, args map[string]any) *sdkmcp.CallToolResult {
	t.Helper()
	result, err := session.CallTool(t.Context(), &sdkmcp.CallToolParams{
		Name:      name,
		Arguments: args,
	})
	if err != nil {
		t.Fatalf("CallTool(%s): %v", name, err)
	}
	return result
}

func resultText(result *sdkmcp.CallToolResult) string {
	for _, c := range result.Content {
		if tc, ok := c.(*sdkmcp.TextContent); ok {
			return tc.Text
		}
	}
	return ""
}

func TestDomainInfo(t *testing.T) {
	fs := fstest.MapFS{
		"circuits/rca.yaml": &fstest.MapFile{
			Data: []byte("circuit: rca\ntopology: cascade\ndescription: Root-cause analysis\n"),
		},
	}

	session := setup(t, fs)
	result := callTool(t, session, "domain_info", nil)

	if result.IsError {
		t.Fatalf("domain_info error: %s", resultText(result))
	}

	var info domainserve.DomainInfo
	if err := json.Unmarshal([]byte(resultText(result)), &info); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if info.Name != "test-domain" {
		t.Errorf("name = %q, want test-domain", info.Name)
	}
	if info.Version != "v0.1.0" {
		t.Errorf("version = %q, want v0.1.0", info.Version)
	}
	if len(info.Circuits) != 1 {
		t.Fatalf("circuits = %d, want 1", len(info.Circuits))
	}
	if info.Circuits[0].Name != "rca" {
		t.Errorf("circuit name = %q, want rca", info.Circuits[0].Name)
	}
	if info.Circuits[0].Topology != "cascade" {
		t.Errorf("topology = %q, want cascade", info.Circuits[0].Topology)
	}
	if info.Circuits[0].Description != "Root-cause analysis" {
		t.Errorf("description = %q, want Root-cause analysis", info.Circuits[0].Description)
	}
}

func TestDomainInfo_MultipleCircuits(t *testing.T) {
	fs := fstest.MapFS{
		"circuits/rca.yaml": &fstest.MapFile{
			Data: []byte("circuit: rca\ntopology: cascade\ndescription: RCA\n"),
		},
		"circuits/trend.yaml": &fstest.MapFile{
			Data: []byte("circuit: trend\ntopology: cascade\ndescription: Trend analysis\n"),
		},
	}

	session := setup(t, fs)
	result := callTool(t, session, "domain_info", nil)

	var info domainserve.DomainInfo
	if err := json.Unmarshal([]byte(resultText(result)), &info); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(info.Circuits) != 2 {
		t.Fatalf("circuits = %d, want 2", len(info.Circuits))
	}
}

func TestDomainInfo_NoCircuits(t *testing.T) {
	fs := fstest.MapFS{
		"prompts/hello.md": &fstest.MapFile{Data: []byte("hello")},
	}

	session := setup(t, fs)
	result := callTool(t, session, "domain_info", nil)

	var info domainserve.DomainInfo
	if err := json.Unmarshal([]byte(resultText(result)), &info); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if info.Circuits != nil {
		t.Errorf("circuits should be nil when circuits/ missing, got %v", info.Circuits)
	}
}

func TestDomainRead(t *testing.T) {
	fs := fstest.MapFS{
		"prompts/recall.md": &fstest.MapFile{Data: []byte("You are a recall judge.")},
	}

	session := setup(t, fs)
	result := callTool(t, session, "domain_read", map[string]any{"path": "prompts/recall.md"})

	if result.IsError {
		t.Fatalf("domain_read error: %s", resultText(result))
	}
	if resultText(result) != "You are a recall judge." {
		t.Errorf("content = %q, want %q", resultText(result), "You are a recall judge.")
	}
}

func TestDomainRead_Missing(t *testing.T) {
	session := setup(t, fstest.MapFS{})
	result := callTool(t, session, "domain_read", map[string]any{"path": "nope.txt"})

	if !result.IsError {
		t.Fatal("expected error for missing file")
	}
}

func TestDomainRead_PathTraversal(t *testing.T) {
	session := setup(t, fstest.MapFS{})

	for _, path := range []string{"../etc/passwd", "../../go.mod", "/absolute/path"} {
		result := callTool(t, session, "domain_read", map[string]any{"path": path})
		if !result.IsError {
			t.Errorf("expected error for path %q", path)
		}
	}
}

func TestDomainList(t *testing.T) {
	fs := fstest.MapFS{
		"prompts/recall.md": &fstest.MapFile{Data: []byte("a")},
		"prompts/triage.md": &fstest.MapFile{Data: []byte("b")},
	}

	session := setup(t, fs)
	result := callTool(t, session, "domain_list", map[string]any{"path": "prompts"})

	if result.IsError {
		t.Fatalf("domain_list error: %s", resultText(result))
	}

	var entries []domainserve.DirEntry
	if err := json.Unmarshal([]byte(resultText(result)), &entries); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("entries = %d, want 2", len(entries))
	}
}

func TestDomainList_Root(t *testing.T) {
	fs := fstest.MapFS{
		"circuits/rca.yaml": &fstest.MapFile{Data: []byte("a")},
		"prompts/x.md":      &fstest.MapFile{Data: []byte("b")},
		"vocabulary.yaml":   &fstest.MapFile{Data: []byte("c")},
	}

	session := setup(t, fs)
	result := callTool(t, session, "domain_list", map[string]any{"path": "."})

	if result.IsError {
		t.Fatalf("domain_list error: %s", resultText(result))
	}

	var entries []domainserve.DirEntry
	if err := json.Unmarshal([]byte(resultText(result)), &entries); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	hasDir := false
	hasFile := false
	for _, e := range entries {
		if e.Name == "circuits" && e.IsDir {
			hasDir = true
		}
		if e.Name == "vocabulary.yaml" && !e.IsDir {
			hasFile = true
		}
	}
	if !hasDir {
		t.Error("missing circuits/ directory in root listing")
	}
	if !hasFile {
		t.Error("missing vocabulary.yaml in root listing")
	}
}

func TestDomainList_PathTraversal(t *testing.T) {
	session := setup(t, fstest.MapFS{})
	result := callTool(t, session, "domain_list", map[string]any{"path": "../etc"})
	if !result.IsError {
		t.Error("expected error for traversal path")
	}
}

func TestHealth(t *testing.T) {
	handler := domainserve.New(fstest.MapFS{}, domainserve.Config{
		Name: "test", Version: "v0",
	})

	for _, path := range []string{"/healthz", "/readyz"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("%s returned %d, want 200", path, w.Code)
		}
	}
}

func callToolCtx(ctx context.Context, t *testing.T, session *sdkmcp.ClientSession, name string, args map[string]any) *sdkmcp.CallToolResult {
	t.Helper()
	result, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
		Name:      name,
		Arguments: args,
	})
	if err != nil {
		t.Fatalf("CallTool(%s): %v", name, err)
	}
	return result
}
