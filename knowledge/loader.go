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

type rawCatalog struct {
	Sources []Source `json:"sources" yaml:"sources"`
}

func (r rawCatalog) toCatalog() *KnowledgeSourceCatalog {
	return &KnowledgeSourceCatalog{Sources: r.Sources}
}
