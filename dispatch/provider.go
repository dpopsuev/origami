package dispatch

import (
	"errors"
	"fmt"
	"log/slog"
)

// ProviderRouter selects a Dispatcher based on the provider name carried
// in the DispatchContext. This enables per-step LLM routing: one node uses
// Cursor (MuxDispatcher), another uses Codex (CLIDispatcher), a third
// calls OpenAI directly (HTTPDispatcher).
//
// If no provider is set in the context, StepProviderHints is checked for a
// fallback mapping (populated by Ouroboros PersonaSheet auto-routing).
// If neither is set, the Default dispatcher is used.
// If a provider is set but not found in the Routes map, an error is returned.
//
// When Fallbacks are configured for a provider and the primary dispatch fails,
// the router iterates through the fallback chain until one succeeds or all fail.
type ProviderRouter struct {
	Default           Dispatcher
	Routes            map[string]Dispatcher
	StepProviderHints map[string]string     // step name → provider (populated by auto-routing)
	Fallbacks         map[string][]string   // provider → ordered fallback provider names
	Logger            *slog.Logger
	OnFallback        func(primary, fallback string, err error) // optional callback on fallback activation
}

// ProviderRouterOption configures a ProviderRouter.
type ProviderRouterOption func(*ProviderRouter)

// NewProviderRouter creates a router with a default dispatcher and optional routes.
func NewProviderRouter(defaultDispatcher Dispatcher, routes map[string]Dispatcher, opts ...ProviderRouterOption) *ProviderRouter {
	if routes == nil {
		routes = make(map[string]Dispatcher)
	}
	r := &ProviderRouter{
		Default: defaultDispatcher,
		Routes:  routes,
		Logger:  discardLogger(),
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// WithProviderLogger sets a structured logger.
func WithProviderLogger(l *slog.Logger) ProviderRouterOption {
	return func(r *ProviderRouter) { r.Logger = l }
}

// WithFallbacks configures fallback chains for providers.
func WithFallbacks(fallbacks map[string][]string) ProviderRouterOption {
	return func(r *ProviderRouter) { r.Fallbacks = fallbacks }
}

// WithFallbackCallback sets a callback invoked when a fallback provider is used.
func WithFallbackCallback(fn func(primary, fallback string, err error)) ProviderRouterOption {
	return func(r *ProviderRouter) { r.OnFallback = fn }
}

// Register adds a named provider route. Overwrites if the name already exists.
func (r *ProviderRouter) Register(provider string, d Dispatcher) {
	r.Routes[provider] = d
}

// Dispatch selects the appropriate dispatcher and delegates.
// On failure, iterates through the fallback chain if configured.
func (r *ProviderRouter) Dispatch(ctx DispatchContext) ([]byte, error) {
	if ctx.Provider == "" && r.StepProviderHints != nil {
		if hint, ok := r.StepProviderHints[ctx.Step]; ok {
			if d, found := r.Routes[hint]; found {
				r.Logger.Debug("provider router: auto-route from PersonaSheet",
					slog.String("provider", hint),
					slog.String("step", ctx.Step),
				)
				return r.dispatchWithFallback(hint, d, ctx)
			}
		}
	}

	if ctx.Provider == "" {
		r.Logger.Debug("provider router: using default dispatcher",
			slog.String("case_id", ctx.CaseID),
			slog.String("step", ctx.Step),
		)
		return r.dispatchWithFallback("default", r.Default, ctx)
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
	return r.dispatchWithFallback(ctx.Provider, d, ctx)
}

// dispatchWithFallback tries the primary dispatcher, then iterates through
// fallbacks on failure. Returns the first successful result or an aggregated error.
func (r *ProviderRouter) dispatchWithFallback(providerName string, primary Dispatcher, ctx DispatchContext) ([]byte, error) {
	result, err := primary.Dispatch(ctx)
	if err == nil {
		return result, nil
	}

	chain := r.Fallbacks[providerName]
	if len(chain) == 0 {
		return nil, err
	}

	var errs []error
	errs = append(errs, fmt.Errorf("primary %s: %w", providerName, err))

	for _, fb := range chain {
		d, ok := r.Routes[fb]
		if !ok {
			errs = append(errs, fmt.Errorf("fallback %s: not registered", fb))
			continue
		}

		r.Logger.Info("provider router: fallback activated",
			slog.String("primary", providerName),
			slog.String("fallback", fb),
			slog.String("step", ctx.Step),
		)
		if r.OnFallback != nil {
			r.OnFallback(providerName, fb, err)
		}

		result, fbErr := d.Dispatch(ctx)
		if fbErr == nil {
			return result, nil
		}
		errs = append(errs, fmt.Errorf("fallback %s: %w", fb, fbErr))
	}

	return nil, fmt.Errorf("all providers failed: %w", errors.Join(errs...))
}

func (r *ProviderRouter) providerNames() []string {
	names := make([]string, 0, len(r.Routes))
	for k := range r.Routes {
		names = append(names, k)
	}
	return names
}
