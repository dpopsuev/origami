package workspace

// Deprecated: Use knowledge.Source instead. The workspace package is superseded
// by github.com/dpopsuev/origami/knowledge which provides KnowledgeSourceCatalog,
// Source (with Kind, Tags), and KnowledgeSourceRouter for tag-based routing.

// Repo is one context source (local path or remote URL) with optional meta.
//
// Deprecated: Use knowledge.Source instead. Source.URI unifies Path and URL;
// Source.Tags replaces Purpose-based keyword matching.
type Repo struct {
	Path    string `json:"path" yaml:"path"`       // local path (absolute or relative to workspace file)
	URL     string `json:"url" yaml:"url"`        // remote git URL (alternative to path)
	Name    string `json:"name,omitempty" yaml:"name,omitempty"`
	Purpose string `json:"purpose,omitempty" yaml:"purpose,omitempty"` // agentic meta: role in RCA
	Branch  string `json:"branch,omitempty" yaml:"branch,omitempty"`   // override envelope ref
}

// Workspace is the context workspace: list of repos for analysis.
//
// Deprecated: Use knowledge.KnowledgeSourceCatalog instead.
type Workspace struct {
	Repos []Repo `json:"repos" yaml:"repos"`
}
