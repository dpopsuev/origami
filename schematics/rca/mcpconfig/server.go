package mcpconfig

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	framework "github.com/dpopsuev/origami"
	cal "github.com/dpopsuev/origami/calibrate"
	"github.com/dpopsuev/origami/connectors/rp"
	"github.com/dpopsuev/origami/dispatch"
	"github.com/dpopsuev/origami/kami"
	fwmcp "github.com/dpopsuev/origami/mcp"
	"github.com/dpopsuev/origami/schematics/rca"
	"github.com/dpopsuev/origami/schematics/rca/rcatype"
	"github.com/dpopsuev/origami/schematics/rca/rpconv"
	"github.com/dpopsuev/origami/schematics/rca/scenarios"
	"github.com/dpopsuev/origami/schematics/rca/store"
	"github.com/dpopsuev/origami/view"
)

var (
	DefaultGetNextStepTimeout = 10 * time.Second
	DefaultSessionTTL         = 5 * time.Minute
)

// Server wraps the generic CircuitServer with RCA-specific domain hooks.
type Server struct {
	*fwmcp.CircuitServer
	ProductName string
	ProjectRoot string

	KamiServer *kami.Server
	store      *view.CircuitStore
	bridge     *kami.EventBridge
}

// NewServer creates an RCA MCP server. The productName identifies the consumer
// (e.g. "asterisk"). Pass it from the consumer's manifest or CLI config.
func NewServer(productName string) *Server {
	cwd, _ := os.Getwd()
	s := &Server{ProductName: productName, ProjectRoot: cwd}
	s.CircuitServer = fwmcp.NewCircuitServer(s.buildConfig())
	return s
}

// Shutdown closes any active Kami bridge and store, then shuts down
// the underlying CircuitServer.
func (s *Server) Shutdown() {
	if s.bridge != nil {
		s.bridge.Close()
		s.bridge = nil
	}
	if s.store != nil {
		s.store.Close()
		s.store = nil
	}
	s.CircuitServer.Shutdown()
}

func (s *Server) buildConfig() fwmcp.CircuitConfig {
	cfg := fwmcp.CircuitConfig{
		Name:        s.ProductName,
		Version:     "dev",
		StepSchemas: rcaStepSchemas(),
		WorkerPreamble: fmt.Sprintf("You are a %s calibration worker.", s.ProductName),
		DefaultGetNextStepTimeout: int(DefaultGetNextStepTimeout / time.Millisecond),
		DefaultSessionTTL:         int(DefaultSessionTTL / time.Millisecond),
		CreateSession: func(ctx context.Context, params fwmcp.StartParams, disp *dispatch.MuxDispatcher, bus *dispatch.SignalBus) (fwmcp.RunFunc, fwmcp.SessionMeta, error) {
			return s.createSession(ctx, params, disp, bus)
		},
		FormatReport: func(result any) (string, any, error) {
			report, ok := result.(*rca.CalibrationReport)
			if !ok {
				return "", nil, fmt.Errorf("unexpected result type: %T", result)
			}
			formatted, err := rca.RenderCalibrationReport(report)
			if err != nil {
				return "", nil, fmt.Errorf("render calibration report: %w", err)
			}
			return formatted, report, nil
		},
	}

	cfg.OnStepDispatched = func(caseID, step string) {
		if s.KamiServer != nil && s.store != nil {
			s.store.OnEvent(framework.WalkEvent{
				Type:   framework.EventNodeEnter,
				Node:   stepToNode(step),
				Walker: caseID,
			})
		}
	}
	cfg.OnStepCompleted = func(caseID, step string, dispatchID int64) {
		if s.KamiServer != nil && s.store != nil {
			s.store.OnEvent(framework.WalkEvent{
				Type:   framework.EventNodeExit,
				Node:   stepToNode(step),
				Walker: caseID,
			})
		}
	}

	cfg.OnCircuitDone = func() {
		if s.KamiServer != nil && s.store != nil {
			s.store.OnEvent(framework.WalkEvent{
				Type: framework.EventWalkComplete,
			})
		}
	}

	cfg.OnSessionEnd = func() {
		if s.KamiServer != nil && s.store != nil {
			s.store.OnEvent(framework.WalkEvent{
				Type: framework.EventWalkComplete,
			})
		}
	}

	return cfg
}

func (s *Server) createSession(ctx context.Context, params fwmcp.StartParams, disp *dispatch.MuxDispatcher, bus *dispatch.SignalBus) (fwmcp.RunFunc, fwmcp.SessionMeta, error) {
	if s.KamiServer != nil {
		if s.bridge != nil {
			s.bridge.Close()
		}

		def, err := rca.AsteriskCircuitDef(rca.DefaultThresholds())
		if err == nil {
			st := view.NewCircuitStore(def)
			br := kami.NewEventBridge(bus)
			br.StartPolling(100 * time.Millisecond)
			s.KamiServer.SetStore(st)
			s.store = st
			s.bridge = br
		}
	}

	extra := params.Extra

	scenarioName, _ := extra["scenario"].(string)
	transformerName, _ := extra["backend"].(string)
	rpBaseURL, _ := extra["rp_base_url"].(string)
	rpProject, _ := extra["rp_project"].(string)

	scenario, err := loadScenario(scenarioName)
	if err != nil {
		return nil, fwmcp.SessionMeta{}, err
	}

	var rpFetcher rcatype.EnvelopeFetcher
	if rpBaseURL != "" {
		if rpProject == "" {
			rpProject = os.Getenv("ASTERISK_RP_PROJECT")
		}
		if rpProject == "" {
			return nil, fwmcp.SessionMeta{}, fmt.Errorf("rp_project is required when rp_base_url is set")
		}
		key, err := rp.ReadAPIKey(".rp-api-key")
		if err != nil {
			return nil, fwmcp.SessionMeta{}, fmt.Errorf("read RP API key: %w", err)
		}
		client, err := rp.New(rpBaseURL, key, rp.WithTimeout(30*time.Second))
		if err != nil {
			return nil, fwmcp.SessionMeta{}, fmt.Errorf("create RP client: %w", err)
		}
		rpFetcher = &rpconv.RPFetcherAdapter{Inner: rp.NewFetcher(client, rpProject)}
		if err := rca.ResolveRPCases(rpFetcher, scenario); err != nil {
			return nil, fwmcp.SessionMeta{}, fmt.Errorf("resolve RP-sourced cases: %w", err)
		}
	}

	root := s.ProjectRoot
	promptFS := rca.DefaultPromptFS
	basePath := filepath.Join(root, ".asterisk/calibrate")

	tokenTracker := dispatch.NewTokenTracker()
	tracked := dispatch.NewTokenTrackingDispatcher(disp, tokenTracker)

	var comps []*framework.Component
	var transformerLabel string
	var idMapper rca.IDMappable
	switch transformerName {
	case "stub":
		stub := rca.NewStubTransformer(scenario)
		comps = []*framework.Component{rca.TransformerComponent(stub)}
		transformerLabel = "stub"
		idMapper = stub
	case "basic":
		basicSt, err := store.Open(":memory:")
		if err != nil {
			return nil, fwmcp.SessionMeta{}, fmt.Errorf("basic transformer: open store: %w", err)
		}
		var repoNames []string
		for _, r := range scenario.Workspace.Repos {
			repoNames = append(repoNames, r.Name)
		}
		comps = []*framework.Component{rca.HeuristicComponent(basicSt, repoNames)}
		transformerLabel = "basic"
	default:
		t := rca.NewRCATransformer(
			tracked,
			promptFS,
			rca.WithRCABasePath(basePath),
		)
		comps = []*framework.Component{rca.TransformerComponent(t)}
		transformerLabel = "rca"
	}

	parallel := params.Parallel
	if parallel < 1 {
		parallel = 1
	}

	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fwmcp.SessionMeta{}, fmt.Errorf("create calibrate dir: %w", err)
	}

	scorecardPath := filepath.Join(root, "scorecards/asterisk-rca.yaml")
	sc, err := cal.LoadScoreCard(scorecardPath)
	if err != nil {
		return nil, fwmcp.SessionMeta{}, fmt.Errorf("load scorecard: %w", err)
	}

	cfg := rca.RunConfig{
		Scenario:     scenario,
		Components: comps,
		TransformerName: transformerLabel,
		IDMapper:     idMapper,
		Runs:         1,
		Thresholds:   rca.DefaultThresholds(),
		TokenTracker: tokenTracker,
		Parallel:     parallel,
		TokenBudget:  parallel,
		BasePath:     basePath,
		SourceFetcher: rpFetcher,
		ScoreCard:    sc,
	}

	runFn := func(ctx context.Context) (any, error) {
		return rca.RunCalibration(ctx, cfg)
	}

	meta := fwmcp.SessionMeta{
		TotalCases: len(scenario.Cases),
		Scenario:   scenario.Name,
	}

	return runFn, meta, nil
}

// rcaStepSchemas returns the F0-F6 step schemas for RCA calibration.
func rcaStepSchemas() []fwmcp.StepSchema {
	return []fwmcp.StepSchema{
		{
			Name:   "F0_RECALL",
			Fields: map[string]string{"match": "bool", "confidence": "float", "reasoning": "string"},
			Defs: []fwmcp.FieldDef{
				{Name: "match", Type: "bool", Required: true},
				{Name: "confidence", Type: "float", Required: true},
				{Name: "reasoning", Type: "string", Required: true},
			},
		},
		{
			Name: "F1_TRIAGE",
			Fields: map[string]string{
				"symptom_category": "string", "severity": "string",
				"defect_type_hypothesis": "string", "candidate_repos[]": "string[]",
				"skip_investigation": "bool", "cascade_suspected": "bool",
			},
			Defs: []fwmcp.FieldDef{
				{Name: "symptom_category", Type: "string", Required: true},
				{Name: "severity", Type: "string", Required: true},
				{Name: "defect_type_hypothesis", Type: "string", Required: true},
				{Name: "candidate_repos", Type: "array", Required: false},
				{Name: "skip_investigation", Type: "bool", Required: false},
				{Name: "cascade_suspected", Type: "bool", Required: false},
			},
		},
		{
			Name:   "F2_RESOLVE",
			Fields: map[string]string{"selected_repos[]": "{name, reason}"},
			Defs: []fwmcp.FieldDef{
				{Name: "selected_repos", Type: "array", Required: true},
			},
		},
		{
			Name: "F3_INVESTIGATE",
			Fields: map[string]string{
				"rca_message": "string", "defect_type": "string", "component": "string",
				"convergence_score": "float", "evidence_refs[]": "string[]",
			},
			Defs: []fwmcp.FieldDef{
				{Name: "rca_message", Type: "string", Required: true},
				{Name: "defect_type", Type: "string", Required: true},
				{Name: "component", Type: "string", Required: true},
				{Name: "convergence_score", Type: "float", Required: false},
				{Name: "evidence_refs", Type: "array", Required: false},
			},
		},
		{
			Name:   "F4_CORRELATE",
			Fields: map[string]string{"is_duplicate": "bool", "confidence": "float"},
			Defs: []fwmcp.FieldDef{
				{Name: "is_duplicate", Type: "bool", Required: true},
				{Name: "confidence", Type: "float", Required: true},
			},
		},
		{
			Name:   "F5_REVIEW",
			Fields: map[string]string{"decision": "approve|reassess|overturn"},
			Defs: []fwmcp.FieldDef{
				{Name: "decision", Type: "string", Required: true},
			},
		},
		{
			Name:   "F6_REPORT",
			Fields: map[string]string{"defect_type": "string", "case_id": "string", "summary": "string"},
			Defs: []fwmcp.FieldDef{
				{Name: "defect_type", Type: "string", Required: true},
				{Name: "case_id", Type: "string", Required: true},
				{Name: "summary", Type: "string", Required: true},
			},
		},
	}
}

func loadScenario(name string) (*rca.Scenario, error) {
	return scenarios.LoadScenario(name)
}

var stepPrefixRE = regexp.MustCompile(`^F\d+_`)

// stepToNode maps an MCP step name (e.g. "F0_RECALL") to a circuit node
// name (e.g. "recall") by stripping the F{n}_ prefix and lowercasing.
func stepToNode(step string) string {
	return strings.ToLower(stepPrefixRE.ReplaceAllString(step, ""))
}
