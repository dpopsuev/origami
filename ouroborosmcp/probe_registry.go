package ouroborosmcp

import (
	"fmt"

	"github.com/dpopsuev/origami/ouroboros"
	"github.com/dpopsuev/origami/ouroboros/probes"
)

// ProbeHandler binds a probe's stimulus, prompt builder, scorer, and response
// parsing strategy. The MCP session uses this to generate prompts and score
// responses for any registered probe.
type ProbeHandler struct {
	ID             string
	Stimulus       probes.ProbeStimulus
	BuildPrompt    func(probes.ProbeStimulus) string
	Score          func(string) map[ouroboros.Dimension]float64
	NeedsCodeBlock bool
}

// Prompt returns the full prompt text, applying BuildPrompt to the handler's Stimulus.
func (h *ProbeHandler) Prompt() string {
	return h.BuildPrompt(h.Stimulus)
}

// ProbeRegistry maps probe IDs to their handlers.
type ProbeRegistry struct {
	handlers map[string]*ProbeHandler
}

// NewProbeRegistry creates a registry pre-populated with all 5 Ouroboros probes
// using DefaultStimuli(). Use NewProbeRegistryWith() to supply custom stimuli.
func NewProbeRegistry() *ProbeRegistry {
	return NewProbeRegistryWith(probes.DefaultStimuli())
}

// NewProbeRegistryWith creates a registry using the provided stimuli set.
// Missing stimuli fall back to DefaultStimuli().
func NewProbeRegistryWith(stimuli probes.StimuliSet) *ProbeRegistry {
	defaults := probes.DefaultStimuli()
	get := func(name string) probes.ProbeStimulus {
		if s, ok := stimuli[name]; ok {
			return s
		}
		return defaults[name]
	}

	r := &ProbeRegistry{handlers: make(map[string]*ProbeHandler)}

	r.Register(&ProbeHandler{
		ID:             "refactor-v1",
		Stimulus:       get("refactor"),
		BuildPrompt:    probes.BuildRefactorPrompt,
		Score:          probes.ScoreRefactor,
		NeedsCodeBlock: true,
	})
	r.Register(&ProbeHandler{
		ID:             "debug-v1",
		Stimulus:       get("debug"),
		BuildPrompt:    probes.BuildDebugPrompt,
		Score:          probes.ScoreDebug,
		NeedsCodeBlock: false,
	})
	r.Register(&ProbeHandler{
		ID:             "summarize-v1",
		Stimulus:       get("summarize"),
		BuildPrompt:    probes.BuildSummarizePrompt,
		Score:          probes.ScoreSummarize,
		NeedsCodeBlock: false,
	})
	r.Register(&ProbeHandler{
		ID:             "ambiguity-v1",
		Stimulus:       get("ambiguity"),
		BuildPrompt:    probes.BuildAmbiguityPrompt,
		Score:          probes.ScoreAmbiguity,
		NeedsCodeBlock: false,
	})
	r.Register(&ProbeHandler{
		ID:             "persistence-v1",
		Stimulus:       get("persistence"),
		BuildPrompt:    probes.BuildPersistencePrompt,
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
