package calibrate

import (
	"context"
	"embed"
	"fmt"

	framework "github.com/dpopsuev/origami"
)

//go:embed pipelines/calibration-runner.yaml
var embeddedPipelines embed.FS

// PipelineOption configures a calibration pipeline run.
type PipelineOption func(*pipelineConfig)

type pipelineConfig struct {
	observer framework.WalkObserver
}

// WithObserver attaches a WalkObserver (e.g., Kami bridge) to the
// calibration pipeline for live visualization and debugging.
func WithObserver(obs framework.WalkObserver) PipelineOption {
	return func(c *pipelineConfig) { c.observer = obs }
}

// RunPipeline executes the calibration-runner pipeline with the given input.
// This is the primary API for running calibration as a DSL pipeline instead
// of procedural code.
//
// Usage:
//
//	report, err := calibrate.RunPipeline(ctx, &calibrate.CalibrationInput{
//	    Scenario:   "ptp-real",
//	    Adapter:    "cursor",
//	    Cases:      cases,
//	    GroundTruth: gt,
//	    ScoreCard:  sc,
//	    CaseRunner: myRunner,
//	    CaseScorer: myScorer,
//	}, calibrate.WithObserver(kamiBridge))
func RunPipeline(ctx context.Context, input *CalibrationInput, opts ...PipelineOption) (*CalibrationReport, error) {
	cfg := &pipelineConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	def, err := PipelineDef()
	if err != nil {
		return nil, err
	}

	edgeIDs := make([]string, len(def.Edges))
	for i, ed := range def.Edges {
		edgeIDs[i] = ed.ID
	}

	reg := framework.GraphRegistries{
		Nodes: CalibrationNodeRegistry(),
		Edges: forwardEdgeFactory(edgeIDs...),
	}

	graph, err := def.BuildGraph(reg)
	if err != nil {
		return nil, fmt.Errorf("build graph: %w", err)
	}

	if cfg.observer != nil {
		if dg, ok := graph.(*framework.DefaultGraph); ok {
			dg.SetObserver(cfg.observer)
		}
	}

	walker := framework.NewProcessWalker("calibration")
	walker.State().Context["input"] = input

	if err := graph.Walk(ctx, walker, def.Start); err != nil {
		return nil, fmt.Errorf("walk pipeline: %w", err)
	}

	reportArt, ok := walker.State().Outputs["report"]
	if !ok {
		return nil, fmt.Errorf("pipeline did not produce a report artifact")
	}

	report, ok := reportArt.Raw().(*CalibrationReport)
	if !ok {
		return nil, fmt.Errorf("report artifact type %T, want *CalibrationReport", reportArt.Raw())
	}

	return report, nil
}

// PipelineDef returns the parsed calibration pipeline definition.
// Useful for Kami registration or custom graph building.
func PipelineDef() (*framework.PipelineDef, error) {
	data, err := pipelineYAML()
	if err != nil {
		return nil, err
	}
	return framework.LoadPipeline(data)
}

// pipelineYAML returns the raw YAML for the calibration-runner pipeline.
// The YAML is loaded from disk (testdata-adjacent) so it can be edited
// without recompilation.
func pipelineYAML() ([]byte, error) {
	return embeddedPipelines.ReadFile("pipelines/calibration-runner.yaml")
}

func forwardEdgeFactory(ids ...string) framework.EdgeFactory {
	ef := make(framework.EdgeFactory, len(ids))
	for _, id := range ids {
		ef[id] = func(def framework.EdgeDef) framework.Edge {
			return &forwardEdge{def: def}
		}
	}
	return ef
}

type forwardEdge struct {
	def framework.EdgeDef
}

func (e *forwardEdge) ID() string         { return e.def.ID }
func (e *forwardEdge) From() string       { return e.def.From }
func (e *forwardEdge) To() string         { return e.def.To }
func (e *forwardEdge) IsShortcut() bool   { return e.def.Shortcut }
func (e *forwardEdge) IsLoop() bool       { return e.def.Loop }

func (e *forwardEdge) Evaluate(_ framework.Artifact, _ *framework.WalkerState) *framework.Transition {
	return &framework.Transition{NextNode: e.def.To}
}
