package rca

import "context"

// CodeReader provides read-only access to source code repositories.
// Implementations may clone repos locally and operate on the filesystem.
type CodeReader interface {
	// EnsureCloned ensures the repository is available locally and returns
	// the path to the local clone. Implementations should cache clones
	// and return immediately if already cloned.
	EnsureCloned(ctx context.Context, org, repo, branch string) (localPath string, err error)

	// SearchCode searches the local clone for files matching the given
	// keywords. Returns results ranked by relevance.
	SearchCode(ctx context.Context, localPath string, keywords []string) ([]SearchResult, error)

	// ReadFile reads a single file from the local clone.
	ReadFile(ctx context.Context, localPath, filePath string) ([]byte, error)

	// ListTree lists the directory structure of the local clone up to
	// maxDepth levels deep. Use maxDepth <= 0 for unlimited depth.
	ListTree(ctx context.Context, localPath string, maxDepth int) ([]TreeEntry, error)
}

// SearchResult represents a single match from a code search.
type SearchResult struct {
	File    string  `json:"file"`
	Line    int     `json:"line"`
	Snippet string  `json:"snippet"`
	Score   float64 `json:"score"`
}

// TreeEntry represents a file or directory in a repository tree.
type TreeEntry struct {
	Path  string `json:"path"`
	IsDir bool   `json:"is_dir"`
}

// CodeReaderFactory creates a CodeReader from connection parameters.
type CodeReaderFactory func(tokenSource string) (CodeReader, error)
