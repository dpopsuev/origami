package knowledge

import (
	"context"

	kn "github.com/dpopsuev/origami/knowledge"
)

// Driver implements knowledge access for a specific SourceKind.
// Drivers are registered with the Router.
type Driver interface {
	Handles() kn.SourceKind
	Ensure(ctx context.Context, src kn.Source) error
	Search(ctx context.Context, src kn.Source, query string, maxResults int) ([]kn.SearchResult, error)
	Read(ctx context.Context, src kn.Source, path string) ([]byte, error)
	List(ctx context.Context, src kn.Source, root string, maxDepth int) ([]kn.ContentEntry, error)
}
