package sumi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	framework "github.com/dpopsuev/origami"
	"github.com/dpopsuev/origami/view"

	tea "github.com/charmbracelet/bubbletea"
)

func windowSizeMsg(w, h int) tea.WindowSizeMsg {
	return tea.WindowSizeMsg{Width: w, Height: h}
}

func TestIntegration_RecorderCapturesFrameOnDiff(t *testing.T) {
	def := &framework.CircuitDef{
		Circuit: "rec-test",
		Nodes: []framework.NodeDef{
			{Name: "alpha", Element: "fire"},
			{Name: "beta", Element: "water"},
		},
		Edges: []framework.EdgeDef{{From: "alpha", To: "beta"}},
		Start: "alpha",
		Done:  "beta",
	}

	store := view.NewCircuitStore(def)
	defer store.Close()

	engine := &view.GridLayout{}
	layout, _ := engine.Layout(def)

	recorder := NewViewRecorder(10)

	m := New(Config{
		Def:      def,
		Store:    store,
		Layout:   layout,
		Opts:     RenderOpts{},
		Recorder: recorder,
	})

	if recorder.Len() != 0 {
		t.Fatalf("expected 0 frames before any events, got %d", recorder.Len())
	}

	// WindowSizeMsg makes model ready and records a frame.
	updated, _ := m.Update(windowSizeMsg(120, 40))
	m = updated.(Model)

	if recorder.Len() != 1 {
		t.Fatalf("expected 1 frame after WindowSizeMsg, got %d", recorder.Len())
	}

	f := recorder.Latest()
	if f.Width != 120 || f.Height != 40 {
		t.Fatalf("expected 120x40, got %dx%d", f.Width, f.Height)
	}
	if f.ViewText == "" {
		t.Fatal("expected non-empty ViewText")
	}

	// DiffMsg records another frame.
	updated, _ = m.Update(DiffMsg(view.StateDiff{Type: view.DiffNodeState, Node: "alpha"}))
	m = updated.(Model)

	if recorder.Len() != 2 {
		t.Fatalf("expected 2 frames after DiffMsg, got %d", recorder.Len())
	}

	f2 := recorder.Latest()
	if !strings.Contains(f2.ViewText, "alpha") {
		t.Fatalf("expected ViewText to mention 'alpha', got: %s", f2.ViewText[:min(100, len(f2.ViewText))])
	}
}

func TestIntegration_RecorderNoColorOutput(t *testing.T) {
	def := &framework.CircuitDef{
		Circuit: "nocolor-test",
		Nodes:   []framework.NodeDef{{Name: "node1"}},
		Start:   "node1",
		Done:    "node1",
	}

	store := view.NewCircuitStore(def)
	defer store.Close()

	engine := &view.GridLayout{}
	layout, _ := engine.Layout(def)

	recorder := NewViewRecorder(5)
	m := New(Config{
		Def:      def,
		Store:    store,
		Layout:   layout,
		Opts:     RenderOpts{},
		Recorder: recorder,
	})

	updated, _ := m.Update(windowSizeMsg(120, 40))
	_ = updated.(Model)

	f := recorder.Latest()
	if f == nil {
		t.Fatal("expected recorded frame")
	}
	if strings.Contains(f.ViewText, "\x1b[") {
		t.Fatal("ViewText should not contain ANSI escape sequences")
	}
}

func TestIntegration_FramePushToKami(t *testing.T) {
	var received view.RecordedFrame
	var gotRequest bool

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/sumi/frame" && r.Method == "POST" {
			if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			gotRequest = true
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	recorder := NewViewRecorder(5)
	recorder.Record(view.RecordedFrame{
		Timestamp:    time.Now(),
		Width:        120,
		Height:       40,
		LayoutTier:   "full",
		SelectedNode: "triage",
		WorkerCount:  1,
		EventCount:   5,
		ViewText:     "test frame",
	})

	// Manually push a frame like framePushLoop would.
	f := recorder.Latest()
	body, _ := json.Marshal(f)
	addr := strings.TrimPrefix(ts.URL, "http://")
	resp, err := http.Post("http://"+addr+"/api/sumi/frame", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	resp.Body.Close()

	if !gotRequest {
		t.Fatal("expected server to receive frame")
	}
	if received.SelectedNode != "triage" {
		t.Fatalf("expected triage, got %s", received.SelectedNode)
	}
	if received.ViewText != "test frame" {
		t.Fatalf("expected 'test frame', got %q", received.ViewText)
	}
}

func TestIntegration_RecorderOnlyOnStateChange(t *testing.T) {
	def := &framework.CircuitDef{
		Circuit: "dirty-test",
		Nodes:   []framework.NodeDef{{Name: "a"}},
		Start:   "a",
		Done:    "a",
	}

	store := view.NewCircuitStore(def)
	defer store.Close()

	engine := &view.GridLayout{}
	layout, _ := engine.Layout(def)

	recorder := NewViewRecorder(20)
	m := New(Config{
		Def:      def,
		Store:    store,
		Layout:   layout,
		Opts:     RenderOpts{},
		Recorder: recorder,
	})

	// WindowSizeMsg records one frame.
	updated, _ := m.Update(windowSizeMsg(120, 40))
	m = updated.(Model)
	if recorder.Len() != 1 {
		t.Fatalf("expected 1 frame after WindowSizeMsg, got %d", recorder.Len())
	}

	// Calling View() without further state change does not record.
	m.View()
	m.View()
	if recorder.Len() != 1 {
		t.Fatalf("expected still 1 frame (no state change), got %d", recorder.Len())
	}

	// Another DiffMsg adds a second frame.
	updated, _ = m.Update(DiffMsg(view.StateDiff{Type: view.DiffNodeState, Node: "a"}))
	m = updated.(Model)
	if recorder.Len() != 2 {
		t.Fatalf("expected 2 frames after DiffMsg, got %d", recorder.Len())
	}
}

