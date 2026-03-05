package github

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/dpopsuev/origami/knowledge"
	skn "github.com/dpopsuev/origami/schematics/knowledge"
)

// GitDriver implements knowledge.Driver for git repositories. It wraps the
// existing RepoCache for shallow cloning and the ripgrep-based search.
type GitDriver struct {
	cache *RepoCache

	mu         sync.RWMutex
	localPaths map[string]string // uri -> local path
}

var _ skn.Driver = (*GitDriver)(nil)

// NewGitDriver creates a GitDriver backed by a shallow-clone cache.
// tokenSource is the path to a GitHub token file; if empty, standard
// resolution ($GITHUB_TOKEN, then .github-token) is used.
func NewGitDriver(tokenSource string) (*GitDriver, error) {
	token, err := ResolveToken(tokenSource)
	if err != nil {
		return nil, err
	}
	return &GitDriver{
		cache:      NewRepoCache(DefaultCacheDir(), token),
		localPaths: make(map[string]string),
	}, nil
}

func (d *GitDriver) Handles() knowledge.SourceKind {
	return knowledge.SourceKindRepo
}

func (d *GitDriver) Ensure(ctx context.Context, src skn.Source) error {
	org, repo, err := parseGitURI(src.URI)
	if err != nil {
		return err
	}

	localPath, err := d.cache.EnsureCloned(ctx, org, repo, src.Branch)
	if err != nil {
		return err
	}

	d.mu.Lock()
	d.localPaths[src.URI] = localPath
	d.mu.Unlock()
	return nil
}

func (d *GitDriver) Search(ctx context.Context, src skn.Source, query string, maxResults int) ([]skn.SearchResult, error) {
	localPath, err := d.resolvePath(ctx, src)
	if err != nil {
		return nil, err
	}

	keywords := strings.Fields(query)
	if len(keywords) == 0 {
		return nil, nil
	}

	rcaResults, err := SearchCode(ctx, localPath, keywords)
	if err != nil {
		return nil, err
	}

	results := make([]skn.SearchResult, 0, len(rcaResults))
	for _, r := range rcaResults {
		if len(results) >= maxResults {
			break
		}
		results = append(results, skn.SearchResult{
			Source:  src.Name,
			Path:    r.File,
			Line:    r.Line,
			Snippet: r.Snippet,
		})
	}
	return results, nil
}

func (d *GitDriver) Read(ctx context.Context, src skn.Source, path string) ([]byte, error) {
	localPath, err := d.resolvePath(ctx, src)
	if err != nil {
		return nil, err
	}
	return ReadFile(ctx, localPath, path)
}

func (d *GitDriver) List(ctx context.Context, src skn.Source, root string, maxDepth int) ([]skn.ContentEntry, error) {
	localPath, err := d.resolvePath(ctx, src)
	if err != nil {
		return nil, err
	}

	rcaEntries, err := ListTree(ctx, localPath, maxDepth)
	if err != nil {
		return nil, err
	}

	entries := make([]skn.ContentEntry, 0, len(rcaEntries))
	for _, e := range rcaEntries {
		if root != "" && root != "." && !strings.HasPrefix(e.Path, root) {
			continue
		}
		entries = append(entries, skn.ContentEntry{
			Path:  e.Path,
			IsDir: e.IsDir,
		})
	}
	return entries, nil
}

func (d *GitDriver) resolvePath(ctx context.Context, src skn.Source) (string, error) {
	d.mu.RLock()
	lp, ok := d.localPaths[src.URI]
	d.mu.RUnlock()
	if ok {
		return lp, nil
	}

	if err := d.Ensure(ctx, src); err != nil {
		return "", err
	}

	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.localPaths[src.URI], nil
}

// parseGitURI extracts org and repo from a GitHub URI.
// Handles "https://github.com/org/repo" and "https://github.com/org/repo.git".
func parseGitURI(uri string) (org, repo string, err error) {
	uri = strings.TrimSuffix(uri, ".git")
	parts := strings.Split(uri, "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("cannot parse git URI %q: expected at least org/repo", uri)
	}
	return parts[len(parts)-2], parts[len(parts)-1], nil
}
