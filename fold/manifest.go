// Package fold compiles a YAML manifest into a standalone Go binary.
// The manifest declares imports, embedded assets, CLI commands, and
// MCP server config. Fold generates a main.go that uses the
// origamicli.NewCLI builder, then invokes go build.
package fold

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Manifest is the top-level origami.yaml schema.
type Manifest struct {
	Name        string                   `yaml:"name"`
	Description string                   `yaml:"description"`
	Version     string                   `yaml:"version"`
	Imports     []string                 `yaml:"imports"`
	Bindings    map[string]string        `yaml:"bindings,omitempty"`
	Circuits    map[string]string        `yaml:"circuits,omitempty"`
	Sources     map[string]string        `yaml:"sources,omitempty"`
	Embed       []string                 `yaml:"embed,omitempty"`
	CLI         CLIConfig                `yaml:"cli,omitempty"`
	Serve       *ProviderRef             `yaml:"serve,omitempty"`
	Demo        *ProviderRef             `yaml:"demo,omitempty"`
	Deploy      map[string]*DeployConfig `yaml:"deploy,omitempty"`
}

// DeployConfig controls how a secondary schematic is deployed.
type DeployConfig struct {
	Mode  string `yaml:"mode"`            // "in-process" (default), "subprocess", "container"
	Image string `yaml:"image,omitempty"` // OCI image name for container mode
}

// CLIConfig declares global flags and per-command configuration.
type CLIConfig struct {
	GlobalFlags []FlagDef            `yaml:"global_flags"`
	Analyze     *CommandDef          `yaml:"analyze,omitempty"`
	Calibrate   *CommandDef          `yaml:"calibrate,omitempty"`
	Consume     *CommandDef          `yaml:"consume,omitempty"`
	Dataset     *CommandDef          `yaml:"dataset,omitempty"`
	Extra       map[string]CommandDef `yaml:"extra,omitempty"`
}

// CommandDef configures a single CLI command. Either Provider (builder
// interface reference) or Circuit (circuit-walk command) must be set.
type CommandDef struct {
	Provider string    `yaml:"provider,omitempty"`
	Circuit  string    `yaml:"circuit,omitempty"`
	Flags    []FlagDef `yaml:"flags,omitempty"`
}

// ProviderRef references a builder interface implementation via FQCN.
type ProviderRef struct {
	Provider string `yaml:"provider"`
}

// FlagDef declares a CLI flag.
type FlagDef struct {
	Name     string `yaml:"name"`
	Type     string `yaml:"type"`
	Default  string `yaml:"default,omitempty"`
	Usage    string `yaml:"usage,omitempty"`
	Required bool   `yaml:"required,omitempty"`
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
