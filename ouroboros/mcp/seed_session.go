package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	framework "github.com/dpopsuev/origami"
	"github.com/dpopsuev/origami/dispatch"
	"github.com/dpopsuev/origami/logging"
	fwmcp "github.com/dpopsuev/origami/mcp"
	"github.com/dpopsuev/origami/ouroboros"
)

// NewSeedProfileConfig returns a CircuitConfig for seed-based model profiling.
// Instead of the discovery loop, this walks the ouroboros-probe circuit for
// each seed, using PoleResult scoring instead of keyword matching.
func NewSeedProfileConfig() fwmcp.CircuitConfig {
	return fwmcp.CircuitConfig{
		Name:    "ouroboros-seed",
		Version: "dev",
		StepSchemas: []fwmcp.StepSchema{
			{Name: "generate", Fields: map[string]string{"response": "raw LLM response"}},
			{Name: "subject", Fields: map[string]string{"response": "raw LLM response"}},
			{Name: "judge", Fields: map[string]string{"response": "raw LLM response"}},
		},
		WorkerPreamble: `You are an Ouroboros seed probe worker. For each step you receive a prompt.
Send it to the target model and return the raw response as {"response": "<text>"}.`,
		DefaultGetNextStepTimeout: 60000,
		DefaultSessionTTL:         600000,
		CreateSession: func(ctx context.Context, params fwmcp.StartParams, disp *dispatch.MuxDispatcher, bus *dispatch.SignalBus) (fwmcp.RunFunc, fwmcp.SessionMeta, error) {
			return createSeedSession(params, disp, bus)
		},
		FormatReport: formatSeedReport,
	}
}

func createSeedSession(
	params fwmcp.StartParams,
	disp *dispatch.MuxDispatcher,
	bus *dispatch.SignalBus,
) (fwmcp.RunFunc, fwmcp.SessionMeta, error) {
	seedPath, _ := params.Extra["seed_path"].(string)
	if seedPath == "" {
		return nil, fwmcp.SessionMeta{}, fmt.Errorf("seed_path is required in extra params")
	}

	seed, err := ouroboros.LoadSeed(seedPath)
	if err != nil {
		return nil, fwmcp.SessionMeta{}, fmt.Errorf("load seed: %w", err)
	}

	meta := fwmcp.SessionMeta{
		TotalCases: 1,
		Scenario:   fmt.Sprintf("seed-%s", seed.Name),
	}

	runFn := func(ctx context.Context) (any, error) {
		return runSeedCircuit(ctx, seed, disp, bus)
	}

	return runFn, meta, nil
}

type seedArtifact struct {
	Response string `json:"response"`
}

func runSeedCircuit(
	ctx context.Context,
	seed *ouroboros.Seed,
	disp *dispatch.MuxDispatcher,
	bus *dispatch.SignalBus,
) (*ouroboros.PoleResult, error) {
	log := logging.New("ouroboros-seed")

	dispatcher := func(ctx context.Context, nodeName string, prompt string) (string, error) {
		artifactBytes, err := disp.Dispatch(dispatch.DispatchContext{
			CaseID:        seed.Name,
			Step:          nodeName,
			PromptContent: prompt,
		})
		if err != nil {
			return "", err
		}

		var art seedArtifact
		if err := json.Unmarshal(artifactBytes, &art); err != nil {
			return "", fmt.Errorf("parse %s artifact: %w", nodeName, err)
		}
		return art.Response, nil
	}

	nodes := ouroboros.CircuitNodes(seed, dispatcher)

	circuitData, err := framework.ResolveCircuitPath("ouroboros/circuits/ouroboros-probe.yaml")
	if err != nil {
		return nil, fmt.Errorf("resolve circuit: %w", err)
	}

	def, err := framework.LoadCircuit(circuitData)
	if err != nil {
		return nil, fmt.Errorf("load circuit: %w", err)
	}

	g, err := def.BuildGraph(framework.GraphRegistries{Nodes: nodes})
	if err != nil {
		return nil, fmt.Errorf("build graph: %w", err)
	}

	walker := framework.NewProcessWalker(fmt.Sprintf("seed-%s", seed.Name))
	start := time.Now()

	if err := g.Walk(ctx, walker, def.Start); err != nil {
		return nil, fmt.Errorf("walk: %w", err)
	}

	elapsed := time.Since(start)
	log.Info("seed circuit completed", "seed", seed.Name, "elapsed", elapsed)

	judgeArtifact := walker.State().Outputs["judge"]
	if judgeArtifact == nil {
		return nil, fmt.Errorf("judge node produced no artifact")
	}

	result, ok := judgeArtifact.Raw().(*ouroboros.PoleResult)
	if !ok {
		return nil, fmt.Errorf("judge artifact type %T, expected *PoleResult", judgeArtifact.Raw())
	}

	bus.Emit("seed_completed", "server", seed.Name, "judge", map[string]string{
		"selected_pole": result.SelectedPole,
		"confidence":    fmt.Sprintf("%.2f", result.Confidence),
	})

	return result, nil
}

func formatSeedReport(result any) (string, any, error) {
	pr, ok := result.(*ouroboros.PoleResult)
	if !ok {
		return "", nil, fmt.Errorf("expected *PoleResult, got %T", result)
	}

	summary := fmt.Sprintf("Seed Probe Result\nSelected Pole: %s\nConfidence: %.2f\nReasoning: %s\n",
		pr.SelectedPole, pr.Confidence, pr.Reasoning)

	return summary, pr, nil
}
