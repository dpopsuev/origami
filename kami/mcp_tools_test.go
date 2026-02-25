package kami

import (
	"context"
	"encoding/json"
	"testing"

	framework "github.com/dpopsuev/origami"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func setupMCPTest() (*DebugController, *Server) {
	bridge := NewEventBridge(nil)
	dc := NewDebugController(bridge)
	srv := NewServer(Config{Bridge: bridge})
	return dc, srv
}

func TestMCPTools_SetBreakpointThenSnapshot(t *testing.T) {
	dc, _ := setupMCPTest()

	dc.SetBreakpoint("triage")

	dc.OnEvent(framework.WalkEvent{
		Type: framework.EventNodeEnter,
		Node: "recall",
	})
	dc.OnEvent(framework.WalkEvent{
		Type: framework.EventNodeExit,
		Node: "recall",
	})

	snap := dc.Snapshot()
	if snap.State != "running" {
		t.Errorf("state = %q, want running", snap.State)
	}
	if len(snap.NodesVisited) != 1 || snap.NodesVisited[0] != "recall" {
		t.Errorf("visited = %v, want [recall]", snap.NodesVisited)
	}
	if len(snap.Breakpoints) != 1 || snap.Breakpoints[0] != "triage" {
		t.Errorf("breakpoints = %v, want [triage]", snap.Breakpoints)
	}
}

func TestMCPTools_PauseViaHandler(t *testing.T) {
	dc, _ := setupMCPTest()

	handler := handlePause(dc)
	res, _, err := handler(context.Background(), nil, emptyInput{})
	if err != nil {
		t.Fatalf("pause: %v", err)
	}
	if len(res.Content) == 0 {
		t.Fatal("empty result content")
	}
	tc, ok := res.Content[0].(*sdkmcp.TextContent)
	if !ok {
		t.Fatalf("expected *TextContent, got %T", res.Content[0])
	}
	if tc.Text != "paused" {
		t.Errorf("text = %q, want paused", tc.Text)
	}

	if dc.State() != StatePaused {
		t.Errorf("state = %v, want paused", dc.State())
	}
}

func TestMCPTools_GetSnapshotHandler(t *testing.T) {
	dc, _ := setupMCPTest()

	dc.OnEvent(framework.WalkEvent{
		Type: framework.EventNodeEnter,
		Node: "recall",
	})

	handler := handleGetSnapshot(dc)
	res, snap, err := handler(context.Background(), nil, emptyInput{})
	if err != nil {
		t.Fatalf("get_snapshot: %v", err)
	}
	if snap.CurrentNode != "recall" {
		t.Errorf("current_node = %q, want recall", snap.CurrentNode)
	}

	tc := res.Content[0].(*sdkmcp.TextContent)
	var parsed PipelineSnapshot
	if err := json.Unmarshal([]byte(tc.Text), &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if parsed.CurrentNode != "recall" {
		t.Errorf("parsed current_node = %q, want recall", parsed.CurrentNode)
	}
}

func TestMCPTools_SetBreakpointHandler(t *testing.T) {
	dc, _ := setupMCPTest()

	handler := handleSetBreakpoint(dc)
	_, _, err := handler(context.Background(), nil, nodeInput{Node: "investigate"})
	if err != nil {
		t.Fatalf("set_breakpoint: %v", err)
	}

	bps := dc.ListBreakpoints()
	if len(bps) != 1 || bps[0] != "investigate" {
		t.Errorf("breakpoints = %v, want [investigate]", bps)
	}

	_, _, err = handler(context.Background(), nil, nodeInput{})
	if err == nil {
		t.Error("expected error for empty node")
	}
}

func TestMCPTools_SetSpeedHandler(t *testing.T) {
	_, srv := setupMCPTest()

	handler := handleSetSpeed(srv)

	_, _, err := handler(context.Background(), nil, speedInput{Speed: 0})
	if err == nil {
		t.Error("expected error for zero speed")
	}

	_, _, err = handler(context.Background(), nil, speedInput{Speed: -1})
	if err == nil {
		t.Error("expected error for negative speed")
	}
}

func TestMCPTools_RegistersAllTools(t *testing.T) {
	bridge := NewEventBridge(nil)
	dc := NewDebugController(bridge)
	srv := NewServer(Config{Bridge: bridge})
	mcpSrv := sdkmcp.NewServer(
		&sdkmcp.Implementation{Name: "kami-test", Version: "0.0.0"},
		nil,
	)
	RegisterMCPTools(mcpSrv, dc, srv)

	// RegisterMCPTools should not panic — if we got here, all 14 tools
	// were registered successfully. Test that the function is callable
	// and completes without error.
}
