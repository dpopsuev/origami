package sumi

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
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
