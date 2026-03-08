// Package fold compiles a YAML manifest into a standalone Go binary.
// The manifest declares embedded assets and MCP server config.
// Fold generates a main.go for a domain-serve binary, then invokes go build.
package fold

import (
	"fmt"
	"os"
	"sort"

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
//
// Exactly one of Embed or Assets must be set. Embed is the legacy mode
// (single directory). Assets is the preferred mode (keyed file map).
type DomainServeConfig struct {
	Port   int       `yaml:"port"`             // listen port (default 9300)
	Embed  string    `yaml:"embed,omitempty"`  // legacy: directory to embed (e.g. "internal/")
	Assets *AssetMap `yaml:"assets,omitempty"` // preferred: keyed file map
}

// AssetMap declares domain files by section and key. Each map section
// (circuits, prompts, ...) maps a logical key to a file path relative
// to origami.yaml. The Files section holds singleton assets that don't
// belong to a typed section (e.g. vocabulary, heuristics).
type AssetMap struct {
	Circuits   map[string]string `yaml:"circuits,omitempty"`
	Prompts    map[string]string `yaml:"prompts,omitempty"`
	Schemas    map[string]string `yaml:"schemas,omitempty"`
	Scenarios  map[string]string `yaml:"scenarios,omitempty"`
	Scorecards map[string]string `yaml:"scorecards,omitempty"`
	Reports    map[string]string `yaml:"reports,omitempty"`
	Sources    map[string]string `yaml:"sources,omitempty"`
	Files      map[string]string `yaml:"files,omitempty"`
}

// AllPaths returns a deduplicated, sorted list of every file path
// referenced by the asset map.
func (a *AssetMap) AllPaths() []string {
	seen := make(map[string]struct{})
	for _, section := range a.allSections() {
		for _, p := range section {
			seen[p] = struct{}{}
		}
	}
	paths := make([]string, 0, len(seen))
	for p := range seen {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	return paths
}

// Sections returns the named map sections as a map of section name to
// key-path pairs. Only non-nil sections are included.
// Files is excluded — use ScalarFiles() for singleton assets.
func (a *AssetMap) Sections() map[string]map[string]string {
	result := make(map[string]map[string]string)
	for name, section := range map[string]map[string]string{
		"circuits":   a.Circuits,
		"prompts":    a.Prompts,
		"schemas":    a.Schemas,
		"scenarios":  a.Scenarios,
		"scorecards": a.Scorecards,
		"reports":    a.Reports,
		"sources":    a.Sources,
	} {
		if len(section) > 0 {
			result[name] = section
		}
	}
	return result
}

// ScalarFiles returns singleton asset entries as a map of name to path.
func (a *AssetMap) ScalarFiles() map[string]string {
	if len(a.Files) == 0 {
		return nil
	}
	cp := make(map[string]string, len(a.Files))
	for k, v := range a.Files {
		cp[k] = v
	}
	return cp
}

func (a *AssetMap) allSections() []map[string]string {
	return []map[string]string{
		a.Circuits, a.Prompts, a.Schemas, a.Scenarios,
		a.Scorecards, a.Reports, a.Sources, a.Files,
	}
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
	if ds := m.DomainServe; ds != nil {
		if ds.Embed != "" && ds.Assets != nil {
			return nil, fmt.Errorf("domain_serve: embed and assets are mutually exclusive")
		}
		if ds.Embed == "" && ds.Assets == nil {
			return nil, fmt.Errorf("domain_serve: one of embed or assets is required")
		}
	}
	return &m, nil
}
