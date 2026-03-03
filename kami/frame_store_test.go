package kami

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dpopsuev/origami/view"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func sampleFrame() view.RecordedFrame {
	return view.RecordedFrame{
		Timestamp:    time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC),
		Width:        120,
		Height:       40,
		LayoutTier:   "wide",
		SelectedNode: "triage",
		FocusedPanel: "graph",
		WorkerCount:  2,
		EventCount:   15,
		ViewText:     "┌── Circuit ──┐\n│  [triage]   │\n└─────────────┘",
	}
}

func TestFrameStore_EmptyLatest(t *testing.T) {
	fs := NewFrameStore()
	if fs.Latest() != nil {
		t.Fatal("expected nil from empty store")
	}
}

func TestFrameStore_StoreAndRetrieve(t *testing.T) {
	fs := NewFrameStore()
	f := sampleFrame()
	fs.Store(f)

	got := fs.Latest()
	if got == nil {
		t.Fatal("expected non-nil frame")
	}
	if got.SelectedNode != "triage" {
		t.Fatalf("expected triage, got %s", got.SelectedNode)
	}
	if got.ViewText != f.ViewText {
		t.Fatalf("view text mismatch")
	}
}

func TestFrameStore_Overwrite(t *testing.T) {
	fs := NewFrameStore()
	fs.Store(sampleFrame())

	f2 := sampleFrame()
	f2.SelectedNode = "recall"
	f2.EventCount = 42
	fs.Store(f2)

	got := fs.Latest()
	if got.SelectedNode != "recall" {
		t.Fatalf("expected recall, got %s", got.SelectedNode)
	}
	if got.EventCount != 42 {
		t.Fatalf("expected 42 events, got %d", got.EventCount)
	}
}

func TestFrameStore_LatestReturnsCopy(t *testing.T) {
	fs := NewFrameStore()
	fs.Store(sampleFrame())
	a := fs.Latest()
	b := fs.Latest()
	a.SelectedNode = "mutated"
	if b.SelectedNode == "mutated" {
		t.Fatal("Latest() should return independent copies")
	}
}

func TestHTTP_StoreFrame(t *testing.T) {
	srv := NewServer(Config{Bridge: NewEventBridge(nil)})
	mux := srv.buildHTTPMux()

	body, _ := json.Marshal(sampleFrame())
	req := httptest.NewRequest("POST", "/api/sumi/frame", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}

	got := srv.frameStore.Latest()
	if got == nil {
		t.Fatal("frame not stored")
	}
	if got.SelectedNode != "triage" {
		t.Fatalf("expected triage, got %s", got.SelectedNode)
	}
}

func TestHTTP_StoreFrame_InvalidJSON(t *testing.T) {
	srv := NewServer(Config{Bridge: NewEventBridge(nil)})
	mux := srv.buildHTTPMux()

	req := httptest.NewRequest("POST", "/api/sumi/frame", strings.NewReader("not json"))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHTTP_GetFrame_Empty(t *testing.T) {
	srv := NewServer(Config{Bridge: NewEventBridge(nil)})
	mux := srv.buildHTTPMux()

	req := httptest.NewRequest("GET", "/api/sumi/frame", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestHTTP_GetFrame_WithData(t *testing.T) {
	srv := NewServer(Config{Bridge: NewEventBridge(nil)})
	srv.frameStore.Store(sampleFrame())
	mux := srv.buildHTTPMux()

	req := httptest.NewRequest("GET", "/api/sumi/frame", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var got view.RecordedFrame
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.SelectedNode != "triage" {
		t.Fatalf("expected triage, got %s", got.SelectedNode)
	}
}

func TestMCP_SumiGetView_NoFrame(t *testing.T) {
	_, srv := setupMCPTest()
	handler := handleGetSumiView(srv)
	res, _, err := handler(context.Background(), nil, emptyInput{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tc, ok := res.Content[0].(*sdkmcp.TextContent)
	if !ok {
		t.Fatalf("expected *TextContent, got %T", res.Content[0])
	}
	if !strings.Contains(tc.Text, "not connected") {
		t.Fatalf("expected 'not connected' message, got %q", tc.Text)
	}
}

func TestMCP_SumiGetView_WithFrame(t *testing.T) {
	_, srv := setupMCPTest()
	srv.frameStore.Store(sampleFrame())

	handler := handleGetSumiView(srv)
	res, _, err := handler(context.Background(), nil, emptyInput{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tc := res.Content[0].(*sdkmcp.TextContent)

	if !strings.Contains(tc.Text, "Selected node: triage") {
		t.Fatalf("missing selected node in header: %s", tc.Text)
	}
	if !strings.Contains(tc.Text, "120x40") {
		t.Fatalf("missing dimensions in header: %s", tc.Text)
	}
	if !strings.Contains(tc.Text, "Workers: 2") {
		t.Fatalf("missing worker count: %s", tc.Text)
	}
	if !strings.Contains(tc.Text, "Events: 15") {
		t.Fatalf("missing event count: %s", tc.Text)
	}
	if !strings.Contains(tc.Text, "---") {
		t.Fatalf("missing separator: %s", tc.Text)
	}
	if !strings.Contains(tc.Text, "[triage]") {
		t.Fatalf("missing view text: %s", tc.Text)
	}
}
