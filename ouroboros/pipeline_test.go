package ouroboros

import (
	"context"
	"fmt"
	"os"
	"testing"

	framework "github.com/dpopsuev/origami"
)

func testSeed() *Seed {
	return &Seed{
		Name:    "test-probe",
		Version: "1.0",
		Dimensions: []Dimension{DimSpeed, DimEvidenceDepth},
		Category: CategorySkill,
		Poles: map[string]Pole{
			"systematic": {
				Signal: "thorough, step-by-step analysis",
				ElementAffinity: map[Dimension]float64{
					DimSpeed:         0.3,
					DimEvidenceDepth: 0.9,
				},
			},
			"heuristic": {
				Signal: "pattern matching, quick answer",
				ElementAffinity: map[Dimension]float64{
					DimSpeed:         0.9,
					DimEvidenceDepth: 0.3,
				},
			},
		},
		Context:               "You are reviewing a complex Go function with subtle concurrency bugs.",
		Rubric:                "Evaluate whether the response uses systematic analysis or heuristic shortcuts.",
		GeneratorInstructions: "Create a realistic code review scenario with a goroutine leak.",
	}
}

func stubProbeDispatcher(responses map[string]string) ProbeDispatcher {
	return func(ctx context.Context, nodeName string, prompt string) (string, error) {
		resp, ok := responses[nodeName]
		if !ok {
			return fmt.Sprintf("stub response for %s", nodeName), nil
		}
		return resp, nil
	}
}

func TestPipelineWalk_FullFlow(t *testing.T) {
	seed := testSeed()
	dispatcher := stubProbeDispatcher(map[string]string{
		"generate": "QUESTION: Review this goroutine leak\nANSWER_systematic: Analyze the context cancellation path\nANSWER_heuristic: Just add a defer cancel()",
		"subject":  "I would carefully trace the goroutine lifecycle, check context propagation, and verify all channels are properly closed. This is a systematic approach.",
		"judge":    "SELECTED_POLE: systematic\nCONFIDENCE: 0.85\nREASONING: The response shows thorough analysis.",
	})

	pipelineYAML, err := os.ReadFile("pipelines/ouroboros-probe.yaml")
	if err != nil {
		t.Fatalf("read pipeline YAML: %v", err)
	}
	def, err := framework.LoadPipeline(pipelineYAML)
	if err != nil {
		t.Fatalf("load pipeline: %v", err)
	}

	nodes := PipelineNodes(seed, dispatcher)
	g, err := def.BuildGraph(framework.GraphRegistries{Nodes: nodes})
	if err != nil {
		t.Fatalf("build graph: %v", err)
	}

	walker := framework.NewProcessWalker("test-walk")
	ctx := context.Background()

	if err := g.Walk(ctx, walker, def.Start); err != nil {
		t.Fatalf("walk failed: %v", err)
	}

	if walker.State().Status != "done" {
		t.Errorf("walker status = %q, want done", walker.State().Status)
	}

	history := walker.State().History
	if len(history) != 3 {
		t.Fatalf("history length = %d, want 3 (generate, subject, judge)", len(history))
	}
	if history[0].Node != "generate" {
		t.Errorf("history[0].Node = %q, want generate", history[0].Node)
	}
	if history[1].Node != "subject" {
		t.Errorf("history[1].Node = %q, want subject", history[1].Node)
	}
	if history[2].Node != "judge" {
		t.Errorf("history[2].Node = %q, want judge", history[2].Node)
	}

	judgeArtifact := walker.State().Outputs["judge"]
	if judgeArtifact == nil {
		t.Fatal("judge artifact is nil")
	}
	result, ok := judgeArtifact.Raw().(*PoleResult)
	if !ok {
		t.Fatalf("judge artifact raw type = %T, want *PoleResult", judgeArtifact.Raw())
	}
	if result.SelectedPole != "systematic" {
		t.Errorf("selected pole = %q, want systematic", result.SelectedPole)
	}
	if len(result.DimensionScores) == 0 {
		t.Error("dimension scores are empty")
	}
	if result.DimensionScores[DimEvidenceDepth] != 0.9 {
		t.Errorf("evidence_depth score = %v, want 0.9", result.DimensionScores[DimEvidenceDepth])
	}
}

func TestPipelineNodes_AllRegistered(t *testing.T) {
	seed := testSeed()
	nodes := PipelineNodes(seed, stubProbeDispatcher(nil))

	for _, family := range []string{"ouroboros-generate", "ouroboros-subject", "ouroboros-judge"} {
		factory, ok := nodes[family]
		if !ok {
			t.Errorf("missing node factory for family %q", family)
			continue
		}
		node := factory(framework.NodeDef{})
		if node == nil {
			t.Errorf("factory for %q returned nil", family)
		}
	}
}

func TestSubjectNode_OnlySeesQuestion(t *testing.T) {
	var capturedPrompt string
	dispatcher := func(ctx context.Context, nodeName string, prompt string) (string, error) {
		if nodeName == "subject" {
			capturedPrompt = prompt
		}
		return "stub response", nil
	}

	node := &subjectNode{dispatch: dispatcher}

	genOutput := &GeneratorOutput{
		Question: "What would you do about this goroutine leak?",
		PoleAnswers: map[string]string{
			"systematic": "Trace the lifecycle",
			"heuristic":  "Add defer cancel()",
		},
	}

	nc := framework.NodeContext{
		PriorArtifact: &seedArtifact{
			typeName:   "generator-output",
			confidence: 1.0,
			raw:        genOutput,
		},
	}

	_, err := node.Process(context.Background(), nc)
	if err != nil {
		t.Fatalf("subject Process: %v", err)
	}

	if capturedPrompt != genOutput.Question {
		t.Errorf("subject received more than the question:\ngot:  %q\nwant: %q", capturedPrompt, genOutput.Question)
	}
}

func TestJudgeNode_ProducesPoleResult(t *testing.T) {
	s := testSeed()
	dispatcher := stubProbeDispatcher(map[string]string{
		"judge": "SELECTED_POLE: heuristic\nCONFIDENCE: 0.7\nREASONING: Quick answer pattern",
	})

	node := &judgeNode{seed: s, dispatch: dispatcher}
	nc := framework.NodeContext{
		PriorArtifact: &seedArtifact{
			typeName:   "subject-response",
			confidence: 1.0,
			raw:        "I'd just add defer cancel() and move on",
		},
	}

	artifact, err := node.Process(context.Background(), nc)
	if err != nil {
		t.Fatalf("judge Process: %v", err)
	}

	result, ok := artifact.Raw().(*PoleResult)
	if !ok {
		t.Fatalf("judge artifact raw = %T, want *PoleResult", artifact.Raw())
	}

	if result.SelectedPole != "heuristic" {
		t.Errorf("selected pole = %q, want heuristic", result.SelectedPole)
	}
	if result.DimensionScores[DimSpeed] != 0.9 {
		t.Errorf("speed score = %v, want 0.9", result.DimensionScores[DimSpeed])
	}
}
