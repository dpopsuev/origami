package ouroborosmcp

import (
	"fmt"

	"github.com/dpopsuev/origami/ouroboros"
	"github.com/dpopsuev/origami/ouroboros/probes"
)

// ProbeHandler binds a probe's prompt builder, scorer, and response parsing
// strategy. The MCP session uses this to generate prompts and score responses
// for any registered probe, not just refactor-v1.
type ProbeHandler struct {
	ID             string
	Prompt         func() string
	Score          func(string) map[ouroboros.Dimension]float64
	NeedsCodeBlock bool
}

// ProbeRegistry maps probe IDs to their handlers.
type ProbeRegistry struct {
	handlers map[string]*ProbeHandler
}

// NewProbeRegistry creates a registry pre-populated with all 5 Ouroboros probes.
func NewProbeRegistry() *ProbeRegistry {
	r := &ProbeRegistry{handlers: make(map[string]*ProbeHandler)}

	r.Register(&ProbeHandler{
		ID:             "refactor-v1",
		Prompt:         probes.RefactorPrompt,
		Score:          probes.ScoreRefactor,
		NeedsCodeBlock: true,
	})
	r.Register(&ProbeHandler{
		ID:             "debug-v1",
		Prompt:         probes.DebugPrompt,
		Score:          probes.ScoreDebug,
		NeedsCodeBlock: false,
	})
	r.Register(&ProbeHandler{
		ID:             "summarize-v1",
		Prompt:         probes.SummarizePrompt,
		Score:          probes.ScoreSummarize,
		NeedsCodeBlock: false,
	})
	r.Register(&ProbeHandler{
		ID:             "ambiguity-v1",
		Prompt:         probes.AmbiguityPrompt,
		Score:          probes.ScoreAmbiguity,
		NeedsCodeBlock: false,
	})
	r.Register(&ProbeHandler{
		ID:             "persistence-v1",
		Prompt:         probes.PersistencePrompt,
		Score:          probes.ScorePersistence,
		NeedsCodeBlock: false,
	})

	return r
}

// Register adds a probe handler to the registry.
func (r *ProbeRegistry) Register(h *ProbeHandler) {
	r.handlers[h.ID] = h
}

// Get returns the handler for the given probe ID, or an error if unknown.
func (r *ProbeRegistry) Get(probeID string) (*ProbeHandler, error) {
	h, ok := r.handlers[probeID]
	if !ok {
		return nil, fmt.Errorf("unknown probe_id %q; available: %v", probeID, r.IDs())
	}
	return h, nil
}

// IDs returns all registered probe IDs.
func (r *ProbeRegistry) IDs() []string {
	ids := make([]string, 0, len(r.handlers))
	for id := range r.handlers {
		ids = append(ids, id)
	}
	return ids
}
