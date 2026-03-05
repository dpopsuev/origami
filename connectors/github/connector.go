package github

import (
	"context"

	"github.com/dpopsuev/origami/schematics/rca"
)

// Connector implements rca.CodeReader backed by shallow git clones and
// local filesystem operations (ripgrep for search, os.ReadFile for reads).
type Connector struct {
	cache *RepoCache
}

var _ rca.CodeReader = (*Connector)(nil)

// NewCodeReader creates a CodeReader that clones repos to a local cache.
// tokenSource is the path to a GitHub token file; if empty, $GITHUB_TOKEN
// is used, then the default .github-token file.
func NewCodeReader(tokenSource string) (rca.CodeReader, error) {
	token, err := ResolveToken(tokenSource)
	if err != nil {
		return nil, err
	}
	cache := NewRepoCache(DefaultCacheDir(), token)
	return &Connector{cache: cache}, nil
}

func (c *Connector) EnsureCloned(ctx context.Context, org, repo, branch string) (string, error) {
	return c.cache.EnsureCloned(ctx, org, repo, branch)
}

func (c *Connector) SearchCode(ctx context.Context, localPath string, keywords []string) ([]rca.SearchResult, error) {
	return SearchCode(ctx, localPath, keywords)
}

func (c *Connector) ReadFile(ctx context.Context, localPath, filePath string) ([]byte, error) {
	return ReadFile(ctx, localPath, filePath)
}

func (c *Connector) ListTree(ctx context.Context, localPath string, maxDepth int) ([]rca.TreeEntry, error) {
	return ListTree(ctx, localPath, maxDepth)
}
