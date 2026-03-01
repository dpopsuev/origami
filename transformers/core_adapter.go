package transformers

import (
	fw "github.com/dpopsuev/origami"
	"github.com/dpopsuev/origami/dispatch"
)

// CoreAdapter returns an Adapter bundling the four built-in transformers
// (llm, http, jq, file) under the "core" namespace.
// The llm transformer requires a Dispatcher; pass nil to omit it.
func CoreAdapter(d dispatch.Dispatcher, opts ...CoreAdapterOption) *fw.Adapter {
	cfg := &coreAdapterConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	reg := fw.TransformerRegistry{}
	if d != nil {
		var llmOpts []LLMOption
		if cfg.baseDir != "" {
			llmOpts = append(llmOpts, WithBaseDir(cfg.baseDir))
		}
		reg["llm"] = NewLLM(d, llmOpts...)
	}
	reg["http"] = NewHTTP()
	reg["jq"] = NewJQ()

	var fileOpts []FileOption
	if cfg.baseDir != "" {
		fileOpts = append(fileOpts, WithRootDir(cfg.baseDir))
	}
	reg["file"] = NewFile(fileOpts...)
	reg["template-params"] = NewTemplateParams()

	return &fw.Adapter{
		Namespace:    "core",
		Name:         "origami-core",
		Version:      "1.0.0",
		Description:  "Built-in transformers: llm, http, jq, file",
		Transformers: reg,
	}
}

// CoreAdapterOption configures CoreAdapter.
type CoreAdapterOption func(*coreAdapterConfig)

type coreAdapterConfig struct {
	baseDir string
}

// WithCoreBaseDir sets the base directory for file and llm transformers.
func WithCoreBaseDir(dir string) CoreAdapterOption {
	return func(c *coreAdapterConfig) { c.baseDir = dir }
}
