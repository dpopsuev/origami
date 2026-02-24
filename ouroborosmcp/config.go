package ouroborosmcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	framework "github.com/dpopsuev/origami"
	"github.com/dpopsuev/origami/dispatch"
	"github.com/dpopsuev/origami/logging"
	fwmcp "github.com/dpopsuev/origami/mcp"
	"github.com/dpopsuev/origami/ouroboros"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// NewOuroborosConfig returns a PipelineConfig that serves Ouroboros discovery
// via the generic PipelineServer. The discovery loop runs as a serial RunFunc
// (parallel=1): each iteration generates a probe prompt, dispatches it via
// MuxDispatcher, and processes the response for identity/scoring/termination.
func NewOuroborosConfig(runsDir string) fwmcp.PipelineConfig {
	registry := NewProbeRegistry()

	return fwmcp.PipelineConfig{
		Name:    "ouroboros",
		Version: "dev",
		StepSchemas: []fwmcp.StepSchema{{
			Name: "discover",
			Fields: map[string]string{
				"response": "raw LLM response text (identity JSON + probe output)",
			},
		}},
		WorkerPreamble: `You are an Ouroboros discovery worker probing AI models to discover their identity.
For each step you receive a probe prompt. Send it EXACTLY as-is to a subagent (Task tool).
Collect the subagent's raw response and wrap it in JSON: {"response": "<raw text>"}.
Submit via submit_artifact. Do NOT modify the probe prompt.`,
		DefaultGetNextStepTimeout: 30000,  // 30s — discovery needs LLM inference time
		DefaultSessionTTL:         600000, // 10min — discovery sessions can be slow
		CreateSession: func(ctx context.Context, params fwmcp.StartParams, disp *dispatch.MuxDispatcher, bus *dispatch.SignalBus) (fwmcp.RunFunc, fwmcp.SessionMeta, error) {
			return createDiscoverySession(params, disp, bus, registry, runsDir)
		},
		FormatReport: formatDiscoveryReport,
	}
}

// RegisterExtraTools adds the assemble_profiles tool to a PipelineServer.
// Call this after NewPipelineServer to register domain-specific tools
// alongside the 6 generic pipeline tools.
func RegisterExtraTools(srv *fwmcp.PipelineServer, runsDir string) {
	sdkmcp.AddTool(srv.MCPServer, &sdkmcp.Tool{
		Name:        "assemble_profiles",
		Description: "Assemble ModelProfiles from all persisted discovery runs. Groups results by model, aggregates dimension scores, computes element matching and persona suggestions.",
	}, func(_ context.Context, _ *sdkmcp.CallToolRequest, _ assembleProfilesInput) (*sdkmcp.CallToolResult, assembleProfilesOutput, error) {
		out, err := assembleProfilesFromStore(runsDir)
		if err != nil {
			return nil, assembleProfilesOutput{}, err
		}
		return nil, out, nil
	})
}

func createDiscoverySession(
	params fwmcp.StartParams,
	disp *dispatch.MuxDispatcher,
	bus *dispatch.SignalBus,
	registry *ProbeRegistry,
	runsDir string,
) (fwmcp.RunFunc, fwmcp.SessionMeta, error) {
	config := ouroboros.DefaultConfig()
	if v, ok := params.Extra["max_iterations"].(float64); ok && v > 0 {
		config.MaxIterations = int(v)
	}
	if v, ok := params.Extra["probe_id"].(string); ok && v != "" {
		config.ProbeID = v
	}
	if v, ok := params.Extra["terminate_on_repeat"].(bool); ok {
		config.TerminateOnRepeat = v
	}

	handler, err := registry.Get(config.ProbeID)
	if err != nil {
		return nil, fwmcp.SessionMeta{}, err
	}

	meta := fwmcp.SessionMeta{
		TotalCases: config.MaxIterations,
		Scenario:   fmt.Sprintf("discovery-%s", config.ProbeID),
	}

	runFn := func(ctx context.Context) (any, error) {
		return runDiscovery(ctx, config, handler, disp, bus, runsDir)
	}

	return runFn, meta, nil
}

type discoveryArtifact struct {
	Response string `json:"response"`
}

func runDiscovery(
	ctx context.Context,
	config ouroboros.DiscoveryConfig,
	handler *ProbeHandler,
	disp *dispatch.MuxDispatcher,
	bus *dispatch.SignalBus,
	runsDir string,
) (*ouroboros.RunReport, error) {
	log := logging.New("ouroboros-discovery")
	var seen []framework.ModelIdentity
	seenMap := make(map[string]ouroboros.DiscoveryResult)
	var results []ouroboros.DiscoveryResult
	startTime := time.Now()
	runID := fmt.Sprintf("mc-%d", time.Now().UnixNano())
	termReason := "max_iterations_reached"

	for i := 0; i < config.MaxIterations; i++ {
		if ctx.Err() != nil {
			termReason = "cancelled"
			break
		}

		var prompt string
		if handler != nil {
			prompt = ouroboros.BuildFullPromptWith(seen, handler.Prompt())
		} else {
			prompt = ouroboros.BuildFullPrompt(seen)
		}

		artifactBytes, err := disp.Dispatch(dispatch.DispatchContext{
			CaseID:        fmt.Sprintf("iter-%d", i),
			Step:          "discover",
			PromptContent: prompt,
		})
		if err != nil {
			log.Warn("dispatch error, ending discovery", "iteration", i, "error", err)
			termReason = fmt.Sprintf("dispatch_error_at_iteration_%d", i)
			break
		}

		var artifact discoveryArtifact
		if jsonErr := json.Unmarshal(artifactBytes, &artifact); jsonErr != nil || artifact.Response == "" {
			bus.Emit("artifact_parse_error", "server", fmt.Sprintf("iter-%d", i), "discover", map[string]string{
				"error": fmt.Sprintf("bad artifact at iteration %d", i),
			})
			continue
		}
		raw := artifact.Response

		mi, parseErr := ouroboros.ParseIdentityResponse(raw)
		if parseErr != nil {
			bus.Emit("identity_parse_error", "server", fmt.Sprintf("iter-%d", i), "discover", map[string]string{
				"error": parseErr.Error(),
			})
			continue
		}

		if framework.IsWrapperName(mi.ModelName) {
			bus.Emit("identity_rejected", "server", "", "", map[string]string{
				"model":  mi.ModelName,
				"reason": "wrapper",
			})
			continue
		}

		var probeOutput string
		var score ouroboros.ProbeScore
		var dimScores map[ouroboros.Dimension]float64

		if handler != nil && !handler.NeedsCodeBlock {
			probeOutput = ouroboros.ExtractProbeText(raw)
			dimScores = handler.Score(probeOutput)
		} else {
			code, codeErr := ouroboros.ParseProbeResponse(raw)
			if codeErr != nil {
				bus.Emit("probe_parse_error", "server", fmt.Sprintf("iter-%d", i), "discover", map[string]string{
					"error": codeErr.Error(),
				})
				continue
			}
			probeOutput = code
			score = ouroboros.ScoreRefactorOutput(code)
			if handler != nil {
				dimScores = handler.Score(code)
			}
		}

		key := ouroboros.ModelKey(mi)

		if _, exists := seenMap[key]; exists {
			bus.Emit("model_repeated", "server", "", "", map[string]string{
				"model":     mi.ModelName,
				"iteration": fmt.Sprintf("%d", i),
			})
			if config.TerminateOnRepeat {
				termReason = fmt.Sprintf("repeat_%s_at_iteration_%d", key, i)
				break
			}
			continue
		}

		result := ouroboros.DiscoveryResult{
			Iteration:       i,
			Model:           mi,
			ExclusionPrompt: ouroboros.BuildExclusionPrompt(seen),
			Probe: ouroboros.ProbeResult{
				ProbeID:         config.ProbeID,
				RawOutput:       probeOutput,
				Score:           score,
				DimensionScores: dimScores,
			},
			Timestamp: time.Now(),
		}
		seenMap[key] = result
		seen = append(seen, mi)
		results = append(results, result)

		bus.Emit("model_discovered", "server", "", "", map[string]string{
			"model":     mi.ModelName,
			"provider":  mi.Provider,
			"iteration": fmt.Sprintf("%d", i),
			"score":     fmt.Sprintf("%.2f", score.TotalScore),
		})
	}

	report := &ouroboros.RunReport{
		RunID:        runID,
		StartTime:    startTime,
		EndTime:      time.Now(),
		Config:       config,
		Results:      results,
		UniqueModels: append([]framework.ModelIdentity{}, seen...),
		TermReason:   termReason,
	}

	if runsDir != "" {
		if store, storeErr := ouroboros.NewFileRunStore(runsDir); storeErr == nil {
			if saveErr := store.SaveRun(*report); saveErr != nil {
				log.Warn("failed to persist run report", "error", saveErr)
			} else {
				log.Info("run report persisted", "run_id", report.RunID, "dir", runsDir)
			}
		}
	}

	return report, nil
}

func formatDiscoveryReport(result any) (string, any, error) {
	report, ok := result.(*ouroboros.RunReport)
	if !ok {
		return "", nil, fmt.Errorf("expected *ouroboros.RunReport, got %T", result)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Discovery Report: %s\n", report.RunID))
	sb.WriteString(fmt.Sprintf("Duration: %s\n", report.EndTime.Sub(report.StartTime).Round(time.Second)))
	sb.WriteString(fmt.Sprintf("Termination: %s\n", report.TermReason))
	sb.WriteString(fmt.Sprintf("Unique models: %d\n\n", len(report.UniqueModels)))

	for i, r := range report.Results {
		sb.WriteString(fmt.Sprintf("%d. %s (%s)\n", i+1, r.Model.ModelName, r.Model.Provider))
		if len(r.Probe.DimensionScores) > 0 {
			var dims []string
			for dim, score := range r.Probe.DimensionScores {
				dims = append(dims, fmt.Sprintf("%s=%.2f", dim, score))
			}
			sb.WriteString(fmt.Sprintf("   Dimensions: %s\n", strings.Join(dims, ", ")))
		}
	}

	return sb.String(), report, nil
}

// assembleProfilesFromStore reads all persisted runs, groups by model,
// aggregates dimension scores, and produces ModelProfiles.
func assembleProfilesFromStore(runsDir string) (assembleProfilesOutput, error) {
	if runsDir == "" {
		return assembleProfilesOutput{}, fmt.Errorf("assemble_profiles: runs_dir is not configured")
	}

	store, err := ouroboros.NewFileRunStore(runsDir)
	if err != nil {
		return assembleProfilesOutput{}, fmt.Errorf("assemble_profiles: %w", err)
	}

	runIDs, err := store.ListRuns()
	if err != nil {
		return assembleProfilesOutput{}, fmt.Errorf("assemble_profiles: list runs: %w", err)
	}

	type probeEntry struct {
		probeID string
		scores  map[ouroboros.Dimension]float64
	}

	modelData := make(map[string]*struct {
		identity framework.ModelIdentity
		probes   []probeEntry
	})

	for _, runID := range runIDs {
		report, loadErr := store.LoadRun(runID)
		if loadErr != nil {
			continue
		}
		for _, result := range report.Results {
			key := ouroboros.ModelKey(result.Model)
			if modelData[key] == nil {
				modelData[key] = &struct {
					identity framework.ModelIdentity
					probes   []probeEntry
				}{identity: result.Model}
			}
			if result.Probe.DimensionScores != nil {
				modelData[key].probes = append(modelData[key].probes, probeEntry{
					probeID: result.Probe.ProbeID,
					scores:  result.Probe.DimensionScores,
				})
			}
		}
	}

	var profiles []ouroboros.ModelProfile
	for _, data := range modelData {
		profile := ouroboros.ModelProfile{
			Model:          data.identity,
			BatteryVersion: ouroboros.BatteryVersion,
			Timestamp:      time.Now(),
			Dimensions:     make(map[ouroboros.Dimension]float64),
			ElementScores:  make(map[framework.Element]float64),
		}

		sums := make(map[ouroboros.Dimension]float64)
		counts := make(map[ouroboros.Dimension]int)
		for _, pe := range data.probes {
			for dim, score := range pe.scores {
				sums[dim] += score
				counts[dim]++
			}
			profile.RawResults = append(profile.RawResults, ouroboros.ProbeResult{
				ProbeID:         pe.probeID,
				DimensionScores: pe.scores,
			})
		}
		for _, dim := range ouroboros.AllDimensions() {
			if counts[dim] > 0 {
				profile.Dimensions[dim] = sums[dim] / float64(counts[dim])
			}
		}

		profile.ElementMatch = ouroboros.ElementMatch(profile)
		profile.ElementScores = ouroboros.ElementScores(profile)
		profile.SuggestedPersonas = ouroboros.SuggestPersona(profile)
		profiles = append(profiles, profile)
	}

	return assembleProfilesOutput{
		Profiles:   profiles,
		ModelCount: len(profiles),
		RunsUsed:   len(runIDs),
	}, nil
}

type assembleProfilesInput struct{}

type assembleProfilesOutput struct {
	Profiles   []ouroboros.ModelProfile `json:"profiles"`
	ModelCount int                     `json:"model_count"`
	RunsUsed   int                     `json:"runs_used"`
	Error      string                  `json:"error,omitempty"`
}
