package knowledge

import "github.com/dpopsuev/origami/knowledge"

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

// Source re-exports knowledge.Source for convenience.
type Source = knowledge.Source

// SourceKind re-exports knowledge.SourceKind for convenience.
type SourceKind = knowledge.SourceKind
