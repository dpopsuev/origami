package calibrate

import (
	"crypto/sha256"
	"fmt"
	"sort"
)

// StepAccuracy captures per-step performance from calibration results.
// It identifies which pipeline steps are performing well and which are
// dragging down overall accuracy.
type StepAccuracy struct {
	Step     string  `json:"step"`
	Accuracy float64 `json:"accuracy"`
	Samples  int     `json:"samples"`
	Rank     int     `json:"rank"`
}

// StepAnalyzer computes per-step accuracy from calibration data.
// Consumers provide domain-specific scoring via the StepScorer function.
type StepAnalyzer struct {
	scorer StepScorer
}

// StepScorer evaluates how accurately a step performed for a given case.
// Returns a value between 0.0 (completely wrong) and 1.0 (perfect).
type StepScorer func(caseID, step string, metrics MetricSet) float64

// NewStepAnalyzer creates a StepAnalyzer with the given scoring function.
func NewStepAnalyzer(scorer StepScorer) *StepAnalyzer {
	return &StepAnalyzer{scorer: scorer}
}

// StepResult pairs a case ID and step name for analysis.
type StepResult struct {
	CaseID  string
	Step    string
	Metrics MetricSet
}

// Analyze computes per-step accuracy rankings from a set of step results.
// Returns steps sorted by accuracy ascending (worst performers first).
func (a *StepAnalyzer) Analyze(results []StepResult) []StepAccuracy {
	type accumulator struct {
		total float64
		count int
	}
	byStep := make(map[string]*accumulator)

	for _, r := range results {
		score := a.scorer(r.CaseID, r.Step, r.Metrics)
		acc, ok := byStep[r.Step]
		if !ok {
			acc = &accumulator{}
			byStep[r.Step] = acc
		}
		acc.total += score
		acc.count++
	}

	out := make([]StepAccuracy, 0, len(byStep))
	for step, acc := range byStep {
		out = append(out, StepAccuracy{
			Step:     step,
			Accuracy: acc.total / float64(acc.count),
			Samples:  acc.count,
		})
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Accuracy < out[j].Accuracy
	})
	for i := range out {
		out[i].Rank = i + 1
	}
	return out
}

// TuningProposal recommends a prompt change for a specific pipeline step.
type TuningProposal struct {
	Step          string  `json:"step"`
	CurrentHash   string  `json:"current_hash"`
	Suggestion    string  `json:"suggestion"`
	Rationale     string  `json:"rationale"`
	ExpectedDelta float64 `json:"expected_delta"`
	Accepted      bool    `json:"accepted,omitempty"`
	RejectReason  string  `json:"reject_reason,omitempty"`
}

// PromptCalibrationLoop iterates over tuning proposals for the worst-performing
// steps. The consumer calls Next() to get proposals and Accept()/Reject() to
// provide feedback. The loop is framework-level; prompt templates are domain-level.
type PromptCalibrationLoop struct {
	proposals []TuningProposal
	index     int
	history   []TuningProposal
}

// NewPromptCalibrationLoop creates a loop from step accuracy rankings and
// a proposer function. The proposer generates a TuningProposal for each
// step that falls below the accuracy threshold.
func NewPromptCalibrationLoop(
	rankings []StepAccuracy,
	threshold float64,
	proposer func(StepAccuracy) *TuningProposal,
) *PromptCalibrationLoop {
	var proposals []TuningProposal
	for _, sa := range rankings {
		if sa.Accuracy >= threshold {
			continue
		}
		p := proposer(sa)
		if p != nil {
			proposals = append(proposals, *p)
		}
	}
	return &PromptCalibrationLoop{proposals: proposals}
}

// HasNext returns true if there are more proposals to review.
func (l *PromptCalibrationLoop) HasNext() bool {
	return l.index < len(l.proposals)
}

// Next returns the next tuning proposal, or nil if exhausted.
func (l *PromptCalibrationLoop) Next() *TuningProposal {
	if !l.HasNext() {
		return nil
	}
	p := l.proposals[l.index]
	return &p
}

// Accept marks the current proposal as accepted and advances.
func (l *PromptCalibrationLoop) Accept() {
	if !l.HasNext() {
		return
	}
	p := l.proposals[l.index]
	p.Accepted = true
	l.history = append(l.history, p)
	l.index++
}

// Reject marks the current proposal as rejected with a reason and advances.
func (l *PromptCalibrationLoop) Reject(reason string) {
	if !l.HasNext() {
		return
	}
	p := l.proposals[l.index]
	p.Accepted = false
	p.RejectReason = reason
	l.history = append(l.history, p)
	l.index++
}

// History returns all reviewed proposals (accepted and rejected).
func (l *PromptCalibrationLoop) History() []TuningProposal {
	return l.history
}

// AcceptedProposals returns only accepted proposals.
func (l *PromptCalibrationLoop) AcceptedProposals() []TuningProposal {
	var out []TuningProposal
	for _, p := range l.history {
		if p.Accepted {
			out = append(out, p)
		}
	}
	return out
}

// Remaining returns the count of unreviewed proposals.
func (l *PromptCalibrationLoop) Remaining() int {
	if l.index >= len(l.proposals) {
		return 0
	}
	return len(l.proposals) - l.index
}

// PromptHash returns a short SHA-256 hash of a prompt string,
// useful for tracking which prompt version was used.
func PromptHash(prompt string) string {
	h := sha256.Sum256([]byte(prompt))
	return fmt.Sprintf("%x", h[:8])
}
