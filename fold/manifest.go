// Package fold compiles a YAML manifest into a standalone Go binary.
// The manifest declares embedded assets and MCP server config.
// Fold generates a main.go for a domain-serve binary, then invokes go build.
package fold

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Manifest is the top-level origami.yaml schema.
type Manifest struct {
	Name        string             `yaml:"name"`
	Description string             `yaml:"description"`
	Version     string             `yaml:"version"`
	DomainServe *DomainServeConfig `yaml:"domain_serve,omitempty"`
}

// DomainServeConfig controls generation of a domain data MCP server binary.
// When present, origami fold produces a binary (<name>-domain-serve)
// that embeds the specified directory and serves it via domainserve.New().
type DomainServeConfig struct {
	Port  int    `yaml:"port"`  // listen port (default 9300)
	Embed string `yaml:"embed"` // directory to embed (e.g. "internal/")
}

// LoadManifest reads and parses an origami.yaml manifest file.
func LoadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}
	return ParseManifest(data)
}

// ParseManifest parses YAML bytes into a Manifest.
func ParseManifest(data []byte) (*Manifest, error) {
	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	if m.Name == "" {
		return nil, fmt.Errorf("manifest: name is required")
	}
	return &m, nil
}
