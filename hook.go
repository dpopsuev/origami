package framework

import (
	"context"
	"fmt"
)

// Hook is a side-effect function invoked after a node completes.
// Hooks receive the validated artifact and can perform side effects
// (store writes, notifications) but do NOT affect routing or data flow.
// This is the Ansible notify/handler pattern.
type Hook interface {
	Name() string
	Run(ctx context.Context, nodeName string, artifact Artifact) error
}

// HookRegistry maps hook names to implementations.
type HookRegistry map[string]Hook

// Get returns the hook registered under name, or an error if not found.
func (r HookRegistry) Get(name string) (Hook, error) {
	if r == nil {
		return nil, fmt.Errorf("hook registry is nil")
	}
	h, ok := r[name]
	if !ok {
		return nil, fmt.Errorf("hook %q not registered", name)
	}
	return h, nil
}

// Register adds a hook. Panics on duplicate.
func (r HookRegistry) Register(h Hook) {
	if _, exists := r[h.Name()]; exists {
		panic(fmt.Sprintf("duplicate hook registration: %q", h.Name()))
	}
	r[h.Name()] = h
}

// HookFunc is a convenience adapter that turns a plain function into a Hook.
type HookFunc struct {
	name string
	fn   func(ctx context.Context, nodeName string, artifact Artifact) error
}

// NewHookFunc creates a Hook from a function.
func NewHookFunc(name string, fn func(ctx context.Context, nodeName string, artifact Artifact) error) *HookFunc {
	return &HookFunc{name: name, fn: fn}
}

func (h *HookFunc) Name() string { return h.name }
func (h *HookFunc) Run(ctx context.Context, nodeName string, artifact Artifact) error {
	return h.fn(ctx, nodeName, artifact)
}
