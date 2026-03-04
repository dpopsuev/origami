package cmd

import (
	"github.com/dpopsuev/origami/schematics/rca"
)

// Option configures the RCA schematic's external dependencies. Products
// inject connector implementations via these options so the schematic
// never directly imports connector packages.
type Option func(*schematicDeps)

type schematicDeps struct {
	sourceFactory  rca.SourceFactory
	pusherFactory  rca.PusherFactory
	fetcherFactory rca.LaunchFetcherFactory
	tokenChecker   rca.TokenChecker
}

// cfg holds the injected dependencies. Products call Apply before Execute.
var cfg schematicDeps

// Apply configures the schematic's external dependencies. Call before Execute.
func Apply(opts ...Option) {
	for _, opt := range opts {
		opt(&cfg)
	}
}

// WithSourceFactory injects a factory that creates a SourceAdapter from
// connection parameters (base URL, API key path, project).
func WithSourceFactory(f rca.SourceFactory) Option {
	return func(d *schematicDeps) { d.sourceFactory = f }
}

// WithPusherFactory injects a factory that creates a DefectPusher.
func WithPusherFactory(f rca.PusherFactory) Option {
	return func(d *schematicDeps) { d.pusherFactory = f }
}

// WithLaunchFetcherFactory injects a factory that creates a LaunchFetcher
// for the ingest/consume circuit.
func WithLaunchFetcherFactory(f rca.LaunchFetcherFactory) Option {
	return func(d *schematicDeps) { d.fetcherFactory = f }
}

// WithTokenChecker injects a function that validates token file existence
// and permissions. Used by commands that require API authentication.
func WithTokenChecker(f rca.TokenChecker) Option {
	return func(d *schematicDeps) { d.tokenChecker = f }
}

func checkTokenFileViaOption(path string) error {
	if cfg.tokenChecker != nil {
		return cfg.tokenChecker(path)
	}
	return checkTokenFile(path)
}
