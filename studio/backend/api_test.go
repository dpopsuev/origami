package backend

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAPI_ListRuns(t *testing.T) {
	store := NewEventStore()
	store.RegisterRun(RunInfo{ID: "r1", Circuit: "test", StartedAt: time.Now(), Status: "running"})
	store.RegisterRun(RunInfo{ID: "r2", Circuit: "test", StartedAt: time.Now(), Status: "completed"})

	api := NewAPI(store)
	req := httptest.NewRequest("GET", "/api/runs", nil)
	w := httptest.NewRecorder()
	api.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var runs []RunInfo
	json.NewDecoder(w.Body).Decode(&runs)
	if len(runs) != 2 {
		t.Errorf("expected 2 runs, got %d", len(runs))
	}
}

func TestAPI_GetRun(t *testing.T) {
	store := NewEventStore()
	store.RegisterRun(RunInfo{ID: "r1", Circuit: "test", StartedAt: time.Now(), Status: "running"})

	api := NewAPI(store)
	req := httptest.NewRequest("GET", "/api/runs/r1", nil)
	req.SetPathValue("runID", "r1")
	w := httptest.NewRecorder()
	api.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestAPI_GetRunNotFound(t *testing.T) {
	store := NewEventStore()
	api := NewAPI(store)

	req := httptest.NewRequest("GET", "/api/runs/missing", nil)
	req.SetPathValue("runID", "missing")
	w := httptest.NewRecorder()
	api.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestAPI_RunEvents(t *testing.T) {
	store := NewEventStore()
	store.Append(StudioEvent{RunID: "r1", Type: "node_enter", Node: "recall"})
	store.Append(StudioEvent{RunID: "r1", Type: "node_exit", Node: "recall"})
	store.Append(StudioEvent{RunID: "r2", Type: "node_enter", Node: "other"})

	api := NewAPI(store)
	req := httptest.NewRequest("GET", "/api/runs/r1/events", nil)
	req.SetPathValue("runID", "r1")
	w := httptest.NewRecorder()
	api.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var events []StudioEvent
	json.NewDecoder(w.Body).Decode(&events)
	if len(events) != 2 {
		t.Errorf("expected 2 events for r1, got %d", len(events))
	}
}

func TestAPI_ListCircuits(t *testing.T) {
	store := NewEventStore()
	store.RegisterRun(RunInfo{ID: "r1", Circuit: "circuit-a", StartedAt: time.Now(), Status: "running"})
	store.RegisterRun(RunInfo{ID: "r2", Circuit: "circuit-b", StartedAt: time.Now(), Status: "running"})
	store.RegisterRun(RunInfo{ID: "r3", Circuit: "circuit-a", StartedAt: time.Now(), Status: "completed"})

	api := NewAPI(store)
	req := httptest.NewRequest("GET", "/api/circuits", nil)
	w := httptest.NewRecorder()
	api.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var result struct {
		Circuits []string `json:"circuits"`
	}
	json.NewDecoder(w.Body).Decode(&result)
	if len(result.Circuits) != 2 {
		t.Errorf("expected 2 unique circuits, got %d", len(result.Circuits))
	}
}
