package kami

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"
)

type nopCloser struct{ *bytes.Buffer }

func (nopCloser) Close() error { return nil }

type readCloser struct{ io.Reader }

func (readCloser) Close() error { return nil }

func TestRecorder_WritesJSONL(t *testing.T) {
	bridge := NewEventBridge(nil)
	defer bridge.Close()

	var buf bytes.Buffer
	rec := NewRecorderWriter(bridge, nopCloser{&buf})
	rec.Start()

	bridge.Emit(Event{Type: EventNodeEnter, Node: "recall", Timestamp: time.Now().UTC()})
	bridge.Emit(Event{Type: EventNodeExit, Node: "recall", Timestamp: time.Now().UTC()})
	bridge.Emit(Event{Type: EventNodeEnter, Node: "triage", Timestamp: time.Now().UTC()})

	time.Sleep(50 * time.Millisecond)
	rec.Close()

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("got %d lines, want 3", len(lines))
	}

	for i, line := range lines {
		var evt Event
		if err := json.Unmarshal([]byte(line), &evt); err != nil {
			t.Fatalf("line %d unmarshal: %v", i, err)
		}
	}
}

func TestReplayer_EmitsEventsWithTiming(t *testing.T) {
	bridge := NewEventBridge(nil)
	defer bridge.Close()

	now := time.Now().UTC()
	events := []Event{
		{Type: EventNodeEnter, Node: "recall", Timestamp: now},
		{Type: EventNodeExit, Node: "recall", Timestamp: now.Add(100 * time.Millisecond)},
		{Type: EventNodeEnter, Node: "triage", Timestamp: now.Add(200 * time.Millisecond)},
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	for _, e := range events {
		enc.Encode(e)
	}

	id, ch := bridge.Subscribe()
	defer bridge.Unsubscribe(id)

	rp := NewReplayerReader(bridge, readCloser{&buf}, 10.0) // 10x speed

	done := make(chan struct{})
	errCh := make(chan error, 1)
	go func() {
		errCh <- rp.Play(done)
	}()

	var received []Event
	deadline := time.After(3 * time.Second)
	for len(received) < 3 {
		select {
		case e := <-ch:
			received = append(received, e)
		case <-deadline:
			t.Fatalf("timeout: got %d events, want 3", len(received))
		}
	}
	close(done)

	if err := <-errCh; err != nil {
		t.Fatalf("replay error: %v", err)
	}

	expectedNodes := []string{"recall", "recall", "triage"}
	for i, want := range expectedNodes {
		if received[i].Node != want {
			t.Errorf("event[%d].Node = %q, want %q", i, received[i].Node, want)
		}
	}
}

func TestReplayer_StoppableViaDone(t *testing.T) {
	bridge := NewEventBridge(nil)
	defer bridge.Close()

	now := time.Now().UTC()
	events := []Event{
		{Type: EventNodeEnter, Node: "recall", Timestamp: now},
		{Type: EventNodeEnter, Node: "triage", Timestamp: now.Add(10 * time.Second)}, // long gap
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	for _, e := range events {
		enc.Encode(e)
	}

	rp := NewReplayerReader(bridge, readCloser{&buf}, 1.0)
	done := make(chan struct{})

	errCh := make(chan error, 1)
	go func() {
		errCh <- rp.Play(done)
	}()

	// Give it time to emit the first event and start waiting for the second
	time.Sleep(50 * time.Millisecond)
	close(done)

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("replay error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("replayer did not stop after done signal")
	}
}

func TestRecordAndReplay_RoundTrip(t *testing.T) {
	bridge1 := NewEventBridge(nil)
	defer bridge1.Close()

	var buf bytes.Buffer
	rec := NewRecorderWriter(bridge1, nopCloser{&buf})
	rec.Start()

	now := time.Now().UTC()
	bridge1.Emit(Event{Type: EventNodeEnter, Node: "A", Timestamp: now})
	bridge1.Emit(Event{Type: EventNodeExit, Node: "A", Timestamp: now.Add(50 * time.Millisecond)})
	bridge1.Emit(Event{Type: EventTransition, Node: "B", Timestamp: now.Add(100 * time.Millisecond)})

	time.Sleep(50 * time.Millisecond)
	rec.Close()

	bridge2 := NewEventBridge(nil)
	defer bridge2.Close()

	id, ch := bridge2.Subscribe()
	defer bridge2.Unsubscribe(id)

	rp := NewReplayerReader(bridge2, readCloser{bytes.NewReader(buf.Bytes())}, 50.0)
	done := make(chan struct{})

	errCh := make(chan error, 1)
	go func() {
		errCh <- rp.Play(done)
	}()

	var received []Event
	deadline := time.After(3 * time.Second)
	for len(received) < 3 {
		select {
		case e := <-ch:
			received = append(received, e)
		case <-deadline:
			t.Fatalf("timeout: got %d events", len(received))
		}
	}
	close(done)

	if received[0].Node != "A" || received[1].Node != "A" || received[2].Node != "B" {
		t.Errorf("unexpected nodes: %v", received)
	}
}
