package sumi

import (
	"strings"
	"testing"
)

func TestRenderWarRoomStatusBar_ContainsCircuitName(t *testing.T) {
	d := StatusBarData{
		CircuitName: "my-circuit",
		NoColor:     true,
		Width:       120,
	}
	bar := RenderWarRoomStatusBar(d)
	if !strings.Contains(bar, "my-circuit") {
		t.Errorf("status bar should contain circuit name, got: %s", bar)
	}
}

func TestRenderWarRoomStatusBar_WorkerCount(t *testing.T) {
	d := StatusBarData{
		CircuitName: "test",
		WorkerCount: 3,
		NoColor:     true,
		Width:       120,
	}
	bar := RenderWarRoomStatusBar(d)
	if !strings.Contains(bar, "Workers: 3") {
		t.Errorf("status bar should show worker count, got: %s", bar)
	}
}

func TestRenderWarRoomStatusBar_EventCount(t *testing.T) {
	d := StatusBarData{
		CircuitName: "test",
		EventCount:  42,
		NoColor:     true,
		Width:       120,
	}
	bar := RenderWarRoomStatusBar(d)
	if !strings.Contains(bar, "Events: 42") {
		t.Errorf("status bar should show event count, got: %s", bar)
	}
}

func TestRenderWarRoomStatusBar_KamiConnected(t *testing.T) {
	d := StatusBarData{
		CircuitName: "test",
		KamiStatus:  KamiConnected,
		NoColor:     true,
		Width:       120,
	}
	bar := RenderWarRoomStatusBar(d)
	if !strings.Contains(bar, "SSE: on") {
		t.Errorf("status bar should show SSE on, got: %s", bar)
	}
}

func TestRenderWarRoomStatusBar_KamiOffline(t *testing.T) {
	d := StatusBarData{
		CircuitName: "test",
		KamiStatus:  KamiOffline,
		NoColor:     true,
		Width:       120,
	}
	bar := RenderWarRoomStatusBar(d)
	if !strings.Contains(bar, "SSE: off") {
		t.Errorf("status bar should show SSE off, got: %s", bar)
	}
}

func TestRenderWarRoomStatusBar_Paused(t *testing.T) {
	d := StatusBarData{
		CircuitName: "test",
		Paused:      true,
		NoColor:     true,
		Width:       120,
	}
	bar := RenderWarRoomStatusBar(d)
	if !strings.Contains(bar, "PAUSED") {
		t.Errorf("status bar should show PAUSED, got: %s", bar)
	}
}

func TestRenderWarRoomStatusBar_Done(t *testing.T) {
	d := StatusBarData{
		CircuitName: "test",
		Completed:   true,
		NoColor:     true,
		Width:       120,
	}
	bar := RenderWarRoomStatusBar(d)
	if !strings.Contains(bar, "DONE") {
		t.Errorf("status bar should show DONE, got: %s", bar)
	}
}

func TestRenderWarRoomStatusBar_Error(t *testing.T) {
	d := StatusBarData{
		CircuitName: "test",
		Error:       "something failed",
		NoColor:     true,
		Width:       120,
	}
	bar := RenderWarRoomStatusBar(d)
	if !strings.Contains(bar, "something failed") {
		t.Errorf("status bar should show error, got: %s", bar)
	}
}

func TestRenderWarRoomStatusBar_SelectedNode(t *testing.T) {
	d := StatusBarData{
		CircuitName:  "test",
		SelectedNode: "triage",
		NoColor:      true,
		Width:        120,
	}
	bar := RenderWarRoomStatusBar(d)
	if !strings.Contains(bar, "triage") {
		t.Errorf("status bar should show selected node, got: %s", bar)
	}
}
