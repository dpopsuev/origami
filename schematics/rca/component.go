// Package rca provides an Origami component that bundles the RCA circuit's
// hooks, transformers, and extractors under the "rca" namespace.
package rca

import (
	"github.com/dpopsuev/origami/schematics/rca/rcatype"
	"github.com/dpopsuev/origami/schematics/rca/store"

	framework "github.com/dpopsuev/origami"
	"github.com/dpopsuev/origami/knowledge"
)

// ComponentConfig holds runtime dependencies injected into the RCA component.
type ComponentConfig struct {
	Store     store.Store
	CaseData  *store.Case
	Envelope  *rcatype.Envelope
	Catalog   *knowledge.KnowledgeSourceCatalog
	CaseDir string
}

// Component returns an Origami Component bundling all RCA circuit plumbing
// (store hooks, context-builder transformer, prompt-filler transformer,
// and per-step extractors) under the "rca" namespace.
func Component(cfg ComponentConfig) *framework.Component {
	return &framework.Component{
		Namespace:    "rca",
		Name:         "origami-rca",
		Version:      "1.0.0",
		Description:  "RCA circuit plumbing for CI root-cause analysis",
		Transformers: buildTransformers(cfg),
		Extractors:   buildExtractors(),
		Hooks:        buildHooks(cfg),
	}
}

// allNodeNames lists every RCA circuit node name. Used when registering
// a single monolithic transformer under each node for backwards compat.
var allNodeNames = []string{"recall", "triage", "resolve", "investigate", "correlate", "review", "report"}

// HeuristicComponent returns a Component with per-node heuristic transformers
// that implement deterministic, keyword-based RCA logic.
func HeuristicComponent(st store.Store, repos []string) *framework.Component {
	ht := NewHeuristicTransformer(st, repos)
	return &framework.Component{
		Namespace: "rca",
		Name:      "rca-heuristic",
		Transformers: framework.TransformerRegistry{
			"recall":      &recallHeuristic{ht: ht},
			"triage":      &triageHeuristic{ht: ht},
			"resolve":     &resolveHeuristic{ht: ht},
			"investigate": &investigateHeuristic{ht: ht},
			"correlate":   &correlateHeuristic{ht: ht},
			"review":      &reviewHeuristic{},
			"report":      &reportHeuristic{},
		},
	}
}

// TransformerComponent wraps a monolithic framework.Transformer (e.g. stub, rca)
// and registers it under every node name so that DSL transformer: resolution
// can find it. The transformer's Transform() dispatches on tc.NodeName.
func TransformerComponent(t framework.Transformer) *framework.Component {
	reg := framework.TransformerRegistry{}
	for _, name := range allNodeNames {
		reg[name] = t
	}
	return &framework.Component{
		Namespace:    "rca",
		Name:         "rca-transformer",
		Transformers: reg,
	}
}

// HITLComponent returns a Component with per-node HITL transformers that
// fill prompt templates and return framework.Interrupt for human input.
func HITLComponent() *framework.Component {
	reg := framework.TransformerRegistry{}
	steps := map[string]CircuitStep{
		"recall": StepF0Recall, "triage": StepF1Triage, "resolve": StepF2Resolve,
		"investigate": StepF3Invest, "correlate": StepF4Correlate, "review": StepF5Review,
		"report": StepF6Report,
	}
	for name, step := range steps {
		reg[name] = &hitlTransformerNode{step: step}
	}
	return &framework.Component{
		Namespace:    "rca",
		Name:         "rca-hitl",
		Transformers: reg,
	}
}

func buildTransformers(_ ComponentConfig) framework.TransformerRegistry {
	return framework.TransformerRegistry{}
}

func buildExtractors() framework.ExtractorRegistry {
	reg := framework.ExtractorRegistry{}
	reg["recall"] = NewStepExtractor[RecallResult]("recall")
	reg["triage"] = NewStepExtractor[TriageResult]("triage")
	reg["resolve"] = NewStepExtractor[ResolveResult]("resolve")
	reg["investigate"] = NewStepExtractor[InvestigateArtifact]("investigate")
	reg["correlate"] = NewStepExtractor[CorrelateResult]("correlate")
	reg["review"] = NewStepExtractor[ReviewDecision]("review")
	reg["report"] = NewStepExtractor[InvestigateArtifact]("report")
	return reg
}

func buildHooks(cfg ComponentConfig) framework.HookRegistry {
	reg := framework.HookRegistry{}

	inject := InjectHooks(cfg.Store, cfg.CaseData, cfg.Envelope, cfg.Catalog, cfg.CaseDir)
	for name, h := range inject {
		reg[name] = h
	}

	if cfg.Store != nil && cfg.CaseData != nil {
		hooks := StoreHooks(cfg.Store, cfg.CaseData)
		for name, h := range hooks {
			reg[name] = h
		}
	}
	return reg
}
