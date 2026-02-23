package dispatch

import (
	"fmt"
	"log/slog"
)

// ProviderRouter selects a Dispatcher based on the provider name carried
// in the DispatchContext. This enables per-step LLM routing: one node uses
// Cursor (MuxDispatcher), another uses Codex (CLIDispatcher), a third
// calls OpenAI directly (HTTPDispatcher).
//
// If no provider is set in the context, the Default dispatcher is used.
// If a provider is set but not found in the Routes map, an error is returned.
type ProviderRouter struct {
	Default Dispatcher
	Routes  map[string]Dispatcher
	Logger  *slog.Logger
}

// NewProviderRouter creates a router with a default dispatcher and optional routes.
func NewProviderRouter(defaultDispatcher Dispatcher, routes map[string]Dispatcher) *ProviderRouter {
	if routes == nil {
		routes = make(map[string]Dispatcher)
	}
	return &ProviderRouter{
		Default: defaultDispatcher,
		Routes:  routes,
		Logger:  discardLogger(),
	}
}

// WithProviderLogger sets a structured logger.
func WithProviderLogger(l *slog.Logger) func(*ProviderRouter) {
	return func(r *ProviderRouter) { r.Logger = l }
}

// Register adds a named provider route. Overwrites if the name already exists.
func (r *ProviderRouter) Register(provider string, d Dispatcher) {
	r.Routes[provider] = d
}

// Dispatch selects the appropriate dispatcher and delegates.
func (r *ProviderRouter) Dispatch(ctx DispatchContext) ([]byte, error) {
	if ctx.Provider == "" {
		r.Logger.Debug("provider router: using default dispatcher",
			slog.String("case_id", ctx.CaseID),
			slog.String("step", ctx.Step),
		)
		return r.Default.Dispatch(ctx)
	}

	d, ok := r.Routes[ctx.Provider]
	if !ok {
		return nil, fmt.Errorf("dispatch/provider: unknown provider %q (registered: %v)",
			ctx.Provider, r.providerNames())
	}

	r.Logger.Debug("provider router: routing to provider",
		slog.String("provider", ctx.Provider),
		slog.String("case_id", ctx.CaseID),
		slog.String("step", ctx.Step),
	)
	return d.Dispatch(ctx)
}

func (r *ProviderRouter) providerNames() []string {
	names := make([]string, 0, len(r.Routes))
	for k := range r.Routes {
		names = append(names, k)
	}
	return names
}
