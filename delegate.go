package framework

// Category: Core Primitives

import (
	"context"
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
)

// DelegateNode is a Node that generates a sub-circuit instead of producing
// an artifact directly. When the walk loop encounters a DelegateNode, it
// calls GenerateCircuit to obtain a CircuitDef, builds and walks the
// sub-graph, and wraps the results in a DelegateArtifact.
//
// Implementations must satisfy both Node and DelegateNode. The Node.Process
// method is not called for delegate nodes during a walk — GenerateCircuit
// is called instead.
type DelegateNode interface {
	Node
	GenerateCircuit(ctx context.Context, nc NodeContext) (*CircuitDef, error)
}

// DelegateArtifact wraps the result of a sub-walk produced by a DelegateNode.
// It carries the generated circuit definition (for observability), the merged
// artifacts from the inner walk, and aggregate timing metrics.
type DelegateArtifact struct {
	// GeneratedCircuit is the CircuitDef returned by GenerateCircuit.
	// Retained for observability, calibration, and git-diffable audit.
	GeneratedCircuit *CircuitDef `json:"generated_circuit"`

	// InnerArtifacts maps inner node names to their produced artifacts.
	InnerArtifacts map[string]Artifact `json:"-"`

	// NodeCount is the number of nodes in the generated circuit.
	NodeCount int `json:"node_count"`

	// Elapsed is the wall-clock duration of the inner walk.
	Elapsed time.Duration `json:"elapsed"`

	// InnerError is non-nil if the sub-walk failed.
	InnerError error `json:"inner_error,omitempty"`
}

// dslDelegateNode is a DelegateNode produced by BuildGraph when a NodeDef
// has delegate: true and generator: set. The generator transformer is called
// to produce a CircuitDef (as YAML or as a *CircuitDef directly).
type dslDelegateNode struct {
	name    string
	element Element
	gen     Transformer
	config  map[string]any
	meta    map[string]any
}

func (n *dslDelegateNode) Name() string            { return n.name }
func (n *dslDelegateNode) ElementAffinity() Element { return n.element }

func (n *dslDelegateNode) Process(ctx context.Context, nc NodeContext) (Artifact, error) {
	da, err := n.GenerateCircuit(ctx, nc)
	if err != nil {
		return nil, err
	}
	return &DelegateArtifact{GeneratedCircuit: da, NodeCount: len(da.Nodes)}, nil
}

func (n *dslDelegateNode) GenerateCircuit(ctx context.Context, nc NodeContext) (*CircuitDef, error) {
	var input any
	if nc.PriorArtifact != nil {
		input = nc.PriorArtifact.Raw()
	}

	tc := &TransformerContext{
		Input:       input,
		Config:      n.config,
		NodeName:    n.name,
		Meta:        n.meta,
		WalkerState: nc.WalkerState,
	}

	result, err := n.gen.Transform(ctx, tc)
	if err != nil {
		return nil, fmt.Errorf("generator %s: %w", n.gen.Name(), err)
	}

	switch v := result.(type) {
	case *CircuitDef:
		return v, nil
	case CircuitDef:
		return &v, nil
	case map[string]any:
		data, err := yaml.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("generator %s: marshal circuit map: %w", n.gen.Name(), err)
		}
		return LoadCircuit(data)
	case string:
		return LoadCircuit([]byte(v))
	case []byte:
		return LoadCircuit(v)
	default:
		return nil, fmt.Errorf("generator %s: unexpected result type %T (want *CircuitDef, map, string, or []byte)", n.gen.Name(), result)
	}
}

// circuitRefNode is a DelegateNode that references a pre-loaded CircuitDef.
// Unlike dslDelegateNode which generates circuits dynamically via a transformer,
// circuitRefNode returns the stored CircuitDef directly — enabling static
// circuit composition where any circuit can be a subgraph node in another.
type circuitRefNode struct {
	name       string
	element    Element
	circuitDef *CircuitDef
	meta       map[string]any
}

func (n *circuitRefNode) Name() string            { return n.name }
func (n *circuitRefNode) ElementAffinity() Element { return n.element }

func (n *circuitRefNode) Process(ctx context.Context, nc NodeContext) (Artifact, error) {
	return &DelegateArtifact{GeneratedCircuit: n.circuitDef, NodeCount: len(n.circuitDef.Nodes)}, nil
}

func (n *circuitRefNode) GenerateCircuit(_ context.Context, _ NodeContext) (*CircuitDef, error) {
	return n.circuitDef, nil
}

func (a *DelegateArtifact) Type() string       { return "delegate" }
func (a *DelegateArtifact) Confidence() float64 { return a.confidence() }
func (a *DelegateArtifact) Raw() any            { return a.InnerArtifacts }

// confidence returns the average confidence across inner artifacts,
// or 0 if there are none.
func (a *DelegateArtifact) confidence() float64 {
	if len(a.InnerArtifacts) == 0 {
		return 0
	}
	var sum float64
	var count int
	for _, art := range a.InnerArtifacts {
		if art != nil {
			sum += art.Confidence()
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return sum / float64(count)
}
