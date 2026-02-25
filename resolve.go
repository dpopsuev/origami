package framework

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	embeddedPipelines   = map[string][]byte{}
	embeddedPipelinesMu sync.RWMutex
)

// RegisterEmbeddedPipeline registers a go:embed pipeline by name.
// Consumers call this in init() to make pipelines resolvable by name
// regardless of the working directory.
//
//	//go:embed pipelines/achilles.yaml
//	var pipelineYAML []byte
//
//	func init() {
//	    framework.RegisterEmbeddedPipeline("achilles", pipelineYAML)
//	}
func RegisterEmbeddedPipeline(name string, content []byte) {
	embeddedPipelinesMu.Lock()
	defer embeddedPipelinesMu.Unlock()
	embeddedPipelines[strings.ToLower(name)] = content
}

// ResolveOption configures pipeline path resolution.
type ResolveOption func(*resolveConfig)

type resolveConfig struct {
	searchDirs []string
}

// WithSearchDirs adds directories to the search path.
func WithSearchDirs(dirs ...string) ResolveOption {
	return func(c *resolveConfig) { c.searchDirs = append(c.searchDirs, dirs...) }
}

// ResolvePipelinePath resolves a pipeline by name, returning the YAML content.
// Resolution order:
//  1. Embedded registry (RegisterEmbeddedPipeline)
//  2. $ORIGAMI_PIPELINES directory
//  3. Additional search dirs (from WithSearchDirs)
//  4. Current working directory
//
// Returns the raw YAML bytes and nil error on success.
func ResolvePipelinePath(name string, opts ...ResolveOption) ([]byte, error) {
	cfg := &resolveConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	key := strings.ToLower(name)

	embeddedPipelinesMu.RLock()
	if content, ok := embeddedPipelines[key]; ok {
		embeddedPipelinesMu.RUnlock()
		return content, nil
	}
	embeddedPipelinesMu.RUnlock()

	candidates := []string{name}
	if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
		candidates = append(candidates, name+".yaml", name+".yml")
	}

	var searched []string

	if envDir := os.Getenv("ORIGAMI_PIPELINES"); envDir != "" {
		for _, c := range candidates {
			p := filepath.Join(envDir, c)
			searched = append(searched, p)
			if data, err := os.ReadFile(p); err == nil {
				return data, nil
			}
		}
	}

	for _, dir := range cfg.searchDirs {
		for _, c := range candidates {
			p := filepath.Join(dir, c)
			searched = append(searched, p)
			if data, err := os.ReadFile(p); err == nil {
				return data, nil
			}
		}
	}

	for _, c := range candidates {
		searched = append(searched, c)
		if data, err := os.ReadFile(c); err == nil {
			return data, nil
		}
	}

	return nil, fmt.Errorf("pipeline %q not found; searched: %s", name, strings.Join(searched, ", "))
}

// clearEmbeddedPipelines is for testing only.
func clearEmbeddedPipelines() {
	embeddedPipelinesMu.Lock()
	embeddedPipelines = map[string][]byte{}
	embeddedPipelinesMu.Unlock()
}
