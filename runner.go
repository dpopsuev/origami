package framework

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
)

// Interrupt signals that a walk should pause at the current node for
// human-in-the-loop review. When a walker's Handle returns an Interrupt,
// the runner checkpoints state and stops without error.
type Interrupt struct {
	Reason string
	Data   map[string]any
}

func (i Interrupt) Error() string {
	if i.Reason != "" {
		return "interrupt: " + i.Reason
	}
	return "interrupt"
}

// IsInterrupt checks whether an error is an Interrupt signal.
func IsInterrupt(err error) bool {
	var i Interrupt
	return errors.As(err, &i)
}

// AsInterrupt extracts the Interrupt from an error, if present.
func AsInterrupt(err error) (Interrupt, bool) {
	var i Interrupt
	ok := errors.As(err, &i)
	return i, ok
}

// Runner drives a pipeline graph with automatic artifact schema validation
// and after-hooks. Domain tools create a Runner from a PipelineDef and their
// registries, then call Walk with a domain Walker.
type Runner struct {
	Pipeline  *PipelineDef
	Graph     Graph
	Schemas   map[string]*ArtifactSchema // node name -> schema (from PipelineDef)
	NodeHooks map[string][]string        // node name -> hook names (from NodeDef.After)
	Hooks     HookRegistry               // resolved hooks
	Logger    *slog.Logger
}

// NewRunner constructs a Runner from a pipeline definition and registries.
// Backward-compatible: accepts (NodeRegistry, EdgeFactory, ...ExtractorRegistry).
func NewRunner(def *PipelineDef, nodes NodeRegistry, edges EdgeFactory, extractors ...ExtractorRegistry) (*Runner, error) {
	var extReg ExtractorRegistry
	if len(extractors) > 0 {
		extReg = extractors[0]
	}
	return NewRunnerWith(def, GraphRegistries{
		Nodes:      nodes,
		Edges:      edges,
		Extractors: extReg,
	})
}

// NewRunnerWith constructs a Runner using the full registries bundle.
func NewRunnerWith(def *PipelineDef, reg GraphRegistries) (*Runner, error) {
	graph, err := def.BuildGraph(reg)
	if err != nil {
		return nil, fmt.Errorf("build graph: %w", err)
	}

	schemas := make(map[string]*ArtifactSchema, len(def.Nodes))
	nodeHooks := make(map[string][]string, len(def.Nodes))
	nodeMeta := make(map[string]map[string]any, len(def.Nodes))
	needsFileWrite := false
	for _, nd := range def.Nodes {
		if nd.Schema != nil {
			schemas[nd.Name] = nd.Schema
		}
		if len(nd.After) > 0 {
			nodeHooks[nd.Name] = nd.After
			for _, h := range nd.After {
				if h == BuiltinHookFileWrite {
					needsFileWrite = true
				}
			}
		}
		if len(nd.Meta) > 0 {
			nodeMeta[nd.Name] = nd.Meta
		}
	}

	hooks := reg.Hooks
	if needsFileWrite {
		if hooks == nil {
			hooks = make(HookRegistry)
		}
		if _, err := hooks.Get(BuiltinHookFileWrite); err != nil {
			hooks.Register(&FileWriteHook{nodeMeta: nodeMeta})
		}
	}

	return &Runner{
		Pipeline:  def,
		Graph:     graph,
		Schemas:   schemas,
		NodeHooks: nodeHooks,
		Hooks:     hooks,
	}, nil
}

// Walk traverses the graph with the given walker, validating artifacts
// against declared schemas and firing after-hooks.
// If walker is nil, a ProcessWalker is used (delegates to node.Process()).
// Chain: hookingWalker -> validatingWalker -> inner walker.
func (r *Runner) Walk(ctx context.Context, walker Walker, startNode string) error {
	if walker == nil {
		walker = NewProcessWalker("default")
	}
	vw := &validatingWalker{
		inner:   walker,
		schemas: r.Schemas,
		log:     r.Logger,
	}
	var w Walker = vw
	if len(r.NodeHooks) > 0 && r.Hooks != nil {
		w = &hookingWalker{
			inner:     vw,
			nodeHooks: r.NodeHooks,
			hooks:     r.Hooks,
			log:       r.Logger,
		}
	}
	return r.Graph.Walk(ctx, w, startNode)
}

// validatingWalker wraps a domain Walker to add schema validation
// after each Handle call.
type validatingWalker struct {
	inner   Walker
	schemas map[string]*ArtifactSchema
	log     *slog.Logger
}

func (vw *validatingWalker) Identity() AgentIdentity {
	return vw.inner.Identity()
}

func (vw *validatingWalker) State() *WalkerState {
	return vw.inner.State()
}

func (vw *validatingWalker) Handle(ctx context.Context, node Node, nc NodeContext) (Artifact, error) {
	artifact, err := vw.inner.Handle(ctx, node, nc)
	if err != nil {
		return nil, err
	}

	schema, hasSchema := vw.schemas[node.Name()]
	if !hasSchema || schema == nil {
		return artifact, nil
	}

	if err := ValidateArtifact(schema, artifact); err != nil {
		if vw.log != nil {
			vw.log.Warn("artifact schema validation failed",
				slog.String("node", node.Name()),
				slog.String("error", err.Error()),
			)
		}
		return nil, fmt.Errorf("node %s: artifact schema violation: %w", node.Name(), err)
	}

	return artifact, nil
}

// hookingWalker wraps a Walker to invoke after-hooks once a node's
// artifact is validated. Hook errors are logged but do not stop the walk
// by default. Set FailOnHookError on the Runner to change this.
type hookingWalker struct {
	inner     Walker
	nodeHooks map[string][]string // node name -> hook names
	hooks     HookRegistry
	log       *slog.Logger
}

func (hw *hookingWalker) Identity() AgentIdentity { return hw.inner.Identity() }
func (hw *hookingWalker) State() *WalkerState     { return hw.inner.State() }

func (hw *hookingWalker) Handle(ctx context.Context, node Node, nc NodeContext) (Artifact, error) {
	artifact, err := hw.inner.Handle(ctx, node, nc)
	if err != nil {
		return nil, err
	}

	hookNames := hw.nodeHooks[node.Name()]
	for _, name := range hookNames {
		hook, hErr := hw.hooks.Get(name)
		if hErr != nil {
			if hw.log != nil {
				hw.log.Warn("hook not found", slog.String("hook", name), slog.String("node", node.Name()))
			}
			continue
		}
		if hErr = hook.Run(ctx, node.Name(), artifact); hErr != nil {
			if hw.log != nil {
				hw.log.Warn("hook error", slog.String("hook", name), slog.String("node", node.Name()), slog.String("error", hErr.Error()))
			}
		}
	}

	return artifact, nil
}

// checkpointingWalker wraps a Walker to save state after each successful
// node Handle. This is the outermost wrapper in the walker chain.
type checkpointingWalker struct {
	inner Walker
	cp    Checkpointer
}

func (cw *checkpointingWalker) Identity() AgentIdentity { return cw.inner.Identity() }
func (cw *checkpointingWalker) State() *WalkerState     { return cw.inner.State() }

func (cw *checkpointingWalker) Handle(ctx context.Context, node Node, nc NodeContext) (Artifact, error) {
	artifact, err := cw.inner.Handle(ctx, node, nc)
	if err != nil {
		return nil, err
	}
	if cpErr := cw.cp.Save(cw.inner.State()); cpErr != nil {
		return nil, fmt.Errorf("checkpoint after node %s: %w", node.Name(), cpErr)
	}
	return artifact, nil
}
