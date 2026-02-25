package framework

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// PipelineDef is the top-level DSL structure for declaring a pipeline graph.
// Layout follows P3 (reading-first): pipeline > zones > nodes > edges > start/done.
type PipelineDef struct {
	Pipeline    string             `yaml:"pipeline"`
	Description string             `yaml:"description,omitempty"`
	Vars        map[string]any     `yaml:"vars,omitempty"`
	Zones       map[string]ZoneDef `yaml:"zones,omitempty"`
	Nodes       []NodeDef          `yaml:"nodes"`
	Edges       []EdgeDef          `yaml:"edges"`
	Walkers     []WalkerDef        `yaml:"walkers,omitempty"`
	Start       string             `yaml:"start"`
	Done        string             `yaml:"done"`
}

// WalkerDef declares a walker (agent) in the pipeline YAML.
// This is the "care, but in YAML" counterpart to DefaultWalker.
type WalkerDef struct {
	Name         string             `yaml:"name"`
	Element      string             `yaml:"element,omitempty"`
	Persona      string             `yaml:"persona,omitempty"`
	Preamble     string             `yaml:"preamble,omitempty"`
	StepAffinity map[string]float64 `yaml:"step_affinity,omitempty"`
}

// ZoneDef declares a meta-phase zone (P7: optional, progressive disclosure).
type ZoneDef struct {
	Nodes      []string `yaml:"nodes"`
	Element    string   `yaml:"element,omitempty"`
	Stickiness int      `yaml:"stickiness,omitempty"`
}

// NodeDef declares a node in the pipeline.
// Resolution priority: Transformer > Extractor > NodeRegistry (Family/Name).
// Transformer is the DSL-first path; Extractor and NodeRegistry are escape hatches.
type NodeDef struct {
	Name        string          `yaml:"name"`
	Element     string          `yaml:"element,omitempty"`
	Family      string          `yaml:"family,omitempty"`
	Extractor   string          `yaml:"extractor,omitempty"`
	Transformer string          `yaml:"transformer,omitempty"`
	Provider    string          `yaml:"provider,omitempty"`
	Prompt      string          `yaml:"prompt,omitempty"`
	Input       string          `yaml:"input,omitempty"`
	After       []string        `yaml:"after,omitempty"`
	Schema      *ArtifactSchema `yaml:"schema,omitempty"`
}

// EdgeDef declares a conditional edge between two nodes.
// P5: both id (machine) and name (human) are present.
// When is an expression evaluated by expr-lang/expr against {output, state, config}.
// Condition is a human-readable comment (not evaluated).
type EdgeDef struct {
	ID        string `yaml:"id"`
	Name      string `yaml:"name"`
	From      string `yaml:"from"`
	To        string `yaml:"to"`
	Shortcut  bool   `yaml:"shortcut,omitempty"`
	Loop      bool   `yaml:"loop,omitempty"`
	Parallel  bool   `yaml:"parallel,omitempty"`
	Condition string `yaml:"condition,omitempty"`
	When      string `yaml:"when,omitempty"`
}

// NodeRegistry maps node family names to Node factory functions.
type NodeRegistry map[string]func(def NodeDef) Node

// EdgeFactory maps edge IDs to Edge factory functions.
type EdgeFactory map[string]func(def EdgeDef) Edge

// LoadPipeline parses a YAML pipeline definition and returns a PipelineDef.
func LoadPipeline(data []byte) (*PipelineDef, error) {
	var def PipelineDef
	if err := yaml.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("parse pipeline YAML: %w", err)
	}
	return &def, nil
}

// MarshalYAML serializes a PipelineDef back to YAML (P8: round-trip fidelity).
func (def *PipelineDef) MarshalYAML() ([]byte, error) {
	return yaml.Marshal(def)
}

// Validate checks referential integrity of the pipeline definition:
//   - pipeline name is non-empty
//   - at least one node and one edge exist
//   - start node exists in the node list
//   - all edge From/To reference existing nodes (or the done pseudo-node)
//   - all zone node references exist
func (def *PipelineDef) Validate() error {
	if def.Pipeline == "" {
		return fmt.Errorf("pipeline name is required")
	}
	if len(def.Nodes) == 0 {
		return fmt.Errorf("at least one node is required")
	}
	if len(def.Edges) == 0 {
		return fmt.Errorf("at least one edge is required")
	}
	if def.Start == "" {
		return fmt.Errorf("start node is required")
	}
	if def.Done == "" {
		return fmt.Errorf("done node is required")
	}

	nodeSet := make(map[string]bool, len(def.Nodes))
	for _, n := range def.Nodes {
		if n.Name == "" {
			return fmt.Errorf("node name is required")
		}
		if nodeSet[n.Name] {
			return fmt.Errorf("duplicate node name %q", n.Name)
		}
		nodeSet[n.Name] = true
	}

	if !nodeSet[def.Start] {
		return fmt.Errorf("start node %q not found in node list", def.Start)
	}

	edgeIDs := make(map[string]bool, len(def.Edges))
	for _, e := range def.Edges {
		if e.ID == "" {
			return fmt.Errorf("edge id is required")
		}
		if edgeIDs[e.ID] {
			return fmt.Errorf("duplicate edge id %q", e.ID)
		}
		edgeIDs[e.ID] = true

		if !nodeSet[e.From] {
			return fmt.Errorf("edge %s references unknown source node %q", e.ID, e.From)
		}
		if e.To != def.Done && !nodeSet[e.To] {
			return fmt.Errorf("edge %s references unknown target node %q", e.ID, e.To)
		}
	}

	for zoneName, z := range def.Zones {
		for _, nodeName := range z.Nodes {
			if !nodeSet[nodeName] {
				return fmt.Errorf("zone %q references unknown node %q", zoneName, nodeName)
			}
		}
	}

	return nil
}

// GraphRegistries bundles all optional registries for BuildGraph.
// Fields are optional; BuildGraph resolves nodes by priority:
// Transformer > Extractor > NodeRegistry (Family/Name).
type GraphRegistries struct {
	Nodes        NodeRegistry
	Edges        EdgeFactory
	Extractors   ExtractorRegistry
	Transformers TransformerRegistry
	Hooks        HookRegistry
}

// BuildGraph constructs a Graph from a PipelineDef using the provided registries.
// Node resolution priority: Transformer > Extractor > NodeRegistry (Family/Name).
// Edge resolution priority: expressionEdge (When) > EdgeFactory > dslEdge.
// For backward compatibility, also accepts (NodeRegistry, EdgeFactory, ...ExtractorRegistry).
func (def *PipelineDef) BuildGraph(nodes NodeRegistry, edges EdgeFactory, extractors ...ExtractorRegistry) (Graph, error) {
	var extReg ExtractorRegistry
	if len(extractors) > 0 && extractors[0] != nil {
		extReg = extractors[0]
	}
	return def.BuildGraphWith(GraphRegistries{
		Nodes:      nodes,
		Edges:      edges,
		Extractors: extReg,
	})
}

// BuildGraphWith constructs a Graph using the full registries bundle.
func (def *PipelineDef) BuildGraphWith(reg GraphRegistries) (Graph, error) {
	if err := def.Validate(); err != nil {
		return nil, fmt.Errorf("validate: %w", err)
	}

	fwNodes := make([]Node, 0, len(def.Nodes))
	for _, nd := range def.Nodes {
		node, err := def.resolveNode(nd, reg)
		if err != nil {
			return nil, err
		}
		fwNodes = append(fwNodes, node)
	}

	fwEdges := make([]Edge, 0, len(def.Edges))
	for _, ed := range def.Edges {
		if ed.When != "" {
			exprEdge, err := CompileExpressionEdge(ed, def.Vars)
			if err != nil {
				return nil, fmt.Errorf("edge %s: %w", ed.ID, err)
			}
			fwEdges = append(fwEdges, exprEdge)
		} else if reg.Edges != nil {
			if factory, ok := reg.Edges[ed.ID]; ok {
				fwEdges = append(fwEdges, factory(ed))
			} else {
				fwEdges = append(fwEdges, &dslEdge{def: ed})
			}
		} else {
			fwEdges = append(fwEdges, &dslEdge{def: ed})
		}
	}

	fwZones := make([]Zone, 0, len(def.Zones))
	for name, zd := range def.Zones {
		fwZones = append(fwZones, Zone{
			Name:            name,
			NodeNames:       zd.Nodes,
			ElementAffinity: Element(strings.ToLower(zd.Element)),
			Stickiness:      zd.Stickiness,
		})
	}

	return NewGraph(def.Pipeline, fwNodes, fwEdges, fwZones, WithDoneNode(def.Done))
}

// resolveNode creates a Node from a NodeDef using the priority chain:
// Transformer > Extractor > NodeRegistry (Family/Name).
func (def *PipelineDef) resolveNode(nd NodeDef, reg GraphRegistries) (Node, error) {
	elem := Element(strings.ToLower(nd.Element))

	if nd.Transformer != "" && reg.Transformers != nil {
		t, err := reg.Transformers.Get(nd.Transformer)
		if err != nil {
			return nil, fmt.Errorf("node %q: %w", nd.Name, err)
		}
		return &transformerNode{
			name:     nd.Name,
			element:  elem,
			trans:    t,
			prompt:   nd.Prompt,
			input:    nd.Input,
			provider: nd.Provider,
			config:   def.Vars,
		}, nil
	}

	if nd.Extractor != "" && reg.Extractors != nil {
		ext, err := reg.Extractors.Get(nd.Extractor)
		if err != nil {
			return nil, fmt.Errorf("node %q: %w", nd.Name, err)
		}
		return &extractorNode{
			name:    nd.Name,
			element: elem,
			ext:     ext,
		}, nil
	}

	if reg.Nodes == nil {
		return nil, fmt.Errorf("no node factory for family %q (node %q): node registry is nil", nd.Family, nd.Name)
	}
	factory, ok := reg.Nodes[nd.Family]
	if !ok {
		factory = reg.Nodes[nd.Name]
	}
	if factory == nil {
		return nil, fmt.Errorf("no node factory for family %q (node %q)", nd.Family, nd.Name)
	}
	return factory(nd), nil
}

// dslEdge is a default Edge implementation created from an EdgeDef when
// no custom factory is registered. It always matches (returns a transition).
type dslEdge struct {
	def EdgeDef
}

func (e *dslEdge) ID() string         { return e.def.ID }
func (e *dslEdge) From() string       { return e.def.From }
func (e *dslEdge) To() string         { return e.def.To }
func (e *dslEdge) IsShortcut() bool   { return e.def.Shortcut }
func (e *dslEdge) IsLoop() bool       { return e.def.Loop }
func (e *dslEdge) IsParallel() bool   { return e.def.Parallel }
func (e *dslEdge) Evaluate(_ Artifact, _ *WalkerState) *Transition {
	return &Transition{
		NextNode:    e.def.To,
		Explanation: e.def.Condition,
	}
}
