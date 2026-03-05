package subprocess

import (
	"context"
	"fmt"
	"sync"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Orchestrator manages multiple named schematic subprocesses with
// lifecycle operations including hot-swap.
type Orchestrator struct {
	mu         sync.RWMutex
	schematics map[string]*Server
}

// NewOrchestrator creates an empty Orchestrator.
func NewOrchestrator() *Orchestrator {
	return &Orchestrator{
		schematics: make(map[string]*Server),
	}
}

// Register adds a named schematic server. It does not start the server.
func (o *Orchestrator) Register(name string, srv *Server) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.schematics[name] = srv
}

// Start launches a named schematic.
func (o *Orchestrator) Start(ctx context.Context, name string) error {
	o.mu.RLock()
	srv, ok := o.schematics[name]
	o.mu.RUnlock()
	if !ok {
		return fmt.Errorf("unknown schematic %q", name)
	}
	return srv.Start(ctx)
}

// Stop shuts down a named schematic.
func (o *Orchestrator) Stop(ctx context.Context, name string) error {
	o.mu.RLock()
	srv, ok := o.schematics[name]
	o.mu.RUnlock()
	if !ok {
		return fmt.Errorf("unknown schematic %q", name)
	}
	return srv.Stop(ctx)
}

// Swap replaces a running schematic with a new binary. The old process is
// gracefully stopped (drain in-flight requests) before the new one starts.
func (o *Orchestrator) Swap(ctx context.Context, name string, newBinary string, newArgs ...string) error {
	o.mu.Lock()
	old, ok := o.schematics[name]
	if !ok {
		o.mu.Unlock()
		return fmt.Errorf("unknown schematic %q", name)
	}

	// Create the replacement server
	replacement := &Server{
		BinaryPath: newBinary,
		Args:       newArgs,
		Env:        old.Env,
	}
	o.schematics[name] = replacement
	o.mu.Unlock()

	// Stop old (graceful drain via CommandTransport's stdin close → SIGTERM)
	if err := old.Stop(ctx); err != nil {
		// Even if stop fails, proceed with starting the new one
		_ = err
	}

	// Start new
	return replacement.Start(ctx)
}

// CallTool calls a tool on a named schematic.
func (o *Orchestrator) CallTool(ctx context.Context, name string, tool string, args map[string]any) (*sdkmcp.CallToolResult, error) {
	o.mu.RLock()
	srv, ok := o.schematics[name]
	o.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unknown schematic %q", name)
	}
	return srv.CallTool(ctx, tool, args)
}

// Healthy checks if a named schematic is healthy.
func (o *Orchestrator) Healthy(ctx context.Context, name string) bool {
	o.mu.RLock()
	srv, ok := o.schematics[name]
	o.mu.RUnlock()
	if !ok {
		return false
	}
	return srv.Healthy(ctx)
}

// StopAll stops all registered schematics.
func (o *Orchestrator) StopAll(ctx context.Context) {
	o.mu.RLock()
	names := make([]string, 0, len(o.schematics))
	for name := range o.schematics {
		names = append(names, name)
	}
	o.mu.RUnlock()

	for _, name := range names {
		o.Stop(ctx, name)
	}
}
