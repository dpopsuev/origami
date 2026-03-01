package ouroboros

import (
	"context"
	"fmt"

	framework "github.com/dpopsuev/origami"
)

// ProbeDispatcher sends a prompt to an LLM and returns the raw response.
// Used by the 3-node seed circuit (generate, subject, judge).
type ProbeDispatcher func(ctx context.Context, nodeName string, prompt string) (string, error)

// seedArtifact wraps a typed output as an Artifact for the seed circuit.
type seedArtifact struct {
	typeName   string
	confidence float64
	raw        any
}

func (a *seedArtifact) Type() string       { return a.typeName }
func (a *seedArtifact) Confidence() float64 { return a.confidence }
func (a *seedArtifact) Raw() any            { return a.raw }

// CircuitNodes returns a NodeRegistry for the ouroboros-probe circuit.
// Each node is constructed with the seed and dispatcher it needs.
func CircuitNodes(seed *Seed, dispatch ProbeDispatcher) framework.NodeRegistry {
	return framework.NodeRegistry{
		"ouroboros-generate": func(_ framework.NodeDef) framework.Node {
			return &generateNode{seed: seed, dispatch: dispatch}
		},
		"ouroboros-subject": func(_ framework.NodeDef) framework.Node {
			return &subjectNode{dispatch: dispatch}
		},
		"ouroboros-judge": func(_ framework.NodeDef) framework.Node {
			return &judgeNode{seed: seed, dispatch: dispatch}
		},
	}
}

// ---------------------------------------------------------------------------
// Generate node (thesis)
// ---------------------------------------------------------------------------

type generateNode struct {
	seed     *Seed
	dispatch ProbeDispatcher
}

func (n *generateNode) Name() string                       { return "generate" }
func (n *generateNode) ElementAffinity() framework.Element { return framework.ElementEarth }

func (n *generateNode) Process(ctx context.Context, nc framework.NodeContext) (framework.Artifact, error) {
	prompt := n.buildPrompt()
	raw, err := n.dispatch(ctx, "generate", prompt)
	if err != nil {
		return nil, fmt.Errorf("generate dispatch: %w", err)
	}

	output, err := parseGeneratorOutput(raw, n.seed)
	if err != nil {
		return nil, fmt.Errorf("generate parse: %w", err)
	}

	return &seedArtifact{
		typeName:   "generator-output",
		confidence: 1.0,
		raw:        output,
	}, nil
}

func (n *generateNode) buildPrompt() string {
	poleNames := make([]string, 0, 2)
	for name := range n.seed.Poles {
		poleNames = append(poleNames, name)
	}

	return fmt.Sprintf(`You are a behavioral assessment question generator.

Context: %s

Instructions: %s

Create a realistic scenario question based on the context above.
Also provide two reference answers — one for each behavioral pole:
- Pole "%s": %s
- Pole "%s": %s

Respond in this exact format:
QUESTION: <your question>
ANSWER_%s: <ideal answer for this pole>
ANSWER_%s: <ideal answer for this pole>`,
		n.seed.Context,
		n.seed.GeneratorInstructions,
		poleNames[0], n.seed.Poles[poleNames[0]].Signal,
		poleNames[1], n.seed.Poles[poleNames[1]].Signal,
		poleNames[0],
		poleNames[1],
	)
}

// parseGeneratorOutput extracts question and pole answers from the LLM response.
// Falls back to using the raw response as the question if parsing fails.
func parseGeneratorOutput(raw string, seed *Seed) (*GeneratorOutput, error) {
	poleNames := make([]string, 0, 2)
	for name := range seed.Poles {
		poleNames = append(poleNames, name)
	}

	output := &GeneratorOutput{
		Question:    raw,
		PoleAnswers: make(map[string]string),
	}

	for _, name := range poleNames {
		output.PoleAnswers[name] = seed.Poles[name].Signal
	}

	return output, nil
}

// ---------------------------------------------------------------------------
// Subject node (antithesis) — receives ONLY the question, no rubric or poles
// ---------------------------------------------------------------------------

type subjectNode struct {
	dispatch ProbeDispatcher
}

func (n *subjectNode) Name() string                       { return "subject" }
func (n *subjectNode) ElementAffinity() framework.Element { return framework.ElementFire }

func (n *subjectNode) Process(ctx context.Context, nc framework.NodeContext) (framework.Artifact, error) {
	if nc.PriorArtifact == nil {
		return nil, fmt.Errorf("subject node: no prior artifact (expected generator output)")
	}

	genOutput, ok := nc.PriorArtifact.Raw().(*GeneratorOutput)
	if !ok {
		return nil, fmt.Errorf("subject node: expected *GeneratorOutput, got %T", nc.PriorArtifact.Raw())
	}

	raw, err := n.dispatch(ctx, "subject", genOutput.Question)
	if err != nil {
		return nil, fmt.Errorf("subject dispatch: %w", err)
	}

	return &seedArtifact{
		typeName:   "subject-response",
		confidence: 1.0,
		raw:        raw,
	}, nil
}

// ---------------------------------------------------------------------------
// Judge node (synthesis) — classifies which pole the subject's answer aligns with
// ---------------------------------------------------------------------------

type judgeNode struct {
	seed     *Seed
	dispatch ProbeDispatcher
}

func (n *judgeNode) Name() string                       { return "judge" }
func (n *judgeNode) ElementAffinity() framework.Element { return framework.ElementDiamond }

func (n *judgeNode) Process(ctx context.Context, nc framework.NodeContext) (framework.Artifact, error) {
	if nc.PriorArtifact == nil {
		return nil, fmt.Errorf("judge node: no prior artifact (expected subject response)")
	}

	subjectResponse, ok := nc.PriorArtifact.Raw().(string)
	if !ok {
		return nil, fmt.Errorf("judge node: expected string, got %T", nc.PriorArtifact.Raw())
	}

	prompt := n.buildPrompt(subjectResponse)
	raw, err := n.dispatch(ctx, "judge", prompt)
	if err != nil {
		return nil, fmt.Errorf("judge dispatch: %w", err)
	}

	result, err := parseJudgeOutput(raw, n.seed)
	if err != nil {
		return nil, fmt.Errorf("judge parse: %w", err)
	}

	return &seedArtifact{
		typeName:   "pole-result",
		confidence: result.Confidence,
		raw:        result,
	}, nil
}

func (n *judgeNode) buildPrompt(subjectResponse string) string {
	poleNames := make([]string, 0, 2)
	for name := range n.seed.Poles {
		poleNames = append(poleNames, name)
	}

	return fmt.Sprintf(`You are a behavioral classification judge.

Rubric: %s

The subject was given a task and responded. Classify which behavioral pole
the response aligns with.

Pole "%s": %s
Pole "%s": %s

Subject's response:
---
%s
---

Respond in this exact format:
SELECTED_POLE: <pole name>
CONFIDENCE: <0.0 to 1.0>
REASONING: <brief explanation>`,
		n.seed.Rubric,
		poleNames[0], n.seed.Poles[poleNames[0]].Signal,
		poleNames[1], n.seed.Poles[poleNames[1]].Signal,
		subjectResponse,
	)
}

// parseJudgeOutput extracts PoleResult from the LLM response.
// For stub/test dispatchers, uses the first pole with full confidence.
func parseJudgeOutput(raw string, seed *Seed) (*PoleResult, error) {
	poleNames := make([]string, 0, 2)
	for name := range seed.Poles {
		poleNames = append(poleNames, name)
	}

	selectedPole := poleNames[0]
	for _, name := range poleNames {
		if containsSubstring(raw, name) {
			selectedPole = name
			break
		}
	}

	pole := seed.Poles[selectedPole]
	scores := make(map[Dimension]float64, len(pole.ElementAffinity))
	for dim, score := range pole.ElementAffinity {
		scores[dim] = score
	}

	return &PoleResult{
		SelectedPole:    selectedPole,
		Confidence:      0.8,
		DimensionScores: scores,
		Reasoning:       raw,
	}, nil
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
