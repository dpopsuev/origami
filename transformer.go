package framework

import (
	"context"
	"fmt"
	"strings"
)

// Transformer processes input data and produces structured output.
// Primary processing primitive in the Origami DSL. Built-in transformers
// (llm, http, jq, file) cover common cases; domain tools register custom
// transformers for specialized needs.
type Transformer interface {
	Name() string
	Transform(ctx context.Context, tc *TransformerContext) (any, error)
}

// TransformerContext carries all inputs needed by a transformer.
type TransformerContext struct {
	Input    any            // prior node's output (or pipeline input)
	Config   map[string]any // pipeline vars
	Prompt   string         // prompt template path or content
	NodeName string         // current node name
	Meta     map[string]any // additional metadata from NodeDef or walk state
}

// TransformerRegistry maps transformer names to implementations.
type TransformerRegistry map[string]Transformer

// Get returns the transformer registered under name, or an error if not found.
func (r TransformerRegistry) Get(name string) (Transformer, error) {
	if r == nil {
		return nil, fmt.Errorf("transformer registry is nil")
	}
	t, ok := r[name]
	if !ok {
		return nil, fmt.Errorf("transformer %q not registered", name)
	}
	return t, nil
}

// Register adds a transformer. Panics on duplicate.
func (r TransformerRegistry) Register(t Transformer) {
	if _, exists := r[t.Name()]; exists {
		panic(fmt.Sprintf("duplicate transformer registration: %q", t.Name()))
	}
	r[t.Name()] = t
}

// transformerNode is a Node that delegates to a Transformer.
// Created by BuildGraph when NodeDef.Transformer is set.
type transformerNode struct {
	name    string
	element Element
	trans   Transformer
	prompt  string         // from NodeDef.Prompt
	input   string         // from NodeDef.Input (e.g. "${recall.output}")
	config  map[string]any // pipeline vars (from PipelineDef.Vars)
}

func (n *transformerNode) Name() string            { return n.name }
func (n *transformerNode) ElementAffinity() Element { return n.element }

func (n *transformerNode) Process(ctx context.Context, nc NodeContext) (Artifact, error) {
	var input any

	if n.input != "" {
		resolved, err := ResolveInput(n.input, nc.WalkerState.Outputs)
		if err != nil {
			return nil, fmt.Errorf("node %s: resolve input: %w", n.name, err)
		}
		if resolved != nil {
			input = resolved.Raw()
		}
	} else if nc.PriorArtifact != nil {
		input = nc.PriorArtifact.Raw()
	}

	prompt := n.prompt
	if prompt != "" {
		sources := make(map[string]any)
		if nc.WalkerState != nil && nc.WalkerState.Outputs != nil {
			for k, v := range nc.WalkerState.Outputs {
				sources[k] = v.Raw()
			}
		}
		tmplCtx := TemplateContext{
			Output:  input,
			State:   nc.WalkerState,
			Config:  n.config,
			Sources: sources,
			Node:    n.name,
		}
		rendered, err := RenderPrompt(prompt, tmplCtx)
		if err != nil {
			return nil, fmt.Errorf("node %s: render prompt: %w", n.name, err)
		}
		prompt = rendered
	}

	tc := &TransformerContext{
		Input:    input,
		Config:   n.config,
		Prompt:   prompt,
		NodeName: n.name,
		Meta:     nc.Meta,
	}

	result, err := n.trans.Transform(ctx, tc)
	if err != nil {
		return nil, fmt.Errorf("transformer %q (node %s): %w", n.trans.Name(), n.name, err)
	}

	return &transformerArtifact{
		typeName:   n.trans.Name(),
		confidence: 1.0,
		raw:        result,
	}, nil
}

// transformerArtifact wraps transformer output as an Artifact.
type transformerArtifact struct {
	typeName   string
	confidence float64
	raw        any
}

func (a *transformerArtifact) Type() string       { return a.typeName }
func (a *transformerArtifact) Confidence() float64 { return a.confidence }
func (a *transformerArtifact) Raw() any            { return a.raw }

// TransformerFunc adapts a plain function into a Transformer.
func TransformerFunc(name string, fn func(context.Context, *TransformerContext) (any, error)) Transformer {
	return &transformerFunc{name: name, fn: fn}
}

type transformerFunc struct {
	name string
	fn   func(context.Context, *TransformerContext) (any, error)
}

func (t *transformerFunc) Name() string { return t.name }
func (t *transformerFunc) Transform(ctx context.Context, tc *TransformerContext) (any, error) {
	return t.fn(ctx, tc)
}

// IsTransformerNode returns true if the node was created from a transformer.
func IsTransformerNode(n Node) bool {
	_, ok := n.(*transformerNode)
	return ok
}

// TransformerNodeName resolves a transformer name, handling the "builtin:" prefix.
func TransformerNodeName(name string) string {
	return strings.TrimPrefix(name, "builtin:")
}
