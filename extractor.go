package framework
// Category: Processing & Support

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// Extractor converts unstructured input into structured output.
// Tome V primitive: the framework owns the interface; domain packages
// provide implementations. Type-erased for registry/DSL integration
// (same pattern as Node). Typed safety comes from generic constructors.
type Extractor interface {
	// Name returns the registered identifier for this extractor.
	Name() string

	// Extract converts input to structured output.
	// Implementations validate the input type at runtime and return
	// a descriptive error on type mismatch or extraction failure.
	Extract(ctx context.Context, input any) (any, error)
}

// ExtractorRegistry maps extractor names to Extractor implementations.
// Used by BuildGraph to wire ExtractorNode references in the DSL.
type ExtractorRegistry map[string]Extractor

// Get returns the extractor registered under name, or an error if not found.
// Supports FQCN resolution: a dot-qualified name does a direct lookup;
// an unqualified name tries direct first, then scans for a matching suffix.
func (r ExtractorRegistry) Get(name string) (Extractor, error) {
	if r == nil {
		return nil, fmt.Errorf("extractor registry is nil")
	}
	if ext, ok := r[name]; ok {
		return ext, nil
	}
	if !strings.Contains(name, ".") {
		suffix := "." + name
		for k, ext := range r {
			if strings.HasSuffix(k, suffix) {
				return ext, nil
			}
		}
	}
	return nil, fmt.Errorf("extractor %q not registered", name)
}

// Register adds an extractor to the registry. Panics on duplicate name.
func (r ExtractorRegistry) Register(ext Extractor) {
	if _, exists := r[ext.Name()]; exists {
		panic(fmt.Sprintf("duplicate extractor registration: %q", ext.Name()))
	}
	r[ext.Name()] = ext
}

// extractorNode is a Node that delegates processing to an Extractor.
// Created automatically by BuildGraph when a NodeDef has an Extractor field.
type extractorNode struct {
	name    string
	element Element
	ext     Extractor
	meta    map[string]any // from NodeDef.Meta
}

func (n *extractorNode) Name() string            { return n.name }
func (n *extractorNode) ElementAffinity() Element { return n.element }

func (n *extractorNode) Process(ctx context.Context, nc NodeContext) (Artifact, error) {
	var input any
	if nc.PriorArtifact != nil {
		input = nc.PriorArtifact.Raw()
	}
	result, err := n.ext.Extract(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("extractor %q: %w", n.ext.Name(), err)
	}
	return &extractorArtifact{
		typeName:   n.ext.Name(),
		confidence: 1.0,
		raw:        result,
	}, nil
}

// extractorArtifact wraps the output of an Extractor as an Artifact.
type extractorArtifact struct {
	typeName   string
	confidence float64
	raw        any
}

func (a *extractorArtifact) Type() string       { return a.typeName }
func (a *extractorArtifact) Confidence() float64 { return a.confidence }
func (a *extractorArtifact) Raw() any            { return a.raw }

// Built-in extractor names recognized by resolveNode.
const (
	BuiltinExtractorJSONSchema = "json-schema"
	BuiltinExtractorRegex      = "regex"
)

// JSONSchemaExtractor is a built-in extractor that unmarshals JSON input
// and validates it against an ArtifactSchema. No Go code required from
// consumers — just `extractor: json-schema` + `schema:` in the circuit YAML.
type JSONSchemaExtractor struct {
	schema *ArtifactSchema
}

func (e *JSONSchemaExtractor) Name() string { return BuiltinExtractorJSONSchema }

func (e *JSONSchemaExtractor) Extract(_ context.Context, input any) (any, error) {
	var data []byte
	switch v := input.(type) {
	case []byte:
		data = v
	case json.RawMessage:
		data = []byte(v)
	case string:
		data = []byte(v)
	default:
		b, err := json.Marshal(input)
		if err != nil {
			return nil, fmt.Errorf("json-schema extractor: marshal input: %w", err)
		}
		data = b
	}

	var result any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("json-schema extractor: unmarshal: %w", err)
	}

	if e.schema != nil {
		art := &extractorArtifact{typeName: BuiltinExtractorJSONSchema, confidence: 1.0, raw: result}
		if err := ValidateArtifact(e.schema, art); err != nil {
			return nil, fmt.Errorf("json-schema extractor: %w", err)
		}
	}

	return result, nil
}

