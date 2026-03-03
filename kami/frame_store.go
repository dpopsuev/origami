package kami

import (
	"sync"

	"github.com/dpopsuev/origami/view"
)

// FrameStore holds the latest Sumi TUI frame pushed via HTTP.
// Single-slot: only the most recent frame is retained.
type FrameStore struct {
	mu    sync.RWMutex
	frame *view.RecordedFrame
}

// NewFrameStore creates an empty FrameStore.
func NewFrameStore() *FrameStore {
	return &FrameStore{}
}

// Store replaces the current frame with f.
func (fs *FrameStore) Store(f view.RecordedFrame) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	fs.frame = &f
}

// Latest returns the most recent frame, or nil if none has been stored.
func (fs *FrameStore) Latest() *view.RecordedFrame {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	if fs.frame == nil {
		return nil
	}
	cp := *fs.frame
	return &cp
}
