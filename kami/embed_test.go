package kami

import (
	"io"
	"strings"
	"testing"
)

func TestFrontendFS_ReturnsValidFilesystem(t *testing.T) {
	fs := FrontendFS()
	if fs == nil {
		t.Fatal("FrontendFS() returned nil — frontend/dist/ likely not built; run: cd kami/frontend && npm run build")
	}

	t.Run("index.html exists and is valid", func(t *testing.T) {
		f, err := fs.Open("/index.html")
		if err != nil {
			t.Fatalf("Open /index.html: %v", err)
		}
		defer f.Close()

		stat, err := f.Stat()
		if err != nil {
			t.Fatalf("Stat: %v", err)
		}
		if stat.Size() == 0 {
			t.Fatal("index.html is empty")
		}

		body, err := io.ReadAll(f)
		if err != nil {
			t.Fatalf("ReadAll: %v", err)
		}
		html := string(body)

		for _, needle := range []string{
			"<!doctype html>",
			`<div id="root">`,
			`<script type="module"`,
		} {
			if !strings.Contains(strings.ToLower(html), strings.ToLower(needle)) {
				t.Errorf("index.html missing %q", needle)
			}
		}
	})

	t.Run("assets directory accessible", func(t *testing.T) {
		f, err := fs.Open("/assets")
		if err != nil {
			t.Fatalf("Open /assets: %v", err)
		}
		defer f.Close()

		stat, err := f.Stat()
		if err != nil {
			t.Fatalf("Stat: %v", err)
		}
		if !stat.IsDir() {
			t.Error("/assets is not a directory")
		}
	})

	t.Run("vite.svg exists", func(t *testing.T) {
		f, err := fs.Open("/vite.svg")
		if err != nil {
			t.Fatalf("Open /vite.svg: %v", err)
		}
		defer f.Close()
	})
}
