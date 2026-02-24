package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/dpopsuev/origami/dispatch"
	"github.com/dpopsuev/origami/logging"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// PipelineServer is a domain-agnostic MCP server that manages pipeline
// sessions, capacity gating, worker prompt generation, and inline dispatch.
// Domain implementations create one by calling NewPipelineServer with a
// PipelineConfig that registers three hooks.
type PipelineServer struct {
	MCPServer *sdkmcp.Server
	Config    *PipelineConfig

	mu        sync.Mutex
	session   *PipelineSession
	sessCount int64

	defaultGetNextStepTimeout time.Duration
	defaultSessionTTL         time.Duration
}

// NewPipelineServer creates an MCP server with auto-registered pipeline tools.
// The config provides domain hooks (session factory, step schemas, report
// formatter) while the server handles all protocol mechanics.
func NewPipelineServer(cfg PipelineConfig) *PipelineServer {
	fw := NewServer(cfg.Name, cfg.Version)

	getNextTimeout := 10 * time.Second
	if cfg.DefaultGetNextStepTimeout > 0 {
		getNextTimeout = time.Duration(cfg.DefaultGetNextStepTimeout) * time.Millisecond
	}
	sessionTTL := 5 * time.Minute
	if cfg.DefaultSessionTTL > 0 {
		sessionTTL = time.Duration(cfg.DefaultSessionTTL) * time.Millisecond
	}

	s := &PipelineServer{
		MCPServer:                 fw.MCPServer,
		Config:                    &cfg,
		defaultGetNextStepTimeout: getNextTimeout,
		defaultSessionTTL:         sessionTTL,
	}
	s.registerTools()
	return s
}

// --- Tool input/output types ---

type startPipelineInput struct {
	Parallel int            `json:"parallel,omitempty" jsonschema:"number of parallel workers (default 1 = serial)"`
	Force    bool           `json:"force,omitempty" jsonschema:"cancel any existing session and start fresh"`
	Extra    map[string]any `json:"extra,omitempty" jsonschema:"domain-specific parameters"`
}

type startPipelineOutput struct {
	SessionID    string `json:"session_id"`
	TotalCases   int    `json:"total_cases"`
	Scenario     string `json:"scenario"`
	Status       string `json:"status"`
	WorkerPrompt string `json:"worker_prompt,omitempty"`
	WorkerCount  int    `json:"worker_count,omitempty"`
}

type getNextStepInput struct {
	SessionID       string `json:"session_id" jsonschema:"session ID from start_pipeline"`
	TimeoutMS       int    `json:"timeout_ms,omitempty" jsonschema:"max wait in milliseconds (0 = block forever)"`
	PreferredCaseID   string `json:"preferred_case_id,omitempty" jsonschema:"prefer steps for this case ID"`
	PreferredZone     string `json:"preferred_zone,omitempty" jsonschema:"prefer steps from this zone/provider"`
	Stickiness        int    `json:"stickiness,omitempty" jsonschema:"zone stickiness level: any(0) slight(1) strong(2) exclusive(3)"`
	ConsecutiveMisses int    `json:"consecutive_misses,omitempty" jsonschema:"caller-tracked empty polls for work stealing"`
}

type getNextStepOutput struct {
	Done             bool   `json:"done"`
	Available        bool   `json:"available,omitempty"`
	CaseID           string `json:"case_id,omitempty"`
	Step             string `json:"step,omitempty"`
	PromptPath       string `json:"prompt_path,omitempty"`
	PromptContent    string `json:"prompt_content,omitempty"`
	ArtifactPath     string `json:"artifact_path,omitempty"`
	DispatchID       int64  `json:"dispatch_id,omitempty"`
	ActiveDispatches int    `json:"active_dispatches"`
	DesiredCapacity  int    `json:"desired_capacity"`
	CapacityWarning  string `json:"capacity_warning,omitempty"`
	ShouldStop       bool   `json:"should_stop,omitempty"`
}

type submitStepInput struct {
	SessionID  string         `json:"session_id" jsonschema:"session ID from start_pipeline"`
	DispatchID int64          `json:"dispatch_id" jsonschema:"dispatch ID from get_next_step for artifact routing"`
	Step       string         `json:"step" jsonschema:"pipeline step name (e.g. F0_RECALL, F1_TRIAGE)"`
	Fields     map[string]any `json:"fields" jsonschema:"artifact fields matching the step schema"`
}

type submitStepOutput struct {
	OK string `json:"ok"`
}

type submitArtifactInput struct {
	SessionID    string `json:"session_id" jsonschema:"session ID from start_pipeline"`
	ArtifactJSON string `json:"artifact_json" jsonschema:"JSON artifact string for this pipeline step"`
	DispatchID   int64  `json:"dispatch_id,omitempty" jsonschema:"dispatch ID from get_next_step for artifact routing"`
}

type submitArtifactOutput struct {
	OK string `json:"ok"`
}

type getReportInput struct {
	SessionID string `json:"session_id" jsonschema:"session ID from start_pipeline"`
}

type getReportOutput struct {
	Status     string `json:"status"`
	Report     string `json:"report,omitempty"`
	Structured any    `json:"structured,omitempty"`
	Error      string `json:"error,omitempty"`
}

type emitSignalInput struct {
	SessionID string            `json:"session_id" jsonschema:"session ID from start_pipeline"`
	Event     string            `json:"event" jsonschema:"signal event (dispatch, start, done, error, loop)"`
	Agent     string            `json:"agent" jsonschema:"agent type (main, sub, server)"`
	CaseID    string            `json:"case_id,omitempty" jsonschema:"case ID if applicable"`
	Step      string            `json:"step,omitempty" jsonschema:"pipeline step if applicable"`
	Meta      map[string]string `json:"meta,omitempty" jsonschema:"optional key-value metadata"`
}

type emitSignalOutput struct {
	OK    string `json:"ok"`
	Index int    `json:"index"`
}

type getSignalsInput struct {
	SessionID string `json:"session_id" jsonschema:"session ID from start_pipeline"`
	Since     int    `json:"since,omitempty" jsonschema:"return signals from this index onward (0-based)"`
}

type getSignalsOutput struct {
	Signals []dispatch.Signal `json:"signals"`
	Total   int               `json:"total"`
}

type getWorkerHealthInput struct {
	SessionID string `json:"session_id" jsonschema:"session ID from start_pipeline"`
}

type getWorkerHealthOutput struct {
	dispatch.HealthSummary
}

// --- Tool registration ---

// noOutputSchema wraps a typed handler so the MCP SDK's Out type parameter is
// `any`, which suppresses outputSchema generation. Some MCP clients (including
// Cursor) don't support outputSchema and fail to parse the tools list when it's
// present.
func noOutputSchema[In, Out any](h func(context.Context, *sdkmcp.CallToolRequest, In) (*sdkmcp.CallToolResult, Out, error)) sdkmcp.ToolHandlerFor[In, any] {
	return func(ctx context.Context, req *sdkmcp.CallToolRequest, input In) (*sdkmcp.CallToolResult, any, error) {
		res, out, err := h(ctx, req, input)
		return res, out, err
	}
}

func (s *PipelineServer) registerTools() {
	sdkmcp.AddTool(s.MCPServer, &sdkmcp.Tool{
		Name:        "start_pipeline",
		Description: "Start a pipeline run. Spawns the runner goroutine and returns a session ID.",
	}, noOutputSchema(s.handleStartPipeline))

	sdkmcp.AddTool(s.MCPServer, &sdkmcp.Tool{
		Name:        "get_next_step",
		Description: "Get the next pipeline step prompt. Blocks until the runner is ready. Returns done=true when all cases are complete.",
	}, noOutputSchema(s.handleGetNextStep))

	sdkmcp.AddTool(s.MCPServer, &sdkmcp.Tool{
		Name:        "submit_step",
		Description: "Submit a schema-validated artifact for a pipeline step. The step name selects the schema; fields are validated before routing.",
	}, noOutputSchema(s.handleSubmitStep))

	sdkmcp.AddTool(s.MCPServer, &sdkmcp.Tool{
		Name:        "submit_artifact",
		Description: "Deprecated: use submit_step instead. Submit a JSON artifact for the current pipeline step. The runner scores it and advances.",
	}, noOutputSchema(s.handleSubmitArtifact))

	sdkmcp.AddTool(s.MCPServer, &sdkmcp.Tool{
		Name:        "get_report",
		Description: "Get the final pipeline report with metrics and per-case results.",
	}, noOutputSchema(s.handleGetReport))

	sdkmcp.AddTool(s.MCPServer, &sdkmcp.Tool{
		Name:        "emit_signal",
		Description: "Emit a signal to the agent message bus for observability. Use to announce dispatch, start, done, error events.",
	}, noOutputSchema(s.handleEmitSignal))

	sdkmcp.AddTool(s.MCPServer, &sdkmcp.Tool{
		Name:        "get_signals",
		Description: "Read signals from the agent message bus. Returns all signals, or signals since a given index.",
	}, noOutputSchema(s.handleGetSignals))

	sdkmcp.AddTool(s.MCPServer, &sdkmcp.Tool{
		Name:        "get_worker_health",
		Description: "Get worker health summary. Shows per-worker status, error counts, and replacement recommendations. The supervisor agent calls this to decide whether to replace or stop workers.",
	}, noOutputSchema(s.handleGetWorkerHealth))
}

// --- Tool handlers ---

func (s *PipelineServer) handleStartPipeline(ctx context.Context, _ *sdkmcp.CallToolRequest, input startPipelineInput) (*sdkmcp.CallToolResult, startPipelineOutput, error) {
	logger := logging.New("pipeline-session")
	s.mu.Lock()
	if s.session != nil {
		select {
		case <-s.session.Done():
			logger.Info("replacing completed/aborted session", "old_id", s.session.ID)
			s.session.Cancel()
		default:
			if input.Force {
				logger.Warn("force-replacing active session", "old_id", s.session.ID)
				s.session.Cancel()
			} else {
				s.mu.Unlock()
				return nil, startPipelineOutput{}, fmt.Errorf("a pipeline session is already running (id=%s)", s.session.ID)
			}
		}
	}
	s.session = nil
	s.mu.Unlock()

	parallel := input.Parallel
	if parallel < 1 {
		parallel = 1
	}

	params := StartParams{
		Parallel: parallel,
		Force:    input.Force,
		Extra:    input.Extra,
	}

	runCtx, runCancel := context.WithCancel(context.Background())
	bus := dispatch.NewSignalBus()
	disp := dispatch.NewMuxDispatcher(runCtx, dispatch.WithMuxSignalBus(bus))

	runFn, meta, err := s.Config.CreateSession(ctx, params, disp, bus)
	if err != nil {
		runCancel()
		return nil, startPipelineOutput{}, fmt.Errorf("create session: %w", err)
	}

	s.mu.Lock()
	s.sessCount++
	seqN := s.sessCount
	s.mu.Unlock()
	sessID := fmt.Sprintf("s-%d-%d", time.Now().UnixMilli(), seqN)
	sess := NewPipelineSession(runCtx, sessID, meta, parallel, disp, bus, runFn, runCancel)
	sess.SetTTL(s.defaultSessionTTL)

	bus.Emit("session_started", "server", "", "", map[string]string{
		"scenario":    meta.Scenario,
		"total_cases": fmt.Sprintf("%d", meta.TotalCases),
	})

	s.mu.Lock()
	s.session = sess
	s.mu.Unlock()

	out := startPipelineOutput{
		SessionID:  sess.ID,
		TotalCases: sess.TotalCases,
		Scenario:   sess.Scenario,
		Status:     string(StateRunning),
	}
	if sess.DesiredCapacity > 1 {
		out.WorkerPrompt = sess.WorkerPrompt(s.Config)
		out.WorkerCount = sess.DesiredCapacity
	}

	return nil, out, nil
}

func (s *PipelineServer) handleGetNextStep(ctx context.Context, _ *sdkmcp.CallToolRequest, input getNextStepInput) (*sdkmcp.CallToolResult, getNextStepOutput, error) {
	sess, err := s.getSession(input.SessionID)
	if err != nil {
		return nil, getNextStepOutput{}, err
	}

	var timeout time.Duration
	if input.TimeoutMS > 0 {
		timeout = time.Duration(input.TimeoutMS) * time.Millisecond
	} else {
		timeout = s.defaultGetNextStepTimeout
	}

	hints := dispatch.PullHints{
		PreferredCaseID:   input.PreferredCaseID,
		PreferredZone:     input.PreferredZone,
		Stickiness:        input.Stickiness,
		ConsecutiveMisses: input.ConsecutiveMisses,
	}

	sess.PullerEnter()
	dc, done, available, err := sess.GetNextStepWithHints(ctx, timeout, hints)
	sess.PullerExit()

	if err != nil {
		return nil, getNextStepOutput{}, fmt.Errorf("get_next_step: %w", err)
	}

	if done {
		sess.SetGateExempt()
		sess.Bus.Emit("pipeline_done", "server", "", "", nil)
		return nil, getNextStepOutput{Done: true}, nil
	}

	if !available {
		sess.SetGateExempt()
		return nil, getNextStepOutput{Done: false, Available: false}, nil
	}

	sess.Bus.Emit("step_ready", "server", dc.CaseID, dc.Step, map[string]string{
		"prompt_path": dc.PromptPath,
	})

	sess.Supervisor.Process()
	inFlight := sess.AgentPull()
	desired := sess.DesiredCapacity
	out := getNextStepOutput{
		Done:             false,
		Available:        true,
		CaseID:           dc.CaseID,
		Step:             dc.Step,
		PromptPath:       dc.PromptPath,
		ArtifactPath:     dc.ArtifactPath,
		DispatchID:       dc.DispatchID,
		ActiveDispatches: inFlight,
		DesiredCapacity:  desired,
		ShouldStop:       sess.Supervisor.ShouldStop(),
	}

	if dc.PromptContent != "" {
		out.PromptContent = dc.PromptContent
	} else if dc.PromptPath != "" {
		if content, err := os.ReadFile(dc.PromptPath); err == nil {
			out.PromptContent = string(content)
		}
	}

	if desired > 1 && inFlight < desired {
		out.CapacityWarning = fmt.Sprintf(
			"system under capacity: %d/%d workers active",
			inFlight, desired)
		logger := logging.New("pipeline-session")
		logger.Debug("under capacity",
			"in_flight", inFlight, "desired", desired, "deficit", desired-inFlight)
	}

	return nil, out, nil
}

func (s *PipelineServer) handleSubmitStep(ctx context.Context, _ *sdkmcp.CallToolRequest, input submitStepInput) (*sdkmcp.CallToolResult, submitStepOutput, error) {
	sess, err := s.getSession(input.SessionID)
	if err != nil {
		return nil, submitStepOutput{}, err
	}

	if gateErr := sess.CheckCapacityGate(); gateErr != nil {
		logger := logging.New("pipeline-session")
		logger.Warn("capacity gate advisory on submit_step",
			"session_id", input.SessionID, "dispatch_id", input.DispatchID, "detail", gateErr.Error())
	}

	if input.DispatchID == 0 {
		return nil, submitStepOutput{}, fmt.Errorf("dispatch_id is required (got 0); did you submit after available=false?")
	}

	if input.Step == "" {
		return nil, submitStepOutput{}, fmt.Errorf("step is required")
	}

	schema, err := s.Config.FindSchema(input.Step)
	if err != nil {
		return nil, submitStepOutput{}, err
	}

	if err := schema.ValidateFields(input.Fields); err != nil {
		return nil, submitStepOutput{}, err
	}

	data, err := json.Marshal(input.Fields)
	if err != nil {
		return nil, submitStepOutput{}, fmt.Errorf("marshal fields: %w", err)
	}

	if err := sess.SubmitArtifact(ctx, input.DispatchID, data); err != nil {
		return nil, submitStepOutput{}, fmt.Errorf("submit_step: %w", err)
	}

	remaining := sess.AgentSubmit()
	sess.Bus.Emit("artifact_submitted", "server", "", input.Step, map[string]string{
		"bytes":     fmt.Sprintf("%d", len(data)),
		"in_flight": fmt.Sprintf("%d", remaining),
		"via":       "submit_step",
	})

	return nil, submitStepOutput{OK: "step accepted"}, nil
}

// handleSubmitArtifact is the deprecated freeform JSON submission path.
// New workers should use submit_step instead.
func (s *PipelineServer) handleSubmitArtifact(ctx context.Context, _ *sdkmcp.CallToolRequest, input submitArtifactInput) (*sdkmcp.CallToolResult, submitArtifactOutput, error) {
	logger := logging.New("pipeline-session")
	logger.Warn("submit_artifact is deprecated; use submit_step for schema-validated submissions",
		"session_id", input.SessionID, "dispatch_id", input.DispatchID)

	sess, err := s.getSession(input.SessionID)
	if err != nil {
		return nil, submitArtifactOutput{}, err
	}

	if gateErr := sess.CheckCapacityGate(); gateErr != nil {
		logger.Warn("capacity gate advisory on submit",
			"session_id", input.SessionID, "dispatch_id", input.DispatchID, "detail", gateErr.Error())
	}

	if input.DispatchID == 0 {
		return nil, submitArtifactOutput{}, fmt.Errorf("dispatch_id is required (got 0); did you submit after available=false?")
	}

	data := []byte(input.ArtifactJSON)
	data = CleanArtifactJSON(data)
	if !json.Valid(data) {
		return nil, submitArtifactOutput{}, fmt.Errorf("artifact_json is not valid JSON")
	}

	if err := sess.SubmitArtifact(ctx, input.DispatchID, data); err != nil {
		return nil, submitArtifactOutput{}, fmt.Errorf("submit_artifact: %w", err)
	}

	remaining := sess.AgentSubmit()
	sess.Bus.Emit("artifact_submitted", "server", "", "", map[string]string{
		"bytes":     fmt.Sprintf("%d", len(data)),
		"in_flight": fmt.Sprintf("%d", remaining),
		"via":       "submit_artifact_deprecated",
	})

	return nil, submitArtifactOutput{OK: "artifact accepted"}, nil
}

func (s *PipelineServer) handleGetReport(ctx context.Context, _ *sdkmcp.CallToolRequest, input getReportInput) (*sdkmcp.CallToolResult, getReportOutput, error) {
	sess, err := s.getSession(input.SessionID)
	if err != nil {
		return nil, getReportOutput{}, err
	}

	select {
	case <-sess.Done():
	case <-ctx.Done():
		return nil, getReportOutput{}, ctx.Err()
	}

	if sessErr := sess.Err(); sessErr != nil {
		return nil, getReportOutput{
			Status: string(StateError),
			Error:  sessErr.Error(),
		}, nil
	}

	result := sess.Result()
	if result == nil {
		return nil, getReportOutput{Status: "no_report"}, nil
	}

	if s.Config.FormatReport == nil {
		return nil, getReportOutput{
			Status:     string(StateDone),
			Structured: result,
		}, nil
	}

	formatted, structured, err := s.Config.FormatReport(result)
	if err != nil {
		return nil, getReportOutput{
			Status: string(StateError),
			Error:  fmt.Sprintf("format report: %v", err),
		}, nil
	}

	return nil, getReportOutput{
		Status:     string(StateDone),
		Report:     formatted,
		Structured: structured,
	}, nil
}

func (s *PipelineServer) handleEmitSignal(ctx context.Context, _ *sdkmcp.CallToolRequest, input emitSignalInput) (*sdkmcp.CallToolResult, emitSignalOutput, error) {
	logger := logging.New("signal-bus")
	if input.Event == "" {
		logger.Warn("emit_signal rejected: empty event field")
		return nil, emitSignalOutput{}, fmt.Errorf("event is required")
	}
	if input.Agent == "" {
		logger.Warn("emit_signal rejected: empty agent field")
		return nil, emitSignalOutput{}, fmt.Errorf("agent is required")
	}

	sess, err := s.getSession(input.SessionID)
	if err != nil {
		return nil, emitSignalOutput{}, err
	}

	sess.Bus.Emit(input.Event, input.Agent, input.CaseID, input.Step, input.Meta)
	idx := sess.Bus.Len()

	if input.Event == "worker_started" {
		workerID := input.Meta["worker_id"]
		mode := input.Meta["mode"]
		if workerID != "" {
			sess.RegisterWorker(workerID, mode)
			logger.Debug("worker registered", "worker_id", workerID, "mode", mode)
		}
	}

	logger.Debug("signal emitted", "index", idx, "event", input.Event, "agent", input.Agent)

	return nil, emitSignalOutput{
		OK:    "signal emitted",
		Index: idx,
	}, nil
}

func (s *PipelineServer) handleGetSignals(ctx context.Context, _ *sdkmcp.CallToolRequest, input getSignalsInput) (*sdkmcp.CallToolResult, getSignalsOutput, error) {
	sess, err := s.getSession(input.SessionID)
	if err != nil {
		return nil, getSignalsOutput{}, err
	}

	signals := sess.Bus.Since(input.Since)
	return nil, getSignalsOutput{
		Signals: signals,
		Total:   sess.Bus.Len(),
	}, nil
}

func (s *PipelineServer) handleGetWorkerHealth(_ context.Context, _ *sdkmcp.CallToolRequest, input getWorkerHealthInput) (*sdkmcp.CallToolResult, getWorkerHealthOutput, error) {
	sess, err := s.getSession(input.SessionID)
	if err != nil {
		return nil, getWorkerHealthOutput{}, err
	}

	sess.Supervisor.Process()
	health := sess.Supervisor.Health()
	health.QueueDepth = sess.dispatcher.ActiveDispatches()

	return nil, getWorkerHealthOutput{HealthSummary: health}, nil
}

// --- Session management helpers ---

// SetSessionTTL configures the inactivity TTL on the current session.
func (s *PipelineServer) SetSessionTTL(ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.session != nil {
		s.session.SetTTL(ttl)
	}
}

// SessionID returns the current session's ID, or empty string if none.
func (s *PipelineServer) SessionID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.session != nil {
		return s.session.ID
	}
	return ""
}

// Shutdown cancels any active session.
func (s *PipelineServer) Shutdown() {
	s.mu.Lock()
	sess := s.session
	s.session = nil
	s.mu.Unlock()

	if sess != nil {
		sess.Cancel()
		<-sess.Done()
	}
}

// Session returns the current session for test introspection. Not for production use.
func (s *PipelineServer) Session() *PipelineSession {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.session
}

func (s *PipelineServer) getSession(id string) (*PipelineSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.session == nil {
		return nil, fmt.Errorf("no active session (call start_pipeline first)")
	}
	if s.session.ID != id {
		return nil, fmt.Errorf("session_id mismatch: have %s, got %s", s.session.ID, id)
	}
	return s.session, nil
}
