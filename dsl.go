package framework

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// CircuitDef is the top-level DSL structure for declaring a circuit graph.
// Layout follows P3 (reading-first): circuit > zones > nodes > edges > start/done.
type CircuitDef struct {
	Circuit    string             `yaml:"circuit"`
	Description string             `yaml:"description,omitempty"`
	Imports     []string           `yaml:"imports,omitempty"`
	Vars        map[string]any     `yaml:"vars,omitempty"`
	Extractors  []ExtractorDef     `yaml:"extractors,omitempty"`
	Zones       map[string]ZoneDef `yaml:"zones,omitempty"`
	Nodes       []NodeDef          `yaml:"nodes"`
	Edges       []EdgeDef          `yaml:"edges"`
	Walkers     []WalkerDef        `yaml:"walkers,omitempty"`
	Start       string             `yaml:"start"`
	Done        string             `yaml:"done"`
}

// ExtractorDef declares a reusable extractor at the circuit level.
// Nodes reference extractors by name via NodeDef.Extractor.
// Type must be a built-in extractor type (json-schema, regex).
type ExtractorDef struct {
	Name    string          `yaml:"name"`
	Type    string          `yaml:"type"`
	Schema  *ArtifactSchema `yaml:"schema,omitempty"`
	Pattern string          `yaml:"pattern,omitempty"`
	OnError string          `yaml:"on_error,omitempty"`
}

// WalkerDef declares a walker (agent) in the circuit YAML.
// This is the "care, but in YAML" counterpart to DefaultWalker.
type WalkerDef struct {
	Name           string             `yaml:"name"`
	Element        string             `yaml:"element,omitempty"`
	Persona        string             `yaml:"persona,omitempty"`
	Preamble       string             `yaml:"preamble,omitempty"`
	OffsetPreamble string             `yaml:"offset_preamble,omitempty"`
	StepAffinity   map[string]float64 `yaml:"step_affinity,omitempty"`
}

// ContextFilterDef declares which context keys are allowed or blocked
// when a walker transitions out of a zone. Implements the decoupling
// capacitor pattern: zone-local data stays local.
type ContextFilterDef struct {
	Pass  []string `yaml:"pass,omitempty"`
	Block []string `yaml:"block,omitempty"`
}

// ZoneDef declares a meta-phase zone (P7: optional, progressive disclosure).
type ZoneDef struct {
	Nodes         []string          `yaml:"nodes"`
	Element       string            `yaml:"element,omitempty"`
	Stickiness    int               `yaml:"stickiness,omitempty"`
	Domain        string            `yaml:"domain,omitempty"`
	ContextFilter *ContextFilterDef `yaml:"context_filter,omitempty"`
}

// NodeDef declares a node in the circuit.
// Resolution priority: Transformer > Extractor > NodeRegistry (Family/Name).
// Transformer is the DSL-first path; Extractor and NodeRegistry are escape hatches.
type NodeDef struct {
	Name        string          `yaml:"name"`
	Description string          `yaml:"description,omitempty"`
	Element     string          `yaml:"element,omitempty"`
	Family      string          `yaml:"family,omitempty"`
	Extractor   string          `yaml:"extractor,omitempty"`
	Renderer    string          `yaml:"renderer,omitempty"`
	Transformer string          `yaml:"transformer,omitempty"`
	Provider    string          `yaml:"provider,omitempty"`
	Prompt      string          `yaml:"prompt,omitempty"`
	Input       string          `yaml:"input,omitempty"`
	Before      []string        `yaml:"before,omitempty"`
	After       []string        `yaml:"after,omitempty"`
	Schema      *ArtifactSchema `yaml:"schema,omitempty"`
	Cache       *CacheDef       `yaml:"cache,omitempty"`
	Meta        map[string]any  `yaml:"meta,omitempty"`
}

// CacheDef configures node-level caching via the DSL.
type CacheDef struct {
	TTL string `yaml:"ttl,omitempty"`
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
	Merge     string `yaml:"merge,omitempty"`
}

// Merge strategy constants for fan-in edges.
const (
	MergeAppend = "append"
	MergeLatest = "latest"
	MergeCustom = "custom"
)

// NodeRegistry maps node family names to Node factory functions.
type NodeRegistry map[string]func(def NodeDef) Node

// EdgeFactory maps edge IDs to Edge factory functions.
type EdgeFactory map[string]func(def EdgeDef) Edge

// LoadCircuit parses a YAML circuit definition and returns a CircuitDef.
func LoadCircuit(data []byte) (*CircuitDef, error) {
	var def CircuitDef
	if err := yaml.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("parse circuit YAML: %w", err)
	}
	return &def, nil
}

// MarshalYAML serializes a CircuitDef back to YAML (P8: round-trip fidelity).
func (def *CircuitDef) MarshalYAML() ([]byte, error) {
	return yaml.Marshal(def)
}

// Validate checks referential integrity of the circuit definition:
//   - circuit name is non-empty
//   - at least one node and one edge exist
//   - start node exists in the node list
//   - all edge From/To reference existing nodes (or the done pseudo-node)
//   - all zone node references exist
func (def *CircuitDef) Validate() error {
	if def.Circuit == "" {
		return fmt.Errorf("circuit name is required")
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

// ComponentLoader resolves an import name (e.g. "core", "vendor.rca-tools")
// to a live Component with populated registries. BuildGraph calls the loader
// for each entry in CircuitDef.Imports.
type ComponentLoader func(name string) (*Component, error)

// GraphRegistries bundles all optional registries for BuildGraph.
// Fields are optional; BuildGraph resolves nodes by priority:
// Transformer > Extractor > NodeRegistry (Family/Name).
type GraphRegistries struct {
	Nodes        NodeRegistry
	Edges        EdgeFactory
	Extractors   ExtractorRegistry
	Renderers    RendererRegistry
	Transformers TransformerRegistry
	Hooks        HookRegistry
	Components   ComponentLoader
}

// BuildGraph constructs a Graph from a CircuitDef using the full registries bundle.
// Node resolution priority: Transformer > Extractor > NodeRegistry (Family/Name).
// Edge resolution priority: expressionEdge (When) > EdgeFactory > dslEdge.
// When CircuitDef.Imports is non-empty and reg.Components is set, imported
// components are loaded and merged into the registries before node resolution.
func (def *CircuitDef) BuildGraph(reg GraphRegistries) (Graph, error) {
	if err := def.Validate(); err != nil {
		return nil, fmt.Errorf("validate: %w", err)
	}

	if len(def.Imports) > 0 && reg.Components != nil {
		comps := make([]*Component, 0, len(def.Imports))
		for _, imp := range def.Imports {
			c, err := reg.Components(imp)
			if err != nil {
				return nil, fmt.Errorf("import %q: %w", imp, err)
			}
			comps = append(comps, c)
		}
		merged, err := MergeComponents(reg, comps...)
		if err != nil {
			return nil, fmt.Errorf("merge imports: %w", err)
		}
		reg.Transformers = merged.Transformers
		reg.Extractors = merged.Extractors
		reg.Hooks = merged.Hooks
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
			Domain:          strings.ToLower(zd.Domain),
			ContextFilter:   zd.ContextFilter,
		})
	}

	return NewGraph(def.Circuit, fwNodes, fwEdges, fwZones, WithDoneNode(def.Done))
}

// resolveNode creates a Node from a NodeDef using the priority chain:
// Transformer > Extractor > NodeRegistry (Family/Name).
func (def *CircuitDef) resolveNode(nd NodeDef, reg GraphRegistries) (Node, error) {
	elem := Element(strings.ToLower(nd.Element))

	if nd.Transformer != "" {
		var t Transformer
		switch nd.Transformer {
		case BuiltinTransformerGoTemplate:
			t = &goTemplateTransformer{}
		case BuiltinTransformerPassthrough:
			t = &passthroughTransformer{}
		default:
			if reg.Transformers == nil {
				return nil, fmt.Errorf("node %q: transformer %q not found (registry is nil)", nd.Name, nd.Transformer)
			}
			var err error
			t, err = reg.Transformers.Get(nd.Transformer)
			if err != nil {
				return nil, fmt.Errorf("node %q: %w", nd.Name, err)
			}
		}
		return &transformerNode{
			name:     nd.Name,
			element:  elem,
			trans:    t,
			prompt:   nd.Prompt,
			input:    nd.Input,
			provider: nd.Provider,
			config:   def.Vars,
			meta:     nd.Meta,
		}, nil
	}

	if nd.Extractor != "" {
		ext, err := def.resolveExtractor(nd, reg)
		if err != nil {
			return nil, err
		}
		return &extractorNode{
			name:    nd.Name,
			element: elem,
			ext:     ext,
			meta:    nd.Meta,
		}, nil
	}

	if nd.Renderer != "" {
		rnd, err := def.resolveRenderer(nd, reg)
		if err != nil {
			return nil, err
		}
		return &rendererNode{
			name:    nd.Name,
			element: elem,
			rnd:     rnd,
			meta:    nd.Meta,
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

// resolveExtractor resolves an extractor reference from a NodeDef.
// Priority: built-in name → circuit-level ExtractorDef → ExtractorRegistry.
func (def *CircuitDef) resolveExtractor(nd NodeDef, reg GraphRegistries) (Extractor, error) {
	switch nd.Extractor {
	case BuiltinExtractorJSONSchema:
		return &JSONSchemaExtractor{schema: nd.Schema}, nil
	case BuiltinExtractorRegex:
		pattern, _ := nd.Meta["pattern"].(string)
		if pattern == "" {
			return nil, fmt.Errorf("node %q: regex extractor requires meta.pattern", nd.Name)
		}
		return NewRegexExtractor(nd.Name, pattern)
	}

	for _, ed := range def.Extractors {
		if ed.Name != nd.Extractor {
			continue
		}
		switch ed.Type {
		case BuiltinExtractorJSONSchema:
			schema := ed.Schema
			if nd.Schema != nil {
				schema = nd.Schema
			}
			return &JSONSchemaExtractor{schema: schema}, nil
		case BuiltinExtractorRegex:
			if ed.Pattern == "" {
				return nil, fmt.Errorf("extractor %q: regex type requires pattern", ed.Name)
			}
			return NewRegexExtractor(ed.Name, ed.Pattern)
		default:
			return nil, fmt.Errorf("extractor %q: unknown type %q", ed.Name, ed.Type)
		}
	}

	if reg.Extractors != nil {
		ext, err := reg.Extractors.Get(nd.Extractor)
		if err == nil {
			return ext, nil
		}
	}
	return nil, fmt.Errorf("node %q: extractor %q not found", nd.Name, nd.Extractor)
}

// resolveRenderer resolves a renderer reference from a NodeDef.
// Priority: built-in name → RendererRegistry.
func (def *CircuitDef) resolveRenderer(nd NodeDef, reg GraphRegistries) (Renderer, error) {
	if nd.Renderer == BuiltinRendererTemplate {
		return &TemplateRenderer{Template: nd.Prompt}, nil
	}
	if reg.Renderers != nil {
		rnd, err := reg.Renderers.Get(nd.Renderer)
		if err == nil {
			return rnd, nil
		}
	}
	return nil, fmt.Errorf("node %q: renderer %q not found", nd.Name, nd.Renderer)
}
