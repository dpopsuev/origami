package knowledge_test

import (
	"context"
	"fmt"
	"testing"

	knowledge "github.com/dpopsuev/origami/schematics/knowledge"
	"github.com/dpopsuev/origami/schematics/toolkit"
)

// stubDriver implements Driver for testing.
type stubDriver struct {
	kind         toolkit.SourceKind
	ensureErr    error
	searchResult []toolkit.SearchResult
	readResult   []byte
	listResult   []toolkit.ContentEntry
}

func (d *stubDriver) Handles() toolkit.SourceKind { return d.kind }

func (d *stubDriver) Ensure(_ context.Context, _ toolkit.Source) error {
	return d.ensureErr
}

func (d *stubDriver) Search(_ context.Context, src toolkit.Source, query string, max int) ([]toolkit.SearchResult, error) {
	if d.searchResult != nil {
		return d.searchResult, nil
	}
	return []toolkit.SearchResult{{
		Source:  src.Name,
		Path:    "test.go",
		Line:    42,
		Snippet: fmt.Sprintf("found %q in %s", query, src.Name),
	}}, nil
}

func (d *stubDriver) Read(_ context.Context, src toolkit.Source, path string) ([]byte, error) {
	if d.readResult != nil {
		return d.readResult, nil
	}
	return []byte(fmt.Sprintf("content of %s from %s", path, src.Name)), nil
}

func (d *stubDriver) List(_ context.Context, _ toolkit.Source, _ string, _ int) ([]toolkit.ContentEntry, error) {
	if d.listResult != nil {
		return d.listResult, nil
	}
	return []toolkit.ContentEntry{{Path: "README.md", IsDir: false, Size: 100}}, nil
}

func TestRouter_DispatchByKind(t *testing.T) {
	ctx := context.Background()

	gitDriver := &stubDriver{kind: toolkit.SourceKindRepo}
	docDriver := &stubDriver{kind: toolkit.SourceKindDoc}

	router := knowledge.NewRouter(
		knowledge.WithGitDriver(gitDriver),
		knowledge.WithDocsDriver(docDriver),
	)

	repoSrc := toolkit.Source{Name: "test-repo", Kind: toolkit.SourceKindRepo, URI: "https://github.com/test/repo"}
	docSrc := toolkit.Source{Name: "test-docs", Kind: toolkit.SourceKindDoc, URI: "https://docs.example.com"}

	// Search git source
	results, err := router.Search(ctx, repoSrc, "main", 10)
	if err != nil {
		t.Fatalf("Search repo: %v", err)
	}
	if len(results) != 1 || results[0].Source != "test-repo" {
		t.Errorf("unexpected repo search result: %v", results)
	}

	// Search doc source
	results, err = router.Search(ctx, docSrc, "docs", 10)
	if err != nil {
		t.Fatalf("Search doc: %v", err)
	}
	if len(results) != 1 || results[0].Source != "test-docs" {
		t.Errorf("unexpected doc search result: %v", results)
	}
}

func TestRouter_UnknownKind(t *testing.T) {
	ctx := context.Background()
	router := knowledge.NewRouter()

	src := toolkit.Source{Name: "unknown", Kind: "unknown"}
	_, err := router.Search(ctx, src, "query", 10)
	if err == nil {
		t.Fatal("expected error for unregistered kind")
	}
}

func TestRouter_Ensure(t *testing.T) {
	ctx := context.Background()
	driver := &stubDriver{kind: toolkit.SourceKindRepo}
	router := knowledge.NewRouter(knowledge.WithGitDriver(driver))

	src := toolkit.Source{Name: "repo", Kind: toolkit.SourceKindRepo, URI: "https://github.com/test/repo"}
	if err := router.Ensure(ctx, src); err != nil {
		t.Fatalf("Ensure: %v", err)
	}

	// With error
	driver.ensureErr = fmt.Errorf("clone failed")
	if err := router.Ensure(ctx, src); err == nil {
		t.Fatal("expected error from driver")
	}
}

func TestRouter_Read(t *testing.T) {
	ctx := context.Background()
	driver := &stubDriver{kind: toolkit.SourceKindRepo}
	router := knowledge.NewRouter(knowledge.WithGitDriver(driver))

	src := toolkit.Source{Name: "repo", Kind: toolkit.SourceKindRepo}
	data, err := router.Read(ctx, src, "main.go")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if string(data) != "content of main.go from repo" {
		t.Errorf("unexpected content: %s", data)
	}
}

func TestRouter_List(t *testing.T) {
	ctx := context.Background()
	driver := &stubDriver{kind: toolkit.SourceKindRepo}
	router := knowledge.NewRouter(knowledge.WithGitDriver(driver))

	src := toolkit.Source{Name: "repo", Kind: toolkit.SourceKindRepo}
	entries, err := router.List(ctx, src, ".", 2)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(entries) != 1 || entries[0].Path != "README.md" {
		t.Errorf("unexpected entries: %v", entries)
	}
}

func TestRouter_Register(t *testing.T) {
	ctx := context.Background()
	router := knowledge.NewRouter()

	// Starts with no drivers
	src := toolkit.Source{Name: "repo", Kind: toolkit.SourceKindRepo}
	_, err := router.Search(ctx, src, "q", 10)
	if err == nil {
		t.Fatal("expected error with no drivers")
	}

	// Register and retry
	router.Register(&stubDriver{kind: toolkit.SourceKindRepo})
	_, err = router.Search(ctx, src, "q", 10)
	if err != nil {
		t.Fatalf("Search after register: %v", err)
	}
}

func TestRouter_ReaderInterface(t *testing.T) {
	var r toolkit.SourceReader = knowledge.NewRouter()
	if r == nil {
		t.Fatal("NewRouter should satisfy Reader interface")
	}
}
