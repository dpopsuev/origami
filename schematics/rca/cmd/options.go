package cmd

import (
	"github.com/dpopsuev/origami/schematics/rca"
)

// Option configures the RCA schematic's external dependencies. Products
// inject connector implementations via these options so the schematic
// never directly imports connector packages.
type Option func(*schematicDeps)

type schematicDeps struct {
	readerFactory     rca.SourceReaderFactory
	writerFactory     rca.DefectWriterFactory
	discovererFactory rca.RunDiscovererFactory
	storeFactory      rca.StoreFactory
	tokenChecker      rca.TokenChecker
}

// cfg holds the injected dependencies. Products call Apply before Execute.
var cfg schematicDeps

// Apply configures the schematic's external dependencies. Call before Execute.
func Apply(opts ...Option) {
	for _, opt := range opts {
		opt(&cfg)
	}
}

// WithSourceReader injects a factory that creates a SourceReader from
// connection parameters (base URL, API key path, project).
func WithSourceReader(f rca.SourceReaderFactory) Option {
	return func(d *schematicDeps) { d.readerFactory = f }
}

// WithDefectWriter injects a factory that creates a DefectWriter.
func WithDefectWriter(f rca.DefectWriterFactory) Option {
	return func(d *schematicDeps) { d.writerFactory = f }
}

// WithRunDiscoverer injects a factory that creates a RunDiscoverer
// for the ingest/consume circuit.
func WithRunDiscoverer(f rca.RunDiscovererFactory) Option {
	return func(d *schematicDeps) { d.discovererFactory = f }
}

// WithStore injects a factory that creates a store.Store from a database path.
// When not set, the built-in SQLite implementation (store.Open) is used.
func WithStore(f rca.StoreFactory) Option {
	return func(d *schematicDeps) { d.storeFactory = f }
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
