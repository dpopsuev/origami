package framework

import (
	"context"
	"testing"
	"time"
)

func TestThermalObserver_WarningEmitted(t *testing.T) {
	inner := &TraceCollector{}
	cancel := func() {}
	obs := &thermalObserver{
		inner:   inner,
		warning: 50 * time.Millisecond,
		ceiling: 200 * time.Millisecond,
		cancel:  cancel,
	}

	obs.OnEvent(WalkEvent{
		Type:    EventNodeExit,
		Node:    "A",
		Elapsed: 60 * time.Millisecond,
	})

	warnings := inner.EventsOfType(EventThermalWarning)
	if len(warnings) != 1 {
		t.Fatalf("expected 1 thermal warning, got %d", len(warnings))
	}

	meta := warnings[0].Metadata
	if meta["cumulative"] != 0.06 {
		t.Errorf("cumulative = %v, want 0.06", meta["cumulative"])
	}
}

func TestThermalObserver_WarningOnlyOnce(t *testing.T) {
	inner := &TraceCollector{}
	obs := &thermalObserver{
		inner:   inner,
		warning: 10 * time.Millisecond,
		ceiling: time.Hour,
		cancel:  func() {},
	}

	for i := 0; i < 5; i++ {
		obs.OnEvent(WalkEvent{
			Type:    EventNodeExit,
			Node:    "A",
			Elapsed: 20 * time.Millisecond,
		})
	}

	warnings := inner.EventsOfType(EventThermalWarning)
	if len(warnings) != 1 {
		t.Errorf("expected exactly 1 warning, got %d", len(warnings))
	}
}

func TestThermalObserver_CeilingCancelsContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	obs := &thermalObserver{
		warning: 10 * time.Millisecond,
		ceiling: 50 * time.Millisecond,
		cancel:  cancel,
	}

	obs.OnEvent(WalkEvent{
		Type:    EventNodeExit,
		Node:    "A",
		Elapsed: 100 * time.Millisecond,
	})

	select {
	case <-ctx.Done():
	default:
		t.Error("expected context to be cancelled at ceiling")
	}
}

func TestThermalObserver_IgnoresErrors(t *testing.T) {
	obs := &thermalObserver{
		warning: 10 * time.Millisecond,
		ceiling: 50 * time.Millisecond,
		cancel:  func() {},
	}

	obs.OnEvent(WalkEvent{
		Type:    EventNodeExit,
		Node:    "A",
		Elapsed: 100 * time.Millisecond,
		Error:   errForTest,
	})

	if obs.Total() != 0 {
		t.Errorf("error events should not accumulate latency, got %v", obs.Total())
	}
}

func TestThermalObserver_ForwardsToInner(t *testing.T) {
	inner := &TraceCollector{}
	obs := &thermalObserver{
		inner:   inner,
		warning: time.Hour,
		ceiling: time.Hour,
		cancel:  func() {},
	}

	obs.OnEvent(WalkEvent{Type: EventNodeEnter, Node: "A"})

	if len(inner.Events()) != 1 {
		t.Errorf("expected inner observer to receive event, got %d", len(inner.Events()))
	}
}

var errForTest = ErrNoEdge
