package fold

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ComponentMeta represents the parsed contents of a component.yaml file.
// Schematics declare required sockets (with the cmd.With* option function),
// while connectors declare which sockets they satisfy (with factory functions).
type ComponentMeta struct {
	Component   string              `yaml:"component"`
	Namespace   string              `yaml:"namespace"`
	Version     string              `yaml:"version"`
	Description string              `yaml:"description"`
	Requires    ComponentRequires   `yaml:"requires"`
	Satisfies   []SatisfiesEntry    `yaml:"satisfies"`
}

// ComponentRequires holds the requirements declared by a component.
type ComponentRequires struct {
	Origami string        `yaml:"origami"`
	Sockets []SocketEntry `yaml:"sockets"`
}

// SocketEntry represents a socket declared by a schematic in its component.yaml.
type SocketEntry struct {
	Name        string `yaml:"name"`
	Type        string `yaml:"type"`
	Option      string `yaml:"option"`
	Description string `yaml:"description"`
}

// SatisfiesEntry represents a socket that a connector can fulfill.
type SatisfiesEntry struct {
	Socket  string `yaml:"socket"`
	Factory string `yaml:"factory"`
}

// LoadComponentMeta reads and parses a component.yaml file.
func LoadComponentMeta(path string) (*ComponentMeta, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read component.yaml: %w", err)
	}
	var meta ComponentMeta
	if err := yaml.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("parse component.yaml %s: %w", path, err)
	}
	return &meta, nil
}

// loadComponentMetaForModule attempts to find and load a component.yaml
// from the local filesystem for the given Go module path and subpath.
func loadComponentMetaForModule(goPath string) (*ComponentMeta, error) {
	modPath := extractModule(goPath)
	localPath := findLocalModule(modPath)
	if localPath == "" {
		return nil, fmt.Errorf("cannot find local module for %s", modPath)
	}

	subPath := goPath[len(modPath):]
	metaPath := filepath.Join(localPath, subPath, "component.yaml")
	return LoadComponentMeta(metaPath)
}
