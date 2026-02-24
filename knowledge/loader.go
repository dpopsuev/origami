package knowledge

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	yaml "go.yaml.in/yaml/v3"
)

// LoadFromPath reads a knowledge source catalog file (YAML or JSON).
// Supports both the new `sources:` format and the legacy `repos:` format
// (from workspace.Workspace) for backward compatibility.
func LoadFromPath(path string) (*KnowledgeSourceCatalog, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read catalog: %w", err)
	}
	return Load(data, filepath.Ext(path))
}

// Load parses a catalog from bytes. ext is a file extension hint
// (e.g. ".json", ".yaml"); empty means auto-detect from content.
func Load(data []byte, ext string) (*KnowledgeSourceCatalog, error) {
	ext = strings.ToLower(ext)
	if ext == ".yml" {
		ext = ".yaml"
	}

	var raw rawCatalog
	var err error

	switch ext {
	case ".yaml":
		err = yaml.Unmarshal(data, &raw)
	case ".json":
		err = json.Unmarshal(data, &raw)
	default:
		trimmed := strings.TrimSpace(string(data))
		if strings.HasPrefix(trimmed, "{") {
			err = json.Unmarshal(data, &raw)
		} else {
			err = yaml.Unmarshal(data, &raw)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("parse catalog: %w", err)
	}

	return raw.toCatalog(), nil
}

// rawCatalog is the union shape that accepts both `sources:` (new) and
// `repos:` (legacy) keys from YAML/JSON.
type rawCatalog struct {
	Sources []Source  `json:"sources" yaml:"sources"`
	Repos   []rawRepo `json:"repos" yaml:"repos"`
}

// rawRepo mirrors workspace.Repo for backward-compatible parsing.
type rawRepo struct {
	Name    string `json:"name" yaml:"name"`
	Path    string `json:"path" yaml:"path"`
	URL     string `json:"url" yaml:"url"`
	Purpose string `json:"purpose,omitempty" yaml:"purpose,omitempty"`
	Branch  string `json:"branch,omitempty" yaml:"branch,omitempty"`
}

func (r rawCatalog) toCatalog() *KnowledgeSourceCatalog {
	if len(r.Sources) > 0 {
		return &KnowledgeSourceCatalog{Sources: r.Sources}
	}

	sources := make([]Source, 0, len(r.Repos))
	for _, repo := range r.Repos {
		uri := repo.Path
		if uri == "" {
			uri = repo.URL
		}
		sources = append(sources, Source{
			Name:    repo.Name,
			Kind:    SourceKindRepo,
			URI:     uri,
			Purpose: repo.Purpose,
			Branch:  repo.Branch,
		})
	}
	return &KnowledgeSourceCatalog{Sources: sources}
}
