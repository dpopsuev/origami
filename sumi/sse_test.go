package sumi

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	framework "github.com/dpopsuev/origami"
	"github.com/dpopsuev/origami/kami"
	"github.com/dpopsuev/origami/view"
)

func testDef() *framework.CircuitDef {
	return &framework.CircuitDef{
		Circuit: "test",
		Nodes: []framework.NodeDef{
			{Name: "recall"},
			{Name: "triage"},
		},
	}
}

func quietLog() *slog.Logger {
	return slog.New(slog.NewTextHandler(nil, &slog.HandlerOptions{Level: slog.LevelError + 1}))
}

type logEntry struct {
	Level string
	Msg   string
	Attrs map[string]string
}

type logSink struct {
	mu      sync.Mutex
	entries []logEntry
}

func (s *logSink) append(e logEntry) {
	s.mu.Lock()
	s.entries = append(s.entries, e)
	s.mu.Unlock()
}

func (s *logSink) snapshot() []logEntry {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := make([]logEntry, len(s.entries))
	copy(cp, s.entries)
	return cp
}

func capturingLog() (*slog.Logger, *logSink) {
	sink := &logSink{}
	handler := &captureHandler{sink: sink}
	return slog.New(handler), sink
}

type captureHandler struct {
	sink  *logSink
	attrs []slog.Attr
}

func (h *captureHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }
func (h *captureHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &captureHandler{sink: h.sink, attrs: append(h.attrs, attrs...)}
}
func (h *captureHandler) WithGroup(_ string) slog.Handler { return h }
func (h *captureHandler) Handle(_ context.Context, r slog.Record) error {
	e := logEntry{
		Level: r.Level.String(),
		Msg:   r.Message,
		Attrs: make(map[string]string),
	}
	for _, a := range h.attrs {
		e.Attrs[a.Key] = a.Value.String()
	}
	r.Attrs(func(a slog.Attr) bool {
		e.Attrs[a.Key] = a.Value.String()
		return true
	})
	h.sink.append(e)
	return nil
}

func TestSSEClient_ReceivesEvents(t *testing.T) {
	store := view.NewCircuitStore(testDef())
	defer store.Close()

	id, ch := store.Subscribe()
	defer store.Unsubscribe(id)

	evt := kami.Event{
		Type:      kami.EventNodeEnter,
		Node:      "recall",
		Agent:     "w1",
		Timestamp: time.Now(),
	}
	data, _ := json.Marshal(evt)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		fmt.Fprintf(w, "data: %s\n\n", data)
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	defer ts.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	addr := ts.Listener.Addr().String()
	go sseClientLoop(ctx, addr, store, quietLog())

	select {
	case diff := <-ch:
		if diff.Node != "recall" {
			t.Errorf("Node = %q, want recall", diff.Node)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for store event from SSE client")
	}
}

func TestSSEClient_ReconnectsOnClose(t *testing.T) {
	store := view.NewCircuitStore(testDef())
	defer store.Close()

	id, ch := store.Subscribe()
	defer store.Unsubscribe(id)

	var connectCount atomic.Int32

	evt := kami.Event{
		Type:      kami.EventNodeEnter,
		Node:      "triage",
		Agent:     "w1",
		Timestamp: time.Now(),
	}
	data, _ := json.Marshal(evt)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		connectCount.Add(1)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		fmt.Fprintf(w, "data: %s\n\n", data)
		w.(http.Flusher).Flush()
		// Close immediately to trigger reconnect.
	}))
	defer ts.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	addr := ts.Listener.Addr().String()
	go sseClientLoop(ctx, addr, store, quietLog())

	received := 0
	timeout := time.After(5 * time.Second)
	for received < 3 {
		select {
		case <-ch:
			received++
		case <-timeout:
			t.Fatalf("received only %d events, wanted at least 3 (reconnect test)", received)
		}
	}

	connects := connectCount.Load()
	if connects < 2 {
		t.Errorf("expected at least 2 connections (reconnect), got %d", connects)
	}
	t.Logf("received %d store diffs across %d connections", received, connects)
}

func TestSSEClient_ContextCancellation(t *testing.T) {
	store := view.NewCircuitStore(testDef())
	defer store.Close()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	defer ts.Close()

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		sseClientLoop(ctx, ts.Listener.Addr().String(), store, quietLog())
		close(done)
	}()

	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("sseClientLoop did not exit after context cancellation")
	}
}

func TestSSEClient_ErrorStatus(t *testing.T) {
	store := view.NewCircuitStore(testDef())
	defer store.Close()

	var connectCount atomic.Int32

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := connectCount.Add(1)
		if n <= 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		evt := kami.Event{Type: kami.EventNodeEnter, Node: "recall", Agent: "w1"}
		data, _ := json.Marshal(evt)
		fmt.Fprintf(w, "data: %s\n\n", data)
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	defer ts.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	id, ch := store.Subscribe()
	defer store.Unsubscribe(id)

	go sseClientLoop(ctx, ts.Listener.Addr().String(), store, quietLog())

	select {
	case <-ch:
		connects := connectCount.Load()
		if connects < 3 {
			t.Errorf("expected at least 3 attempts, got %d", connects)
		}
		t.Logf("recovered after %d connection attempts", connects)
	case <-time.After(10 * time.Second):
		t.Fatal("SSE client did not recover from error status")
	}
}

// --- Instrumentation tests ---
// These verify that the SSE pipeline emits structured log entries at key
// decision points, enabling debug-mode diagnosis of live runs.

func TestSSE_Logging_StreamConnectAndEvent(t *testing.T) {
	store := view.NewCircuitStore(testDef())
	defer store.Close()

	evt := kami.Event{
		Type:  kami.EventNodeEnter,
		Node:  "recall",
		Agent: "w1",
	}
	data, _ := json.Marshal(evt)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		fmt.Fprintf(w, "data: %s\n\n", data)
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	defer ts.Close()

	ctx, cancel := context.WithCancel(context.Background())

	log, entries := capturingLog()
	id, ch := store.Subscribe()
	defer store.Unsubscribe(id)

	go sseClientLoop(ctx, ts.Listener.Addr().String(), store, log)

	select {
	case <-ch:
	case <-time.After(3 * time.Second):
		t.Fatal("timeout")
	}
	cancel()
	time.Sleep(100 * time.Millisecond)

	// Should have logged SSE connection and event receipt.
	captured := entries.snapshot()
	hasConnect := false
	hasEvent := false
	for _, e := range captured {
		if e.Msg == "SSE connected" {
			hasConnect = true
		}
		if e.Msg == "SSE event received" {
			hasEvent = true
		}
	}
	if !hasConnect {
		t.Error("missing 'SSE connected' log entry — streamSSE should log on successful connect")
	}
	if !hasEvent {
		t.Error("missing 'SSE event received' log entry — streamSSE should log each event")
	}
	t.Logf("captured %d log entries", len(captured))
}

func TestSSE_Logging_RebootstrapSuccess(t *testing.T) {
	store := view.NewCircuitStore(testDef())
	defer store.Close()

	bridge := kami.NewEventBridge(nil)
	defer bridge.Close()
	srv := kami.NewServer(kami.Config{Bridge: bridge, Store: store})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	httpAddr, _, err := srv.StartOnAvailablePort(ctx)
	if err != nil {
		t.Fatalf("kami start: %v", err)
	}

	clientStore := view.NewCircuitStore(&framework.CircuitDef{Circuit: "empty"})
	defer clientStore.Close()

	log, entries := capturingLog()
	rebootstrapStore(httpAddr, clientStore, log)

	hasRebootstrap := false
	for _, e := range entries.snapshot() {
		if e.Msg == "re-bootstrapped store from snapshot" {
			hasRebootstrap = true
			if e.Attrs["circuit"] != "test" {
				t.Errorf("circuit = %q, want 'test'", e.Attrs["circuit"])
			}
			if e.Attrs["nodes"] != "2" {
				t.Errorf("nodes = %q, want '2'", e.Attrs["nodes"])
			}
		}
	}
	if !hasRebootstrap {
		t.Error("missing 're-bootstrapped store from snapshot' log entry")
	}
}

func TestSSE_Logging_RebootstrapFailure(t *testing.T) {
	clientStore := view.NewCircuitStore(&framework.CircuitDef{Circuit: "empty"})
	defer clientStore.Close()

	log, entries := capturingLog()
	rebootstrapStore("127.0.0.1:1", clientStore, log)

	hasFailure := false
	for _, e := range entries.snapshot() {
		if e.Msg == "re-bootstrap snapshot unavailable" {
			hasFailure = true
		}
	}
	if !hasFailure {
		t.Error("missing 're-bootstrap snapshot unavailable' log entry on connection failure")
	}
}

func TestSSE_Logging_ReconnectCycle(t *testing.T) {
	store := view.NewCircuitStore(testDef())
	defer store.Close()

	var connectCount atomic.Int32

	evt := kami.Event{
		Type:  kami.EventNodeEnter,
		Node:  "recall",
		Agent: "w1",
	}
	data, _ := json.Marshal(evt)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := connectCount.Add(1)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		fmt.Fprintf(w, "data: %s\n\n", data)
		w.(http.Flusher).Flush()
		if n < 3 {
			return // close to trigger reconnect
		}
		<-r.Context().Done()
	}))
	defer ts.Close()

	ctx, cancel := context.WithCancel(context.Background())

	log, entries := capturingLog()
	id, ch := store.Subscribe()
	defer store.Unsubscribe(id)

	go sseClientLoop(ctx, ts.Listener.Addr().String(), store, log)

	received := 0
	timeout := time.After(5 * time.Second)
	for received < 3 {
		select {
		case <-ch:
			received++
		case <-timeout:
			t.Fatalf("timeout with %d events", received)
		}
	}
	cancel()
	time.Sleep(200 * time.Millisecond)

	// Count log entries by message type.
	counts := map[string]int{}
	for _, e := range entries.snapshot() {
		counts[e.Msg]++
	}

	// Should see at least 1 reconnect attempt (SSE reconnecting log).
	if counts["SSE reconnecting"] < 1 {
		t.Error("expected at least 1 'SSE reconnecting' log")
	}

	// Should see at least 2 connections.
	if counts["SSE connected"] < 2 {
		t.Errorf("expected >= 2 SSE connections, got %d", counts["SSE connected"])
	}

	// Should see re-bootstrap attempts on reconnects (not first connect).
	// The test server doesn't serve /api/snapshot, so re-bootstrap
	// attempts produce decode failures, which still proves the attempt was made.
	rebootstrapAttempts := counts["re-bootstrapped store from snapshot"] +
		counts["re-bootstrap snapshot decode failed"] +
		counts["re-bootstrap snapshot unavailable"] +
		counts["re-bootstrap snapshot non-200"]
	if rebootstrapAttempts < 1 {
		t.Error("expected at least 1 re-bootstrap attempt on reconnect")
	}

	t.Logf("log counts: %v", counts)
}
