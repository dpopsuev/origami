package backend

import (
	"testing"
	"time"
)

func TestEventStore_AppendAndRetrieve(t *testing.T) {
	store := NewEventStore()
	store.Append(StudioEvent{RunID: "run-1", Type: "node_enter", Node: "recall"})
	store.Append(StudioEvent{RunID: "run-1", Type: "node_exit", Node: "recall"})
	store.Append(StudioEvent{RunID: "run-2", Type: "node_enter", Node: "triage"})

	events := store.Events("run-1")
	if len(events) != 2 {
		t.Errorf("expected 2 events for run-1, got %d", len(events))
	}

	events = store.Events("run-2")
	if len(events) != 1 {
		t.Errorf("expected 1 event for run-2, got %d", len(events))
	}
}

func TestEventStore_EventsSince(t *testing.T) {
	store := NewEventStore()
	store.Append(StudioEvent{RunID: "r1", Type: "a"})
	store.Append(StudioEvent{RunID: "r1", Type: "b"})
	store.Append(StudioEvent{RunID: "r1", Type: "c"})

	events := store.EventsSince("r1", 1)
	if len(events) != 2 {
		t.Errorf("expected 2 events after ID 1, got %d", len(events))
	}
}

func TestEventStore_Subscribe(t *testing.T) {
	store := NewEventStore()
	subID, ch := store.Subscribe()
	defer store.Unsubscribe(subID)

	go func() {
		store.Append(StudioEvent{RunID: "r1", Type: "test"})
	}()

	select {
	case evt := <-ch:
		if evt.Type != "test" {
			t.Errorf("expected type 'test', got %q", evt.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestEventStore_RunLifecycle(t *testing.T) {
	store := NewEventStore()
	store.RegisterRun(RunInfo{
		ID:        "r1",
		Circuit:  "test",
		StartedAt: time.Now(),
		Status:    "running",
		NodeCount: 3,
		EdgeCount: 2,
	})

	runs := store.Runs()
	if len(runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(runs))
	}
	if runs[0].Status != "running" {
		t.Errorf("expected status 'running', got %q", runs[0].Status)
	}

	store.CompleteRun("r1", "completed")

	r := store.Run("r1")
	if r == nil {
		t.Fatal("expected run r1")
	}
	if r.Status != "completed" {
		t.Errorf("expected status 'completed', got %q", r.Status)
	}
	if r.EndedAt.IsZero() {
		t.Error("expected EndedAt to be set")
	}
}

func TestEventStore_RunNotFound(t *testing.T) {
	store := NewEventStore()
	if r := store.Run("nonexistent"); r != nil {
		t.Error("expected nil for nonexistent run")
	}
}
