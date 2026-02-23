package mcp

import "github.com/dpopsuev/origami/dispatch"

// Signal and SignalBus are now defined in the dispatch package.
// These aliases preserve backward compatibility for existing consumers.

// Signal represents a single event on the agent message bus.
type Signal = dispatch.Signal

// SignalBus is a thread-safe, append-only signal log for agent coordination.
type SignalBus = dispatch.SignalBus

// NewSignalBus returns a new SignalBus.
var NewSignalBus = dispatch.NewSignalBus
