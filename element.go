// Package framework defines the generic agent circuit framework.
// It provides interfaces for building directed graphs of processing nodes
// traversed by identity-bearing walkers, with conditional edges governing
// transitions. Domain-specific circuits (e.g. F0-F6 RCA) are instances
// of this generic structure.
package framework

import "fmt"

// Approach is the user-facing name for a behavioral archetype.
// Circuit authors write `approach: analytical` in YAML; the framework
// resolves it to an internal Element for scheduling and traversal.
type Approach string

const (
	ApproachRapid      Approach = "rapid"
	ApproachAggressive Approach = "aggressive"
	ApproachMethodical Approach = "methodical"
	ApproachRigorous   Approach = "rigorous"
	ApproachAnalytical Approach = "analytical"
	ApproachHolistic   Approach = "holistic"
)

// Element represents an internal behavioral archetype governing how an agent
// moves through a circuit graph. Elements are the internal identity behind
// approaches — they drive color coding, personas, and scheduling.
type Element string

const (
	ElementFire      Element = "fire"
	ElementLightning Element = "lightning"
	ElementEarth     Element = "earth"
	ElementDiamond   Element = "diamond"
	ElementWater     Element = "water"
	ElementAir       Element = "air"
)

// SpeedClass describes an element's processing velocity.
type SpeedClass string

const (
	SpeedFastest  SpeedClass = "fastest"
	SpeedFast     SpeedClass = "fast"
	SpeedSteady   SpeedClass = "steady"
	SpeedPrecise  SpeedClass = "precise"
	SpeedDeep     SpeedClass = "deep"
	SpeedHolistic SpeedClass = "holistic"
)

// ElementTraits quantifies an element's behavioral characteristics.
// These traits are used by the scheduler for routing decisions
// and by the graph walker for loop/convergence control.
type ElementTraits struct {
	Element              Element    `json:"element"`
	Speed                SpeedClass `json:"speed"`
	MaxLoops             int        `json:"max_loops"`
	ConvergenceThreshold float64    `json:"convergence_threshold"`
	ShortcutAffinity     float64    `json:"shortcut_affinity"` // 0.0-1.0
	EvidenceDepth        int        `json:"evidence_depth"`
	FailureMode          string     `json:"failure_mode"`
}

var defaultTraits = map[Element]ElementTraits{
	ElementFire: {
		Element: ElementFire, Speed: SpeedFast,
		MaxLoops: 0, ConvergenceThreshold: 0.50,
		ShortcutAffinity: 0.9, EvidenceDepth: 2,
		FailureMode: "burns out (token waste)",
	},
	ElementLightning: {
		Element: ElementLightning, Speed: SpeedFastest,
		MaxLoops: 0, ConvergenceThreshold: 0.40,
		ShortcutAffinity: 1.0, EvidenceDepth: 1,
		FailureMode: "brittle (wrong path, no recovery)",
	},
	ElementEarth: {
		Element: ElementEarth, Speed: SpeedSteady,
		MaxLoops: 1, ConvergenceThreshold: 0.70,
		ShortcutAffinity: 0.1, EvidenceDepth: 5,
		FailureMode: "bloat (too many steps)",
	},
	ElementDiamond: {
		Element: ElementDiamond, Speed: SpeedPrecise,
		MaxLoops: 0, ConvergenceThreshold: 0.95,
		ShortcutAffinity: 0.5, EvidenceDepth: 10,
		FailureMode: "shatters (ambiguity kills it)",
	},
	ElementWater: {
		Element: ElementWater, Speed: SpeedDeep,
		MaxLoops: 3, ConvergenceThreshold: 0.85,
		ShortcutAffinity: 0.1, EvidenceDepth: 8,
		FailureMode: "slow (analysis paralysis)",
	},
	ElementAir: {
		Element: ElementAir, Speed: SpeedHolistic,
		MaxLoops: 1, ConvergenceThreshold: 0.60,
		ShortcutAffinity: 0.6, EvidenceDepth: 3,
		FailureMode: "floaty (vague, no evidence)",
	},
}

var coreElements = []Element{
	ElementFire, ElementLightning, ElementEarth,
	ElementDiamond, ElementWater, ElementAir,
}

// DefaultTraits returns the canonical trait set for a given element.
// Returns a zero-value ElementTraits if the element is unknown.
func DefaultTraits(e Element) ElementTraits {
	return defaultTraits[e]
}

// AllElements returns the six core elements.
func AllElements() []Element {
	out := make([]Element, len(coreElements))
	copy(out, coreElements)
	return out
}

// --- Approach ↔ Element mapping ---

var approachToElement = map[Approach]Element{
	ApproachRapid:      ElementFire,
	ApproachAggressive: ElementLightning,
	ApproachMethodical: ElementEarth,
	ApproachRigorous:   ElementDiamond,
	ApproachAnalytical: ElementWater,
	ApproachHolistic:   ElementAir,
}

var elementToApproach = map[Element]Approach{
	ElementFire:      ApproachRapid,
	ElementLightning: ApproachAggressive,
	ElementEarth:     ApproachMethodical,
	ElementDiamond:   ApproachRigorous,
	ElementWater:     ApproachAnalytical,
	ElementAir:       ApproachHolistic,
}

var approachEmoji = map[Approach]string{
	ApproachRapid:      "🔥",
	ApproachAggressive: "⚡",
	ApproachMethodical: "🪨",
	ApproachRigorous:   "💎",
	ApproachAnalytical: "💧",
	ApproachHolistic:   "🌀",
}

var coreApproaches = []Approach{
	ApproachRapid, ApproachAggressive, ApproachMethodical,
	ApproachRigorous, ApproachAnalytical, ApproachHolistic,
}

// ResolveApproach maps a user-facing approach name to an internal Element.
func ResolveApproach(name string) (Element, bool) {
	e, ok := approachToElement[Approach(name)]
	return e, ok
}

// ApproachForElement returns the user-facing approach name for an element.
func ApproachForElement(e Element) Approach {
	return elementToApproach[e]
}

// ApproachEmoji returns the emoji for an approach.
func ApproachEmoji(a Approach) string {
	return approachEmoji[a]
}

// ApproachTraits returns the ElementTraits for an approach.
func ApproachTraits(a Approach) ElementTraits {
	return defaultTraits[approachToElement[a]]
}

// ApproachTraitsSummary returns a formatted multi-line summary for LSP hover.
func ApproachTraitsSummary(a Approach) string {
	t := ApproachTraits(a)
	if t.Element == "" {
		return ""
	}
	return fmt.Sprintf(
		"Speed:          %s\nThoroughness:   %d evidence, %d loops\nConfidence bar: %.2f\nSkip tolerance: %.1f",
		t.Speed, t.EvidenceDepth, t.MaxLoops, t.ConvergenceThreshold, t.ShortcutAffinity,
	)
}

// AllApproaches returns the six core approaches.
func AllApproaches() []Approach {
	out := make([]Approach, len(coreApproaches))
	copy(out, coreApproaches)
	return out
}
