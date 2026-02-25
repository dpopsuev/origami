package framework

import "sync"

// OutputCapture collects artifacts produced at each node during a walk.
// It implements WalkObserver and is safe for concurrent use during
// parallel fan-out walks.
//
// Usage:
//
//	capture := NewOutputCapture()
//	err := Run(ctx, path, input,
//	    WithOutputCapture(capture),
//	)
//	artifacts := capture.Artifacts()
type OutputCapture struct {
	mu        sync.RWMutex
	artifacts map[string]Artifact
}

// NewOutputCapture creates an OutputCapture ready for use.
func NewOutputCapture() *OutputCapture {
	return &OutputCapture{
		artifacts: make(map[string]Artifact),
	}
}

// OnEvent implements WalkObserver. It captures artifacts from node_exit events.
func (c *OutputCapture) OnEvent(e WalkEvent) {
	if e.Type != EventNodeExit || e.Node == "" {
		return
	}
	if e.Artifact == nil {
		return
	}
	c.mu.Lock()
	c.artifacts[e.Node] = e.Artifact
	c.mu.Unlock()
}

// Artifacts returns a copy of all captured node artifacts.
func (c *OutputCapture) Artifacts() map[string]Artifact {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make(map[string]Artifact, len(c.artifacts))
	for k, v := range c.artifacts {
		out[k] = v
	}
	return out
}

// ArtifactAt returns the artifact for a specific node, if captured.
func (c *OutputCapture) ArtifactAt(node string) (Artifact, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	a, ok := c.artifacts[node]
	return a, ok
}

// Reset clears all captured artifacts.
func (c *OutputCapture) Reset() {
	c.mu.Lock()
	c.artifacts = make(map[string]Artifact)
	c.mu.Unlock()
}

// WithOutputCapture attaches an OutputCapture as a walk observer.
// If another observer is already set, both are composed via MultiObserver.
func WithOutputCapture(capture *OutputCapture) RunOption {
	return func(c *runConfig) {
		if c.observer == nil {
			c.observer = capture
		} else {
			c.observer = MultiObserver{c.observer, capture}
		}
	}
}
