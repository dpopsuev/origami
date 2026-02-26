package knowledge

// SourceKind classifies what a knowledge source represents.
type SourceKind string

const (
	SourceKindRepo SourceKind = "repo"
	SourceKindSpec SourceKind = "spec"
	SourceKindDoc  SourceKind = "doc"
	SourceKindAPI  SourceKind = "api"
)

// ReadPolicy controls when a source is included in pipeline routing.
type ReadPolicy string

const (
	// ReadAlways means the source is included in every pipeline run
	// regardless of tag matching or routing rules — mandatory prerequisite knowledge.
	ReadAlways ReadPolicy = "always"

	// ReadConditional means the source follows existing RouteRule logic —
	// included only when tags match. This is the default.
	ReadConditional ReadPolicy = "conditional"
)

// Source is a single knowledge source — a repository, specification document,
// API endpoint, or other information resource available to the pipeline.
type Source struct {
	Name       string            `json:"name" yaml:"name"`
	Kind       SourceKind        `json:"kind" yaml:"kind"`
	URI        string            `json:"uri" yaml:"uri"`
	Purpose    string            `json:"purpose,omitempty" yaml:"purpose,omitempty"`
	Branch     string            `json:"branch,omitempty" yaml:"branch,omitempty"`
	Tags       map[string]string `json:"tags,omitempty" yaml:"tags,omitempty"`
	ReadPolicy ReadPolicy        `json:"read_policy,omitempty" yaml:"read_policy,omitempty"`
	ReadWhen   string            `json:"read_when,omitempty" yaml:"read_when,omitempty"`
	LocalPath  string            `json:"local_path,omitempty" yaml:"local_path,omitempty"`
}

// IsAlwaysRead returns true if this source should be included in every
// pipeline run regardless of routing rules.
func (s Source) IsAlwaysRead() bool {
	return s.ReadPolicy == ReadAlways
}

// KnowledgeSourceCatalog holds all knowledge sources available to a pipeline.
type KnowledgeSourceCatalog struct {
	Sources []Source `json:"sources" yaml:"sources"`
}

// AlwaysReadSources returns all sources with ReadPolicy == ReadAlways.
func (c *KnowledgeSourceCatalog) AlwaysReadSources() []Source {
	if c == nil {
		return nil
	}
	var out []Source
	for _, s := range c.Sources {
		if s.IsAlwaysRead() {
			out = append(out, s)
		}
	}
	return out
}
