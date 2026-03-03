package sumi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	framework "github.com/dpopsuev/origami"
	"github.com/dpopsuev/origami/kami"
	"github.com/dpopsuev/origami/view"
)

func rcaDef() *framework.CircuitDef {
	return &framework.CircuitDef{
		Circuit: "asterisk-rca",
		Nodes: []framework.NodeDef{
			{Name: "recall"},
			{Name: "triage"},
			{Name: "resolve"},
			{Name: "investigate"},
			{Name: "correlate"},
			{Name: "review"},
			{Name: "report"},
		},
	}
}

// TestWatch_EmptyDef_DropsNodeStateEvents demonstrates the bug:
// when runWatch creates an empty CircuitDef, node_enter SSE events
// update walker position but NOT node state, resulting in "(empty circuit)".
func TestWatch_EmptyDef_DropsNodeStateEvents(t *testing.T) {
	emptyDef := &framework.CircuitDef{Circuit: "watch"}
	clientStore := view.NewCircuitStore(emptyDef)
	defer clientStore.Close()

	id, ch := clientStore.Subscribe()
	defer clientStore.Unsubscribe(id)

	evt := kami.Event{
		Type:      kami.EventNodeEnter,
		Node:      "recall",
		Agent:     "C08",
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
	go sseClientLoop(ctx, ts.Listener.Addr().String(), clientStore, quietLog())

	select {
	case <-ch:
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for SSE event")
	}

	snap := clientStore.Snapshot()

	if len(snap.Nodes) != 0 {
		t.Errorf("expected 0 nodes in empty def store, got %d", len(snap.Nodes))
	}

	_, recallExists := snap.Nodes["recall"]
	if recallExists {
		t.Error("recall should NOT appear in node map of empty def store")
	}

	wp, ok := snap.Walkers["C08"]
	if !ok {
		t.Fatal("walker C08 should exist (walkers are added dynamically)")
	}
	if wp.Node != "recall" {
		t.Errorf("walker at %q, want recall", wp.Node)
	}

	t.Log("BUG: Sumi watch mode sees walker position but no circuit nodes — " +
		"this is why it shows '(empty circuit)' with 'Walker: C08 @ recall'")
}

// TestWatch_ProperDef_NodeStateUpdates verifies that with a real
// circuit definition, SSE events correctly update node states.
func TestWatch_ProperDef_NodeStateUpdates(t *testing.T) {
	def := rcaDef()
	clientStore := view.NewCircuitStore(def)
	defer clientStore.Close()

	id, ch := clientStore.Subscribe()
	defer clientStore.Unsubscribe(id)

	events := []kami.Event{
		{Type: kami.EventNodeEnter, Node: "recall", Agent: "C08", Timestamp: time.Now()},
		{Type: kami.EventNodeExit, Node: "recall", Timestamp: time.Now()},
		{Type: kami.EventNodeEnter, Node: "triage", Agent: "C08", Timestamp: time.Now()},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		for _, e := range events {
			data, _ := json.Marshal(e)
			fmt.Fprintf(w, "data: %s\n\n", data)
		}
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	defer ts.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go sseClientLoop(ctx, ts.Listener.Addr().String(), clientStore, quietLog())

	received := 0
	timeout := time.After(3 * time.Second)
	for received < 4 {
		select {
		case <-ch:
			received++
		case <-timeout:
			t.Fatalf("received only %d diffs, wanted at least 4", received)
		}
	}

	snap := clientStore.Snapshot()

	if snap.Nodes["recall"].State != view.NodeCompleted {
		t.Errorf("recall state = %q, want completed", snap.Nodes["recall"].State)
	}
	if snap.Nodes["triage"].State != view.NodeActive {
		t.Errorf("triage state = %q, want active", snap.Nodes["triage"].State)
	}
	if snap.Walkers["C08"].Node != "triage" {
		t.Errorf("walker C08 at %q, want triage", snap.Walkers["C08"].Node)
	}

	for _, name := range []string{"resolve", "investigate", "correlate", "review", "report"} {
		if snap.Nodes[name].State != view.NodeIdle {
			t.Errorf("unvisited node %q state = %q, want idle", name, snap.Nodes[name].State)
		}
	}
}

// TestWatch_MultipleWalkers verifies concurrent walker tracking via SSE.
func TestWatch_MultipleWalkers(t *testing.T) {
	def := rcaDef()
	clientStore := view.NewCircuitStore(def)
	defer clientStore.Close()

	id, ch := clientStore.Subscribe()
	defer clientStore.Unsubscribe(id)

	events := []kami.Event{
		{Type: kami.EventNodeEnter, Node: "recall", Agent: "C04"},
		{Type: kami.EventNodeEnter, Node: "triage", Agent: "C05"},
		{Type: kami.EventNodeEnter, Node: "investigate", Agent: "C08"},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		for _, e := range events {
			data, _ := json.Marshal(e)
			fmt.Fprintf(w, "data: %s\n\n", data)
		}
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	defer ts.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go sseClientLoop(ctx, ts.Listener.Addr().String(), clientStore, quietLog())

	received := 0
	timeout := time.After(3 * time.Second)
	for received < 6 {
		select {
		case <-ch:
			received++
		case <-timeout:
			t.Fatalf("received only %d diffs, wanted at least 6", received)
		}
	}

	snap := clientStore.Snapshot()
	if len(snap.Walkers) != 3 {
		t.Fatalf("expected 3 walkers, got %d", len(snap.Walkers))
	}

	expected := map[string]string{"C04": "recall", "C05": "triage", "C08": "investigate"}
	for id, node := range expected {
		wp, ok := snap.Walkers[id]
		if !ok {
			t.Errorf("walker %s missing", id)
			continue
		}
		if wp.Node != node {
			t.Errorf("walker %s at %q, want %q", id, wp.Node, node)
		}
	}
}

// TestWatch_FullCircuitTraversal verifies a complete case traversal
// produces correct final state in the client store.
func TestWatch_FullCircuitTraversal(t *testing.T) {
	def := rcaDef()
	clientStore := view.NewCircuitStore(def)
	defer clientStore.Close()

	id, ch := clientStore.Subscribe()
	defer clientStore.Unsubscribe(id)

	allNodes := []string{"recall", "triage", "resolve", "investigate", "correlate", "review", "report"}
	var events []kami.Event
	for _, node := range allNodes {
		events = append(events,
			kami.Event{Type: kami.EventNodeEnter, Node: node, Agent: "C04"},
			kami.Event{Type: kami.EventNodeExit, Node: node},
		)
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		for _, e := range events {
			data, _ := json.Marshal(e)
			fmt.Fprintf(w, "data: %s\n\n", data)
		}
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	defer ts.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go sseClientLoop(ctx, ts.Listener.Addr().String(), clientStore, quietLog())

	received := 0
	timeout := time.After(5 * time.Second)
	for received < 21 {
		select {
		case <-ch:
			received++
		case <-timeout:
			t.Fatalf("received only %d diffs, expected 21 (7 enters x2 diffs + 7 exits x1 diff)", received)
		}
	}

	snap := clientStore.Snapshot()
	for _, node := range allNodes {
		ns := snap.Nodes[node]
		if ns.State != view.NodeCompleted {
			t.Errorf("node %q state = %q, want completed", node, ns.State)
		}
	}

	wp := snap.Walkers["C04"]
	if wp.Node != "report" {
		t.Errorf("walker C04 at %q, want report (last node)", wp.Node)
	}
}

// TestWatch_SnapshotBootstrap verifies that Sumi can fetch /api/snapshot
// to bootstrap its local store with the correct circuit definition.
func TestWatch_SnapshotBootstrap(t *testing.T) {
	serverDef := rcaDef()
	serverStore := view.NewCircuitStore(serverDef)
	defer serverStore.Close()

	serverStore.OnEvent(framework.WalkEvent{Type: framework.EventNodeEnter, Node: "recall", Walker: "C08"})

	bridge := kami.NewEventBridge(nil)
	defer bridge.Close()
	srv := kami.NewServer(kami.Config{Bridge: bridge, Store: serverStore})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	httpAddr, _, err := srv.StartOnAvailablePort(ctx)
	if err != nil {
		t.Fatalf("start: %v", err)
	}

	resp, err := http.Get(fmt.Sprintf("http://%s/api/snapshot", httpAddr))
	if err != nil {
		t.Fatalf("GET /api/snapshot: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("snapshot returned %d, want 200", resp.StatusCode)
	}

	var snap view.CircuitSnapshot
	if err := json.NewDecoder(resp.Body).Decode(&snap); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if snap.CircuitName != "asterisk-rca" {
		t.Errorf("circuit name = %q, want asterisk-rca", snap.CircuitName)
	}
	if len(snap.Nodes) != 7 {
		t.Fatalf("expected 7 nodes, got %d", len(snap.Nodes))
	}
	if snap.Nodes["recall"].State != view.NodeActive {
		t.Errorf("recall state = %q, want active", snap.Nodes["recall"].State)
	}
	wp, ok := snap.Walkers["C08"]
	if !ok {
		t.Fatal("walker C08 missing from snapshot")
	}
	if wp.Node != "recall" {
		t.Errorf("walker at %q, want recall", wp.Node)
	}

	clientDef := &framework.CircuitDef{
		Circuit: snap.CircuitName,
	}
	for name := range snap.Nodes {
		clientDef.Nodes = append(clientDef.Nodes, framework.NodeDef{Name: name})
	}

	clientStore := view.NewCircuitStore(clientDef)
	defer clientStore.Close()

	clientSnap := clientStore.Snapshot()
	if len(clientSnap.Nodes) != 7 {
		t.Errorf("client store has %d nodes, want 7", len(clientSnap.Nodes))
	}

	for name := range snap.Nodes {
		if _, ok := clientSnap.Nodes[name]; !ok {
			t.Errorf("client store missing node %q", name)
		}
	}

	t.Log("snapshot bootstrap: client store created with 7 nodes from server snapshot")
}

// TestWatch_EventToWalkEvent_MappingComplete verifies all kami event types
// map correctly to framework walk events.
func TestWatch_EventToWalkEvent_MappingComplete(t *testing.T) {
	cases := []struct {
		kamiType kami.EventType
		wantType framework.WalkEventType
	}{
		{kami.EventNodeEnter, framework.EventNodeEnter},
		{kami.EventNodeExit, framework.EventNodeExit},
		{kami.EventTransition, framework.EventTransition},
		{kami.EventWalkComplete, framework.EventWalkComplete},
		{kami.EventWalkError, framework.EventWalkError},
		{kami.EventFanOutStart, framework.EventFanOutStart},
		{kami.EventFanOutEnd, framework.EventFanOutEnd},
	}

	for _, tc := range cases {
		t.Run(string(tc.kamiType), func(t *testing.T) {
			evt := kami.Event{
				Type:  tc.kamiType,
				Node:  "recall",
				Agent: "C04",
			}
			if tc.kamiType == kami.EventWalkError {
				evt.Error = "test error"
			}
			we := eventToWalkEvent(evt)
			if we.Type != tc.wantType {
				t.Errorf("type = %q, want %q", we.Type, tc.wantType)
			}
			if we.Node != "recall" {
				t.Errorf("node = %q, want recall", we.Node)
			}
			if we.Walker != "C04" {
				t.Errorf("walker = %q, want C04", we.Walker)
			}
			if tc.kamiType == kami.EventWalkError && we.Error == nil {
				t.Error("expected non-nil error for walk_error event")
			}
		})
	}
}

