package knowledge

// SourceKind classifies what a knowledge source represents.
type SourceKind string

const (
	SourceKindRepo SourceKind = "repo"
	SourceKindSpec SourceKind = "spec"
	SourceKindDoc  SourceKind = "doc"
	SourceKindAPI  SourceKind = "api"
)

// Source is a single knowledge source — a repository, specification document,
// API endpoint, or other information resource available to the pipeline.
type Source struct {
	Name    string            `json:"name" yaml:"name"`
	Kind    SourceKind        `json:"kind" yaml:"kind"`
	URI     string            `json:"uri" yaml:"uri"`
	Purpose string            `json:"purpose,omitempty" yaml:"purpose,omitempty"`
	Branch  string            `json:"branch,omitempty" yaml:"branch,omitempty"`
	Tags    map[string]string `json:"tags,omitempty" yaml:"tags,omitempty"`
}

// KnowledgeSourceCatalog holds all knowledge sources available to a pipeline.
type KnowledgeSourceCatalog struct {
	Sources []Source `json:"sources" yaml:"sources"`
}
