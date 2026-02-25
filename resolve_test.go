package framework

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolvePipelinePath_Embedded(t *testing.T) {
	clearEmbeddedPipelines()
	defer clearEmbeddedPipelines()

	content := []byte("pipeline: test\nnodes: []\nedges: []")
	RegisterEmbeddedPipeline("myPipeline", content)

	got, err := ResolvePipelinePath("mypipeline")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != string(content) {
		t.Fatalf("content mismatch: got %q, want %q", got, content)
	}
}

func TestResolvePipelinePath_EmbeddedCaseInsensitive(t *testing.T) {
	clearEmbeddedPipelines()
	defer clearEmbeddedPipelines()

	content := []byte("pipeline: ci")
	RegisterEmbeddedPipeline("CI-Pipeline", content)

	got, err := ResolvePipelinePath("ci-pipeline")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != string(content) {
		t.Fatalf("content mismatch")
	}
}

func TestResolvePipelinePath_FilesystemFallback(t *testing.T) {
	clearEmbeddedPipelines()
	defer clearEmbeddedPipelines()

	dir := t.TempDir()
	content := []byte("pipeline: fs-test\nnodes: []\nedges: []")
	if err := os.WriteFile(filepath.Join(dir, "test.yaml"), content, 0644); err != nil {
		t.Fatal(err)
	}

	got, err := ResolvePipelinePath("test", WithSearchDirs(dir))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != string(content) {
		t.Fatalf("content mismatch")
	}
}

func TestResolvePipelinePath_EnvVar(t *testing.T) {
	clearEmbeddedPipelines()
	defer clearEmbeddedPipelines()

	dir := t.TempDir()
	content := []byte("pipeline: env-test")
	if err := os.WriteFile(filepath.Join(dir, "envpipe.yaml"), content, 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("ORIGAMI_PIPELINES", dir)

	got, err := ResolvePipelinePath("envpipe")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != string(content) {
		t.Fatalf("content mismatch")
	}
}

func TestResolvePipelinePath_NotFound(t *testing.T) {
	clearEmbeddedPipelines()
	defer clearEmbeddedPipelines()

	_, err := ResolvePipelinePath("nonexistent-pipeline-xyz")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); !strings.Contains(got, "not found") {
		t.Fatalf("error should mention 'not found': %s", got)
	}
	if got := err.Error(); !strings.Contains(got, "searched:") {
		t.Fatalf("error should list searched paths: %s", got)
	}
}

func TestResolvePipelinePath_AutoYamlSuffix(t *testing.T) {
	clearEmbeddedPipelines()
	defer clearEmbeddedPipelines()

	dir := t.TempDir()
	content := []byte("pipeline: suffix-test")
	if err := os.WriteFile(filepath.Join(dir, "myfile.yaml"), content, 0644); err != nil {
		t.Fatal(err)
	}

	got, err := ResolvePipelinePath("myfile", WithSearchDirs(dir))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != string(content) {
		t.Fatalf("content mismatch")
	}
}
