package knowledge

import (
	"context"

	kn "github.com/dpopsuev/origami/knowledge"
)

// Reader is the unified knowledge access interface. Consuming schematics
// depend on this interface and don't know whether it's backed by an
// in-process router, an MCP subprocess, or a container.
type Reader interface {
	Ensure(ctx context.Context, src Source) error
	Search(ctx context.Context, src Source, query string, maxResults int) ([]SearchResult, error)
	Read(ctx context.Context, src Source, path string) ([]byte, error)
	List(ctx context.Context, src Source, root string, maxDepth int) ([]ContentEntry, error)
}

// Driver implements knowledge access for a specific SourceKind.
// Drivers are registered with the Router.
type Driver interface {
	Handles() kn.SourceKind
	Ensure(ctx context.Context, src Source) error
	Search(ctx context.Context, src Source, query string, maxResults int) ([]SearchResult, error)
	Read(ctx context.Context, src Source, path string) ([]byte, error)
	List(ctx context.Context, src Source, root string, maxDepth int) ([]ContentEntry, error)
}
