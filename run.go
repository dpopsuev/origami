package framework

import (
	"context"
	"fmt"
	"log/slog"
	"os"
)

// RunOption configures a Run invocation.
type RunOption func(*runConfig)

type runConfig struct {
	transformers TransformerRegistry
	hooks        HookRegistry
	extractors   ExtractorRegistry
	nodes        NodeRegistry
	edges        EdgeFactory
	overrides    map[string]any
	walker       Walker
	observer     WalkObserver
	logger       *slog.Logger
}

// WithTransformers registers transformers for the run.
func WithTransformers(reg TransformerRegistry) RunOption {
	return func(c *runConfig) { c.transformers = reg }
}

// WithHooks registers hooks for the run.
func WithHooks(reg HookRegistry) RunOption {
	return func(c *runConfig) { c.hooks = reg }
}

// WithExtractors registers extractors for the run.
func WithExtractors(reg ExtractorRegistry) RunOption {
	return func(c *runConfig) { c.extractors = reg }
}

// WithNodes registers node factories for the run.
func WithNodes(reg NodeRegistry) RunOption {
	return func(c *runConfig) { c.nodes = reg }
}

// WithEdges registers edge factories for the run.
func WithEdges(reg EdgeFactory) RunOption {
	return func(c *runConfig) { c.edges = reg }
}

// WithOverrides sets variable overrides (equivalent to --set key=value).
func WithOverrides(overrides map[string]any) RunOption {
	return func(c *runConfig) { c.overrides = overrides }
}

// WithWalker sets a custom Walker. If nil, ProcessWalker is used.
func WithWalker(w Walker) RunOption {
	return func(c *runConfig) { c.walker = w }
}

// WithRunObserver attaches a walk observer for the run.
func WithRunObserver(obs WalkObserver) RunOption {
	return func(c *runConfig) { c.observer = obs }
}

// WithLogger sets the logger for the run.
func WithLogger(l *slog.Logger) RunOption {
	return func(c *runConfig) { c.logger = l }
}

// Run loads a pipeline YAML, builds a graph, and walks it.
// This is the primary Go API for executing Origami pipelines.
//
//	err := framework.Run(ctx, "pipelines/rca.yaml", input,
//	    framework.WithTransformers(reg),
//	    framework.WithHooks(hooks),
//	    framework.WithOverrides(map[string]any{"recall_hit": 0.9}),
//	)
func Run(ctx context.Context, pipelinePath string, input any, opts ...RunOption) error {
	cfg := &runConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	data, err := os.ReadFile(pipelinePath)
	if err != nil {
		return fmt.Errorf("read pipeline %s: %w", pipelinePath, err)
	}

	def, err := LoadPipeline(data)
	if err != nil {
		return fmt.Errorf("parse pipeline %s: %w", pipelinePath, err)
	}

	if len(cfg.overrides) > 0 {
		def.Vars = MergeVars(def.Vars, cfg.overrides)
	}

	reg := GraphRegistries{
		Nodes:        cfg.nodes,
		Edges:        cfg.edges,
		Extractors:   cfg.extractors,
		Transformers: cfg.transformers,
		Hooks:        cfg.hooks,
	}

	runner, err := NewRunnerWith(def, reg)
	if err != nil {
		return fmt.Errorf("build runner: %w", err)
	}
	runner.Logger = cfg.logger

	if cfg.observer != nil {
		if dg, ok := runner.Graph.(*DefaultGraph); ok {
			dg.observer = cfg.observer
		}
	}

	walker := cfg.walker
	if walker == nil {
		walker = NewProcessWalker("run")
	}

	if input != nil {
		walker.State().Context["input"] = input
	}

	return runner.Walk(ctx, walker, def.Start)
}

// Validate loads and validates a pipeline YAML without executing it.
// Checks: YAML syntax, referential integrity, expression compilation,
// transformer resolution, hook resolution.
func Validate(pipelinePath string, opts ...RunOption) error {
	cfg := &runConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	data, err := os.ReadFile(pipelinePath)
	if err != nil {
		return fmt.Errorf("read pipeline %s: %w", pipelinePath, err)
	}

	def, err := LoadPipeline(data)
	if err != nil {
		return fmt.Errorf("parse pipeline %s: %w", pipelinePath, err)
	}

	if err := def.Validate(); err != nil {
		return fmt.Errorf("validate pipeline: %w", err)
	}

	reg := GraphRegistries{
		Nodes:        cfg.nodes,
		Edges:        cfg.edges,
		Extractors:   cfg.extractors,
		Transformers: cfg.transformers,
		Hooks:        cfg.hooks,
	}

	if _, err := def.BuildGraphWith(reg); err != nil {
		return fmt.Errorf("build graph (dry run): %w", err)
	}

	for _, nd := range def.Nodes {
		for _, hookName := range nd.After {
			if reg.Hooks != nil {
				if _, hErr := reg.Hooks.Get(hookName); hErr != nil {
					return fmt.Errorf("node %q: hook %q: %w", nd.Name, hookName, hErr)
				}
			}
		}
	}

	return nil
}
