package framework

import (
	"context"
	"fmt"
	"log/slog"
)

// Runner drives a pipeline graph with automatic artifact schema validation.
// Domain tools create a Runner from a PipelineDef and their registries,
// then call Walk with a domain Walker. The Runner validates each artifact
// against the node's declared schema (if any) before edge evaluation.
type Runner struct {
	Pipeline *PipelineDef
	Graph    Graph
	Schemas  map[string]*ArtifactSchema // node name -> schema (from PipelineDef)
	Logger   *slog.Logger
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
	graph, err := def.BuildGraphWith(reg)
	if err != nil {
		return nil, fmt.Errorf("build graph: %w", err)
	}

	schemas := make(map[string]*ArtifactSchema, len(def.Nodes))
	for _, nd := range def.Nodes {
		if nd.Schema != nil {
			schemas[nd.Name] = nd.Schema
		}
	}

	return &Runner{
		Pipeline: def,
		Graph:    graph,
		Schemas:  schemas,
	}, nil
}

// Walk traverses the graph with the given walker, validating artifacts
// against declared schemas. It wraps the walker with a validating layer
// and delegates to the graph's Walk method.
// If walker is nil, a ProcessWalker is used (delegates to node.Process()).
func (r *Runner) Walk(ctx context.Context, walker Walker, startNode string) error {
	if walker == nil {
		walker = NewProcessWalker("default")
	}
	vw := &validatingWalker{
		inner:   walker,
		schemas: r.Schemas,
		log:     r.Logger,
	}
	return r.Graph.Walk(ctx, vw, startNode)
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
