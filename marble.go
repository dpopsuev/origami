package framework

import (
	"context"
	"fmt"
	"strings"
)

const maxMarbleDepth = 8

// Marble is a reusable pipeline component that wraps a Node with additional
// metadata about its composability. Atomic marbles wrap a single node;
// composite marbles contain a sub-graph that is walked when the marble is processed.
type Marble interface {
	Node
	PipelineDef() *PipelineDef
	IsComposite() bool
}

// MarbleRegistry maps marble names to factory functions.
// Names can be simple ("scorer") or fully-qualified ("namespace.scorer").
type MarbleRegistry map[string]func(NodeDef) Marble

// AtomicMarble wraps a Node as a non-composite marble.
type AtomicMarble struct {
	inner Node
}

func NewAtomicMarble(inner Node) *AtomicMarble {
	return &AtomicMarble{inner: inner}
}

func (m *AtomicMarble) Name() string                                                          { return m.inner.Name() }
func (m *AtomicMarble) ElementAffinity() Element                                              { return m.inner.ElementAffinity() }
func (m *AtomicMarble) Process(ctx context.Context, nc NodeContext) (Artifact, error)          { return m.inner.Process(ctx, nc) }
func (m *AtomicMarble) PipelineDef() *PipelineDef                                             { return nil }
func (m *AtomicMarble) IsComposite() bool                                                     { return false }

// CompositeMarble wraps a compiled sub-graph. When processed, it walks the
// sub-graph and returns the final artifact.
type CompositeMarble struct {
	name      string
	element   Element
	def       *PipelineDef
	reg       GraphRegistries
	depth     int
	inputMap  func(Artifact) any
	outputMap func(Artifact) Artifact
}

// CompositeMarbleOption configures a CompositeMarble.
type CompositeMarbleOption func(*CompositeMarble)

// WithInputMapper sets a function that transforms the parent artifact into
// sub-graph input before the sub-walk begins.
func WithInputMapper(fn func(Artifact) any) CompositeMarbleOption {
	return func(m *CompositeMarble) { m.inputMap = fn }
}

// WithOutputMapper sets a function that transforms the sub-walk's final
// artifact before returning it to the parent graph.
func WithOutputMapper(fn func(Artifact) Artifact) CompositeMarbleOption {
	return func(m *CompositeMarble) { m.outputMap = fn }
}

// NewCompositeMarble creates a composite marble from a pipeline definition.
func NewCompositeMarble(name string, elem Element, def *PipelineDef, reg GraphRegistries, opts ...CompositeMarbleOption) *CompositeMarble {
	m := &CompositeMarble{
		name:    name,
		element: elem,
		def:     def,
		reg:     reg,
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

func (m *CompositeMarble) Name() string            { return m.name }
func (m *CompositeMarble) ElementAffinity() Element { return m.element }
func (m *CompositeMarble) PipelineDef() *PipelineDef { return m.def }
func (m *CompositeMarble) IsComposite() bool         { return true }

func (m *CompositeMarble) Process(ctx context.Context, nc NodeContext) (Artifact, error) {
	if m.depth >= maxMarbleDepth {
		return nil, fmt.Errorf("marble nesting depth %d exceeds limit %d", m.depth, maxMarbleDepth)
	}

	graph, err := m.def.BuildGraph(m.reg)
	if err != nil {
		return nil, fmt.Errorf("marble %s: build sub-graph: %w", m.name, err)
	}

	walker := NewProcessWalker(m.name + "-sub")
	if m.inputMap != nil && nc.PriorArtifact != nil {
		walker.State().Context["input"] = m.inputMap(nc.PriorArtifact)
	} else if nc.PriorArtifact != nil {
		walker.State().Context["input"] = nc.PriorArtifact.Raw()
	}

	if err := graph.Walk(ctx, walker, m.def.Start); err != nil {
		return nil, fmt.Errorf("marble %s: sub-walk: %w", m.name, err)
	}

	var result Artifact
	for _, out := range walker.State().Outputs {
		result = out
	}

	if m.outputMap != nil && result != nil {
		result = m.outputMap(result)
	}

	return result, nil
}

// resolveMarble resolves a NodeDef with a marble: field to a Marble node.
// It checks the marble registry and handles FQCN resolution.
func resolveMarble(nd NodeDef, marbles MarbleRegistry, depth int) (Node, error) {
	if marbles == nil {
		return nil, fmt.Errorf("node %q: marble %q requested but no marble registry provided", nd.Name, nd.Marble)
	}

	factory, ok := marbles[nd.Marble]
	if !ok {
		if ns, name, err := resolveFQCNParts(nd.Marble); err == nil {
			factory, ok = marbles[ns+"."+name]
		}
	}
	if !ok {
		return nil, fmt.Errorf("node %q: marble %q not found in registry", nd.Name, nd.Marble)
	}

	marble := factory(nd)

	if cm, isComposite := marble.(*CompositeMarble); isComposite {
		cm.depth = depth + 1
	}

	return marble, nil
}

func resolveFQCNParts(fqcn string) (string, string, error) {
	parts := strings.SplitN(fqcn, ".", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("not a FQCN: %q", fqcn)
	}
	return parts[0], parts[1], nil
}

// DetectMarbleCycle checks for cycles in marble nesting by verifying that
// no marble references itself directly or transitively.
func DetectMarbleCycle(marbles MarbleRegistry) error {
	for name, factory := range marbles {
		marble := factory(NodeDef{Name: "cycle-check"})
		if err := checkCycle(marble, marbles, make(map[string]bool), name, 0); err != nil {
			return err
		}
	}
	return nil
}

func checkCycle(marble Marble, registry MarbleRegistry, visited map[string]bool, path string, depth int) error {
	if depth > maxMarbleDepth {
		return fmt.Errorf("marble cycle detected: %s (depth %d)", path, depth)
	}

	if !marble.IsComposite() {
		return nil
	}

	def := marble.PipelineDef()
	if def == nil {
		return nil
	}

	for _, nd := range def.Nodes {
		if nd.Marble == "" {
			continue
		}
		if visited[nd.Marble] {
			return fmt.Errorf("marble cycle detected: %s -> %s", path, nd.Marble)
		}

		factory, ok := registry[nd.Marble]
		if !ok {
			continue
		}

		visited[nd.Marble] = true
		child := factory(nd)
		if err := checkCycle(child, registry, visited, path+"->"+nd.Marble, depth+1); err != nil {
			return err
		}
		delete(visited, nd.Marble)
	}

	return nil
}
