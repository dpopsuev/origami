package framework_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	framework "github.com/dpopsuev/origami"
	"github.com/dpopsuev/origami/ouroboros"
)

func TestE2E_OuroborosProbe(t *testing.T) {
	seed := &ouroboros.Seed{
		Name:       "e2e-probe",
		Version:    "1.0",
		Dimensions: []ouroboros.Dimension{"speed", "evidence_depth"},
		Category:   ouroboros.CategorySkill,
		Poles: map[string]ouroboros.Pole{
			"systematic": {
				Signal: "thorough analysis",
				ElementAffinity: map[ouroboros.Dimension]float64{
					"speed": 0.3, "evidence_depth": 0.9,
				},
			},
			"heuristic": {
				Signal: "quick pattern match",
				ElementAffinity: map[ouroboros.Dimension]float64{
					"speed": 0.9, "evidence_depth": 0.3,
				},
			},
		},
		Context:               "Code review scenario",
		Rubric:                "Classify systematic vs heuristic",
		GeneratorInstructions: "Create a code review question",
	}

	dispatcher := func(_ context.Context, nodeName string, _ string) (string, error) {
		switch nodeName {
		case "generate":
			return "What would you do about this leak?", nil
		case "subject":
			return "I would trace the goroutine lifecycle systematically", nil
		case "judge":
			return "SELECTED_POLE: systematic\nCONFIDENCE: 0.9", nil
		default:
			return "", fmt.Errorf("unexpected node: %s", nodeName)
		}
	}

	pipelineYAML, err := os.ReadFile("ouroboros/pipelines/ouroboros-probe.yaml")
	if err != nil {
		t.Fatalf("read pipeline: %v", err)
	}
	def, err := framework.LoadPipeline(pipelineYAML)
	if err != nil {
		t.Fatalf("load pipeline: %v", err)
	}

	nodes := ouroboros.PipelineNodes(seed, dispatcher)
	g, err := def.BuildGraph(nodes, nil)
	if err != nil {
		t.Fatalf("build graph: %v", err)
	}

	walker := framework.NewProcessWalker("e2e-ouroboros")
	if err := g.Walk(context.Background(), walker, def.Start); err != nil {
		t.Fatalf("walk: %v", err)
	}

	if walker.State().Status != "done" {
		t.Errorf("status = %q, want done", walker.State().Status)
	}

	history := walker.State().History
	expectedNodes := []string{"generate", "subject", "judge"}
	if len(history) != len(expectedNodes) {
		t.Fatalf("history length = %d, want %d", len(history), len(expectedNodes))
	}
	for i, want := range expectedNodes {
		if history[i].Node != want {
			t.Errorf("history[%d].Node = %q, want %q", i, history[i].Node, want)
		}
	}

	judgeArtifact := walker.State().Outputs["judge"]
	if judgeArtifact == nil {
		t.Fatal("judge output is nil")
	}
	if judgeArtifact.Type() != "pole-result" {
		t.Errorf("judge artifact type = %q, want pole-result", judgeArtifact.Type())
	}
}
