package knowledge

import "context"

// Reader is the unified knowledge access interface. Consuming schematics
// depend on this interface without knowing whether it's backed by an
// in-process router, an MCP subprocess, or a container.
type Reader interface {
	Ensure(ctx context.Context, src Source) error
	Search(ctx context.Context, src Source, query string, maxResults int) ([]SearchResult, error)
	Read(ctx context.Context, src Source, path string) ([]byte, error)
	List(ctx context.Context, src Source, root string, maxDepth int) ([]ContentEntry, error)
}

// SearchResult represents a single search hit from a knowledge source.
type SearchResult struct {
	Source  string `json:"source"`
	Path    string `json:"path"`
	Line    int    `json:"line,omitempty"`
	Snippet string `json:"snippet"`
}

// ContentEntry represents a file or document in a knowledge source listing.
type ContentEntry struct {
	Path  string `json:"path"`
	IsDir bool   `json:"is_dir"`
	Size  int64  `json:"size,omitempty"`
}
