package framework
// Category: Processing & Support

import "sync"

// ArtifactCapture provides access to artifacts captured during a walk.
// Obtain one via NewCapture() and use the returned WalkObserver during the walk.
type ArtifactCapture interface {
	ArtifactAt(node string) (Artifact, bool)
	Artifacts() map[string]Artifact
}

// NewCapture returns a WalkObserver that captures artifacts and an ArtifactCapture
// to read them after the walk. Use the observer with MultiObserver or run config.
func NewCapture() (WalkObserver, ArtifactCapture) {
	c := newOutputCapture()
	return c, c
}

// outputCapture collects artifacts produced at each node during a walk.
// It implements WalkObserver and is safe for concurrent use during
// parallel fan-out walks.
//
// Usage:
//
//	capture := newOutputCapture()
//	err := Run(ctx, path, input,
//	    withOutputCapture(capture),
//	)
//	artifacts := capture.Artifacts()
type outputCapture struct {
	mu        sync.RWMutex
	artifacts map[string]Artifact
}

// newOutputCapture creates an outputCapture ready for use.
func newOutputCapture() *outputCapture {
	return &outputCapture{
		artifacts: make(map[string]Artifact),
	}
}

// OnEvent implements WalkObserver. It captures artifacts from node_exit events.
func (c *outputCapture) OnEvent(e WalkEvent) {
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
func (c *outputCapture) Artifacts() map[string]Artifact {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make(map[string]Artifact, len(c.artifacts))
	for k, v := range c.artifacts {
		out[k] = v
	}
	return out
}

// ArtifactAt returns the artifact for a specific node, if captured.
func (c *outputCapture) ArtifactAt(node string) (Artifact, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	a, ok := c.artifacts[node]
	return a, ok
}

// Reset clears all captured artifacts.
func (c *outputCapture) Reset() {
	c.mu.Lock()
	c.artifacts = make(map[string]Artifact)
	c.mu.Unlock()
}

// withOutputCapture attaches an outputCapture as a walk observer.
// If another observer is already set, both are composed via MultiObserver.
func withOutputCapture(capture *outputCapture) RunOption {
	return func(c *runConfig) {
		if c.observer == nil {
			c.observer = capture
		} else {
			c.observer = MultiObserver{c.observer, capture}
		}
	}
}
