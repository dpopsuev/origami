package fold

import (
	"strings"
	"testing"
)

func TestGenerateDockerfile(t *testing.T) {
	df, err := GenerateDockerfile("knowledge", "schematics/knowledge/cmd/serve", "1.24")
	if err != nil {
		t.Fatal(err)
	}

	content := string(df)

	if !strings.Contains(content, "DO NOT EDIT") {
		t.Error("missing DO NOT EDIT header")
	}
	if !strings.Contains(content, "FROM golang:1.24 AS builder") {
		t.Errorf("missing builder stage in:\n%s", content)
	}
	if !strings.Contains(content, "go build -o /knowledge ./schematics/knowledge/cmd/serve/") {
		t.Errorf("missing go build command in:\n%s", content)
	}
	if !strings.Contains(content, "FROM gcr.io/distroless/static") {
		t.Errorf("missing distroless base in:\n%s", content)
	}
	if !strings.Contains(content, `ENTRYPOINT ["/knowledge"]`) {
		t.Errorf("missing entrypoint in:\n%s", content)
	}
	if !strings.Contains(content, "EXPOSE 9100") {
		t.Errorf("missing expose in:\n%s", content)
	}
}

func TestGenerateDockerfile_NoServePath(t *testing.T) {
	_, err := GenerateDockerfile("test", "", "1.24")
	if err == nil {
		t.Fatal("expected error for empty serve path")
	}
}

func TestGenerateDockerfile_DefaultGoVersion(t *testing.T) {
	df, err := GenerateDockerfile("test", "cmd/serve", "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(df), "FROM golang:1.24") {
		t.Errorf("expected default Go version 1.24 in:\n%s", string(df))
	}
}
