package backend

import (
	"fmt"
	"testing"
	"time"

	framework "github.com/dpopsuev/origami"
)

func TestStudioObserver_RecordsEvents(t *testing.T) {
	store := NewEventStore()
	obs := NewStudioObserver(store, "run-1", "test-pipeline")

	obs.OnEvent(framework.WalkEvent{
		Type:   "node_enter",
		Node:   "recall",
		Walker: "herald",
	})

	obs.OnEvent(framework.WalkEvent{
		Type:    "transition",
		Edge:    "E1",
		Elapsed: 150 * time.Millisecond,
	})

	events := store.Events("run-1")
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}

	if events[0].Type != "node_enter" {
		t.Errorf("expected first event type 'node_enter', got %q", events[0].Type)
	}
	if events[0].Node != "recall" {
		t.Errorf("expected node 'recall', got %q", events[0].Node)
	}
	if events[0].Walker != "herald" {
		t.Errorf("expected walker 'herald', got %q", events[0].Walker)
	}

	if events[1].ElapsedMs != 150 {
		t.Errorf("expected elapsed_ms=150, got %d", events[1].ElapsedMs)
	}
}

func TestStudioObserver_RecordsErrors(t *testing.T) {
	store := NewEventStore()
	obs := NewStudioObserver(store, "run-1", "test")

	obs.OnEvent(framework.WalkEvent{
		Type:  "walk_error",
		Error: fmt.Errorf("context deadline exceeded"),
	})

	events := store.Events("run-1")
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Error != "context deadline exceeded" {
		t.Errorf("unexpected error: %q", events[0].Error)
	}
}

func TestStudioObserver_Accessors(t *testing.T) {
	store := NewEventStore()
	obs := NewStudioObserver(store, "run-42", "my-pipeline")

	if obs.RunID() != "run-42" {
		t.Errorf("RunID() = %q, want 'run-42'", obs.RunID())
	}
	if obs.Pipeline() != "my-pipeline" {
		t.Errorf("Pipeline() = %q, want 'my-pipeline'", obs.Pipeline())
	}
}
