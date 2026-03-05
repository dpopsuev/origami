package knowledge

import (
	"context"

	kn "github.com/dpopsuev/origami/knowledge"
)

// Reader re-exports knowledge.Reader so existing consumers don't break.
type Reader = kn.Reader

// Driver implements knowledge access for a specific SourceKind.
// Drivers are registered with the Router.
type Driver interface {
	Handles() kn.SourceKind
	Ensure(ctx context.Context, src Source) error
	Search(ctx context.Context, src Source, query string, maxResults int) ([]SearchResult, error)
	Read(ctx context.Context, src Source, path string) ([]byte, error)
	List(ctx context.Context, src Source, root string, maxDepth int) ([]ContentEntry, error)
}
