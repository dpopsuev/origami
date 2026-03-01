package framework

import "time"

// DialecticConfig controls the adversarial dialectic pipeline activation and limits.
// When the Thesis path's confidence falls below the contradiction threshold,
// the adversarial path activates for thesis-antithesis-synthesis debate.
type DialecticConfig struct {
	Enabled                bool          `json:"enabled"`
	TTL                    time.Duration `json:"ttl"`
	MaxTurns               int           `json:"max_turns"`
	MaxNegations           int           `json:"max_negations"`
	ContradictionThreshold float64       `json:"contradiction_threshold"`
}

// DefaultDialecticConfig returns conservative defaults for the dialectic pipeline.
func DefaultDialecticConfig() DialecticConfig {
	return DialecticConfig{
		Enabled:                false,
		TTL:                    10 * time.Minute,
		MaxTurns:               6,
		MaxNegations:           2,
		ContradictionThreshold: 0.85,
	}
}

// NeedsAntithesis returns true when a Thesis path confidence falls in the
// uncertain range that triggers adversarial dialectic review.
func (c DialecticConfig) NeedsAntithesis(confidence float64) bool {
	if !c.Enabled {
		return false
	}
	return confidence >= 0.50 && confidence < c.ContradictionThreshold
}

// SynthesisDecision represents the outcome of the adversarial dialectic.
type SynthesisDecision string

const (
	SynthesisAffirm    SynthesisDecision = "affirm"
	SynthesisAmend     SynthesisDecision = "amend"
	SynthesisAcquit    SynthesisDecision = "acquit"
	SynthesisRemand    SynthesisDecision = "remand"
	SynthesisUnresolved SynthesisDecision = "unresolved"
)

// EvidenceItem is a single piece of evidence with an assigned weight.
type EvidenceItem struct {
	Description string  `json:"description"`
	Source      string  `json:"source"`
	Weight      float64 `json:"weight"`
}

// ThesisChallenge is the D0 thesis-holder artifact: charged defect type with
// itemized evidence and a thesis-holder narrative.
type ThesisChallenge struct {
	ChargedDefectType string         `json:"charged_defect_type"`
	ThesisNarrative   string         `json:"thesis_narrative"`
	Evidence          []EvidenceItem `json:"evidence"`
	ConfidenceScore   float64        `json:"confidence"`
}

func (t *ThesisChallenge) Type() string       { return "thesis_challenge" }
func (t *ThesisChallenge) Confidence() float64 { return t.ConfidenceScore }
func (t *ThesisChallenge) Raw() any            { return t }

// EvidenceChallenge captures a specific challenge to an evidence item.
type EvidenceChallenge struct {
	EvidenceIndex int    `json:"evidence_index"`
	Challenge     string `json:"challenge"`
	Severity      string `json:"severity"`
}

// AntithesisResponse is the D2 antithesis-holder artifact: challenges to evidence,
// alternative hypothesis, and concession flag.
type AntithesisResponse struct {
	Challenges            []EvidenceChallenge `json:"challenges"`
	AlternativeHypothesis string              `json:"alternative_hypothesis,omitempty"`
	Concession            bool                `json:"concession"`
	ConfidenceScore       float64             `json:"confidence"`
}

func (a *AntithesisResponse) Type() string       { return "antithesis_response" }
func (a *AntithesisResponse) Confidence() float64 { return a.ConfidenceScore }
func (a *AntithesisResponse) Raw() any            { return a }

// DialecticRound captures one round of thesis argument, antithesis
// rebuttal, and arbiter notes.
type DialecticRound struct {
	Round             int    `json:"round"`
	ThesisArgument    string `json:"thesis_argument"`
	AntithesisRebuttal string `json:"antithesis_rebuttal"`
	ArbiterNotes      string `json:"arbiter_notes"`
}

// DialecticRecord is the D3 dialectic artifact: rounds of structured debate.
type DialecticRecord struct {
	Rounds    []DialecticRound `json:"rounds"`
	MaxRounds int              `json:"max_rounds"`
	Converged bool             `json:"converged"`
}

func (d *DialecticRecord) Type() string       { return "dialectic_record" }
func (d *DialecticRecord) Confidence() float64 { return 0 }
func (d *DialecticRecord) Raw() any            { return d }

// Synthesis is the D4 final decision artifact.
type Synthesis struct {
	Decision            SynthesisDecision  `json:"decision"`
	FinalClassification string             `json:"final_classification"`
	ConfidenceScore     float64            `json:"confidence"`
	Reasoning           string             `json:"reasoning"`
	NegationFeedback    *NegationFeedback  `json:"negation_feedback,omitempty"`
}

func (s *Synthesis) Type() string       { return "synthesis" }
func (s *Synthesis) Confidence() float64 { return s.ConfidenceScore }
func (s *Synthesis) Raw() any            { return s }

// NegationFeedback provides structured feedback when a case is remanded
// back to the Thesis path for reinvestigation.
type NegationFeedback struct {
	ChallengedEvidence []int    `json:"challenged_evidence"`
	AlternativeHyp     string   `json:"alternative_hypothesis"`
	SpecificQuestions  []string `json:"specific_questions"`
}

// DialecticEvidenceGap extends EvidenceGap with dialectic-specific context.
// It embeds the shared EvidenceGap type so dialectic gaps can be collected
// into an EvidenceGapBrief on unresolved contradiction.
type DialecticEvidenceGap struct {
	EvidenceGap
	DialecticPhase string `json:"dialectic_phase,omitempty"`
}

// BuildDialecticEdgeFactory returns an EdgeFactory with skeleton dialectic
// evaluators (HD1-HD12) for the adversarial dialectic pipeline. Each evaluator
// checks the artifact type and dialectic-specific conditions.
func BuildDialecticEdgeFactory(cfg DialecticConfig) EdgeFactory {
	return EdgeFactory{
		"HD1": dialecticEdgeFactory(func(a Artifact, _ *WalkerState) *Transition {
			tc, ok := unwrapThesisChallenge(a)
			if !ok {
				return nil
			}
			if tc.ConfidenceScore >= 0.95 {
				return &Transition{NextNode: "defend", Explanation: "fast-track: thesis confidence >= 0.95"}
			}
			return nil
		}),
		"HD2": dialecticEdgeFactory(func(a Artifact, _ *WalkerState) *Transition {
			ar, ok := unwrapAntithesisResponse(a)
			if !ok {
				return nil
			}
			if ar.Concession {
				return &Transition{NextNode: "verdict", Explanation: "concession: antithesis-holder concedes"}
			}
			return nil
		}),
		"HD3": dialecticEdgeFactory(func(a Artifact, _ *WalkerState) *Transition {
			ar, ok := unwrapAntithesisResponse(a)
			if !ok {
				return nil
			}
			if len(ar.Challenges) > 0 && ar.AlternativeHypothesis == "" {
				return &Transition{NextNode: "hearing", Explanation: "partial negation: challenges without alternative"}
			}
			return nil
		}),
		"HD4": dialecticEdgeFactory(func(a Artifact, _ *WalkerState) *Transition {
			ar, ok := unwrapAntithesisResponse(a)
			if !ok {
				return nil
			}
			if ar.AlternativeHypothesis != "" {
				return &Transition{NextNode: "hearing", Explanation: "alternative hypothesis presented"}
			}
			return nil
		}),
		"HD5": dialecticEdgeFactory(func(a Artifact, s *WalkerState) *Transition {
			rec, ok := unwrapDialecticRecord(a)
			if !ok {
				return nil
			}
			if rec.Converged || len(rec.Rounds) >= rec.MaxRounds {
				return &Transition{NextNode: "verdict", Explanation: "dialectic complete"}
			}
			return nil
		}),
		"HD6": dialecticEdgeFactory(func(a Artifact, _ *WalkerState) *Transition {
			s, ok := unwrapSynthesis(a)
			if !ok {
				return nil
			}
			if s.Decision == SynthesisAffirm {
				return &Transition{NextNode: "_done", Explanation: "synthesis: affirm"}
			}
			return nil
		}),
		"HD7": dialecticEdgeFactory(func(a Artifact, _ *WalkerState) *Transition {
			s, ok := unwrapSynthesis(a)
			if !ok {
				return nil
			}
			if s.Decision == SynthesisAmend {
				return &Transition{NextNode: "_done", Explanation: "synthesis: amend"}
			}
			return nil
		}),
		"HD8": dialecticEdgeFactory(func(a Artifact, ws *WalkerState) *Transition {
			s, ok := unwrapSynthesis(a)
			if !ok {
				return nil
			}
			if s.Decision == SynthesisRemand && ws.LoopCounts["verdict"] < cfg.MaxNegations {
				return &Transition{
					NextNode:    "indict",
					Explanation: "synthesis: remand for reinvestigation",
				}
			}
			return nil
		}),
		"HD9": dialecticEdgeFactory(func(a Artifact, _ *WalkerState) *Transition {
			s, ok := unwrapSynthesis(a)
			if !ok {
				return nil
			}
			if s.Decision == SynthesisAcquit {
				return &Transition{NextNode: "_done", Explanation: "synthesis: acquit (evidence gap brief)"}
			}
			return nil
		}),
		"HD10": dialecticEdgeFactory(func(_ Artifact, ws *WalkerState) *Transition {
			if ws.LoopCounts["_handoff"] > cfg.MaxTurns {
				return &Transition{NextNode: "_done", Explanation: "unresolved contradiction: turn limit exceeded"}
			}
			return nil
		}),
		"HD11": dialecticEdgeFactory(func(_ Artifact, ws *WalkerState) *Transition {
			if ws.LoopCounts["_handoff"] > cfg.MaxTurns {
				return &Transition{NextNode: "_done", Explanation: "unresolved contradiction: turn counter exceeded"}
			}
			return nil
		}),
		"HD12": dialecticEdgeFactory(func(a Artifact, _ *WalkerState) *Transition {
			s, ok := unwrapSynthesis(a)
			if !ok {
				return nil
			}
			if s.Decision == SynthesisUnresolved {
				return &Transition{NextNode: "_done", Explanation: "synthesis: unresolved contradiction declared by arbiter"}
			}
			return nil
		}),
	}
}

type dialecticEvalFunc func(Artifact, *WalkerState) *Transition

func dialecticEdgeFactory(eval dialecticEvalFunc) func(EdgeDef) Edge {
	return func(def EdgeDef) Edge {
		return &dialecticEdge{def: def, eval: eval}
	}
}

type dialecticEdge struct {
	def  EdgeDef
	eval dialecticEvalFunc
}

func (e *dialecticEdge) ID() string       { return e.def.ID }
func (e *dialecticEdge) From() string     { return e.def.From }
func (e *dialecticEdge) To() string       { return e.def.To }
func (e *dialecticEdge) IsShortcut() bool { return e.def.Shortcut }
func (e *dialecticEdge) IsLoop() bool     { return e.def.Loop }
func (e *dialecticEdge) Evaluate(a Artifact, s *WalkerState) *Transition {
	return e.eval(a, s)
}

func unwrapThesisChallenge(a Artifact) (*ThesisChallenge, bool) {
	if a == nil {
		return nil, false
	}
	tc, ok := a.Raw().(*ThesisChallenge)
	return tc, ok
}

func unwrapAntithesisResponse(a Artifact) (*AntithesisResponse, bool) {
	if a == nil {
		return nil, false
	}
	ar, ok := a.Raw().(*AntithesisResponse)
	return ar, ok
}

func unwrapDialecticRecord(a Artifact) (*DialecticRecord, bool) {
	if a == nil {
		return nil, false
	}
	rec, ok := a.Raw().(*DialecticRecord)
	return rec, ok
}

func unwrapSynthesis(a Artifact) (*Synthesis, bool) {
	if a == nil {
		return nil, false
	}
	s, ok := a.Raw().(*Synthesis)
	return s, ok
}
