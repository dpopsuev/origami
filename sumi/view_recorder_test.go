package sumi

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/dpopsuev/origami/view"
)

func makeFrame(n int) view.RecordedFrame {
	return view.RecordedFrame{
		Timestamp:    time.Date(2026, 3, 1, 12, 0, n, 0, time.UTC),
		Width:        120,
		Height:       40,
		LayoutTier:   "wide",
		SelectedNode: fmt.Sprintf("node-%d", n),
		WorkerCount:  n,
		EventCount:   n * 10,
		ViewText:     fmt.Sprintf("frame-%d", n),
	}
}

func TestViewRecorder_EmptyLatest(t *testing.T) {
	vr := NewViewRecorder(5)
	if vr.Latest() != nil {
		t.Fatal("expected nil from empty recorder")
	}
	if vr.Len() != 0 {
		t.Fatalf("expected Len()=0, got %d", vr.Len())
	}
}

func TestViewRecorder_RecordAndLatest(t *testing.T) {
	vr := NewViewRecorder(5)
	f := makeFrame(1)
	vr.Record(f)

	got := vr.Latest()
	if got == nil {
		t.Fatal("expected non-nil frame")
	}
	if got.ViewText != "frame-1" {
		t.Fatalf("expected frame-1, got %s", got.ViewText)
	}
	if vr.Len() != 1 {
		t.Fatalf("expected Len()=1, got %d", vr.Len())
	}
}

func TestViewRecorder_Last(t *testing.T) {
	vr := NewViewRecorder(10)
	for i := 0; i < 5; i++ {
		vr.Record(makeFrame(i))
	}

	last3 := vr.Last(3)
	if len(last3) != 3 {
		t.Fatalf("expected 3 frames, got %d", len(last3))
	}
	if last3[0].ViewText != "frame-2" {
		t.Fatalf("expected frame-2, got %s", last3[0].ViewText)
	}
	if last3[2].ViewText != "frame-4" {
		t.Fatalf("expected frame-4, got %s", last3[2].ViewText)
	}

	// Request more than available
	all := vr.Last(20)
	if len(all) != 5 {
		t.Fatalf("expected 5 frames, got %d", len(all))
	}
}

func TestViewRecorder_Overflow(t *testing.T) {
	vr := NewViewRecorder(3)
	for i := 0; i < 7; i++ {
		vr.Record(makeFrame(i))
	}

	if vr.Len() != 3 {
		t.Fatalf("expected Len()=3 after overflow, got %d", vr.Len())
	}

	latest := vr.Latest()
	if latest.ViewText != "frame-6" {
		t.Fatalf("expected latest=frame-6, got %s", latest.ViewText)
	}

	all := vr.Last(3)
	if all[0].ViewText != "frame-4" {
		t.Fatalf("expected oldest=frame-4, got %s", all[0].ViewText)
	}
	if all[2].ViewText != "frame-6" {
		t.Fatalf("expected newest=frame-6, got %s", all[2].ViewText)
	}
}

func TestViewRecorder_LastReturnsChronological(t *testing.T) {
	vr := NewViewRecorder(5)
	for i := 0; i < 8; i++ {
		vr.Record(makeFrame(i))
	}

	frames := vr.Last(5)
	for i := 1; i < len(frames); i++ {
		if !frames[i].Timestamp.After(frames[i-1].Timestamp) {
			t.Fatalf("frames not in chronological order at index %d", i)
		}
	}
}

func TestViewRecorder_DefaultCapacity(t *testing.T) {
	vr := NewViewRecorder(0)
	if vr.cap != defaultRecorderCapacity {
		t.Fatalf("expected default capacity %d, got %d", defaultRecorderCapacity, vr.cap)
	}
}

func TestViewRecorder_ConcurrentReadWrite(t *testing.T) {
	vr := NewViewRecorder(10)
	var wg sync.WaitGroup

	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			vr.Record(makeFrame(i))
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			vr.Latest()
			vr.Last(5)
			vr.Len()
		}
	}()

	wg.Wait()

	if vr.Len() != 10 {
		t.Fatalf("expected Len()=10 after 100 writes to cap-10, got %d", vr.Len())
	}
}
