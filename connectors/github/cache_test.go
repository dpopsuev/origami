package github

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRepoCache_RepoPath(t *testing.T) {
	c := NewRepoCache("/tmp/cache", "")
	got := c.repoPath("openshift", "ptp-operator", "release-4.21")
	want := filepath.Join("/tmp/cache", "openshift", "ptp-operator", "release-4.21")
	if got != want {
		t.Errorf("repoPath = %q, want %q", got, want)
	}
}

func TestRepoCache_Clear(t *testing.T) {
	dir := t.TempDir()
	c := NewRepoCache(dir, "")
	sub := filepath.Join(dir, "openshift", "ptp-operator", "main")
	os.MkdirAll(sub, 0755)
	os.WriteFile(filepath.Join(sub, "test.txt"), []byte("test"), 0644)

	if err := c.Clear(); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("cache dir should be removed")
	}
}

func TestDefaultCacheDir(t *testing.T) {
	dir := DefaultCacheDir()
	if dir == "" {
		t.Error("DefaultCacheDir returned empty")
	}
}
