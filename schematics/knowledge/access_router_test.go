package knowledge_test

import (
	"context"
	"fmt"
	"testing"

	kn "github.com/dpopsuev/origami/knowledge"
	knowledge "github.com/dpopsuev/origami/schematics/knowledge"
)

// stubDriver implements Driver for testing.
type stubDriver struct {
	kind         kn.SourceKind
	ensureErr    error
	searchResult []kn.SearchResult
	readResult   []byte
	listResult   []kn.ContentEntry
}

func (d *stubDriver) Handles() kn.SourceKind { return d.kind }

func (d *stubDriver) Ensure(_ context.Context, _ kn.Source) error {
	return d.ensureErr
}

func (d *stubDriver) Search(_ context.Context, src kn.Source, query string, max int) ([]kn.SearchResult, error) {
	if d.searchResult != nil {
		return d.searchResult, nil
	}
	return []kn.SearchResult{{
		Source:  src.Name,
		Path:    "test.go",
		Line:    42,
		Snippet: fmt.Sprintf("found %q in %s", query, src.Name),
	}}, nil
}

func (d *stubDriver) Read(_ context.Context, src kn.Source, path string) ([]byte, error) {
	if d.readResult != nil {
		return d.readResult, nil
	}
	return []byte(fmt.Sprintf("content of %s from %s", path, src.Name)), nil
}

func (d *stubDriver) List(_ context.Context, _ kn.Source, _ string, _ int) ([]kn.ContentEntry, error) {
	if d.listResult != nil {
		return d.listResult, nil
	}
	return []kn.ContentEntry{{Path: "README.md", IsDir: false, Size: 100}}, nil
}

func TestRouter_DispatchByKind(t *testing.T) {
	ctx := context.Background()

	gitDriver := &stubDriver{kind: kn.SourceKindRepo}
	docDriver := &stubDriver{kind: kn.SourceKindDoc}

	router := knowledge.NewRouter(
		knowledge.WithGitDriver(gitDriver),
		knowledge.WithDocsDriver(docDriver),
	)

	repoSrc := kn.Source{Name: "test-repo", Kind: kn.SourceKindRepo, URI: "https://github.com/test/repo"}
	docSrc := kn.Source{Name: "test-docs", Kind: kn.SourceKindDoc, URI: "https://docs.example.com"}

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

	src := kn.Source{Name: "unknown", Kind: "unknown"}
	_, err := router.Search(ctx, src, "query", 10)
	if err == nil {
		t.Fatal("expected error for unregistered kind")
	}
}

func TestRouter_Ensure(t *testing.T) {
	ctx := context.Background()
	driver := &stubDriver{kind: kn.SourceKindRepo}
	router := knowledge.NewRouter(knowledge.WithGitDriver(driver))

	src := kn.Source{Name: "repo", Kind: kn.SourceKindRepo, URI: "https://github.com/test/repo"}
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
	driver := &stubDriver{kind: kn.SourceKindRepo}
	router := knowledge.NewRouter(knowledge.WithGitDriver(driver))

	src := kn.Source{Name: "repo", Kind: kn.SourceKindRepo}
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
	driver := &stubDriver{kind: kn.SourceKindRepo}
	router := knowledge.NewRouter(knowledge.WithGitDriver(driver))

	src := kn.Source{Name: "repo", Kind: kn.SourceKindRepo}
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
	src := kn.Source{Name: "repo", Kind: kn.SourceKindRepo}
	_, err := router.Search(ctx, src, "q", 10)
	if err == nil {
		t.Fatal("expected error with no drivers")
	}

	// Register and retry
	router.Register(&stubDriver{kind: kn.SourceKindRepo})
	_, err = router.Search(ctx, src, "q", 10)
	if err != nil {
		t.Fatalf("Search after register: %v", err)
	}
}

func TestRouter_ReaderInterface(t *testing.T) {
	var r kn.Reader = knowledge.NewRouter()
	if r == nil {
		t.Fatal("NewRouter should satisfy Reader interface")
	}
}
