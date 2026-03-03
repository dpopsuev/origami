package sumi

import (
	"sync"

	"github.com/dpopsuev/origami/view"
)

const defaultRecorderCapacity = 30

// ViewRecorder is a thread-safe ring buffer that captures rendered TUI frames.
// Sumi records a frame on every state change; consumers (the Kami push
// goroutine, F12 dump) read the latest frame without blocking the render path.
type ViewRecorder struct {
	mu     sync.RWMutex
	frames []view.RecordedFrame
	head   int
	count  int
	cap    int
}

// NewViewRecorder creates a ring buffer with the given capacity.
func NewViewRecorder(capacity int) *ViewRecorder {
	if capacity <= 0 {
		capacity = defaultRecorderCapacity
	}
	return &ViewRecorder{
		frames: make([]view.RecordedFrame, capacity),
		cap:    capacity,
	}
}

// Record adds a frame, overwriting the oldest if full.
func (vr *ViewRecorder) Record(f view.RecordedFrame) {
	vr.mu.Lock()
	defer vr.mu.Unlock()
	idx := (vr.head + vr.count) % vr.cap
	if vr.count == vr.cap {
		vr.head = (vr.head + 1) % vr.cap
	} else {
		vr.count++
	}
	vr.frames[idx] = f
}

// Latest returns the most recently recorded frame, or nil if empty.
func (vr *ViewRecorder) Latest() *view.RecordedFrame {
	vr.mu.RLock()
	defer vr.mu.RUnlock()
	if vr.count == 0 {
		return nil
	}
	idx := (vr.head + vr.count - 1) % vr.cap
	f := vr.frames[idx]
	return &f
}

// Last returns the last n frames in chronological order (oldest first).
// If fewer than n frames exist, returns all available frames.
func (vr *ViewRecorder) Last(n int) []view.RecordedFrame {
	vr.mu.RLock()
	defer vr.mu.RUnlock()
	if n > vr.count {
		n = vr.count
	}
	out := make([]view.RecordedFrame, n)
	start := vr.count - n
	for i := 0; i < n; i++ {
		out[i] = vr.frames[(vr.head+start+i)%vr.cap]
	}
	return out
}

// Len returns the number of recorded frames.
func (vr *ViewRecorder) Len() int {
	vr.mu.RLock()
	defer vr.mu.RUnlock()
	return vr.count
}
