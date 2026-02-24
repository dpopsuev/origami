package mcp_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dpopsuev/origami/dispatch"
	"github.com/dpopsuev/origami/mcp"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestMain(m *testing.M) {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError,
	})))
	os.Exit(m.Run())
}

// testStepSchemas defines a simple 3-step pipeline for generic tests.
var testStepSchemas = []mcp.StepSchema{
	{
		Name:   "STEP_A",
		Fields: map[string]string{"value": "string", "score": "float"},
		Defs: []mcp.FieldDef{
			{Name: "value", Type: "string", Required: true},
			{Name: "score", Type: "float", Required: true},
		},
	},
	{
		Name:   "STEP_B",
		Fields: map[string]string{"result": "bool"},
		Defs: []mcp.FieldDef{
			{Name: "result", Type: "bool", Required: true},
		},
	},
	{
		Name:   "STEP_C",
		Fields: map[string]string{"summary": "string"},
		Defs: []mcp.FieldDef{
			{Name: "summary", Type: "string", Required: true},
		},
	},
}

// testArtifact returns minimal valid JSON for a given step.
func testArtifact(step string, workerID int) string {
	switch step {
	case "STEP_A":
		return fmt.Sprintf(`{"value":"worker-%d","score":0.9}`, workerID)
	case "STEP_B":
		return `{"result":true}`
	case "STEP_C":
		return fmt.Sprintf(`{"summary":"done by worker-%d"}`, workerID)
	default:
		return fmt.Sprintf(`{"step":"%s","worker":%d}`, step, workerID)
	}
}

// testReport is the domain result type for test pipelines.
type testReport struct {
	CasesProcessed int
	StepsProcessed int
}

// stubRunFunc creates a RunFunc that dispatches nCases in parallel (up to
// the MuxDispatcher's capacity), each with nSteps sequential steps. This
// mirrors how real pipeline runners operate: cases run concurrently but
// steps within a case are sequential.
func stubRunFunc(disp *dispatch.MuxDispatcher, nCases, nSteps, parallel int, steps []string, promptDir string) mcp.RunFunc {
	return func(ctx context.Context) (any, error) {
		sem := make(chan struct{}, parallel)
		var mu sync.Mutex
		total := 0
		errCh := make(chan error, nCases)

		var wg sync.WaitGroup
		for c := 0; c < nCases; c++ {
			wg.Add(1)
			go func(caseIdx int) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				caseID := fmt.Sprintf("C%02d", caseIdx+1)
				for s := 0; s < nSteps; s++ {
					step := steps[s%len(steps)]
					promptPath := ""
					if promptDir != "" {
						promptPath = fmt.Sprintf("%s/%s_%s.md", promptDir, caseID, step)
					}
					dc := dispatch.DispatchContext{
						CaseID:       caseID,
						Step:         step,
						PromptPath:   promptPath,
						ArtifactPath: fmt.Sprintf("/tmp/test_%s_%s.json", caseID, step),
					}
					if _, err := disp.Dispatch(dc); err != nil {
						errCh <- err
						return
					}
					mu.Lock()
					total++
					mu.Unlock()
				}
			}(c)
		}
		wg.Wait()
		close(errCh)
		if err := <-errCh; err != nil {
			return nil, err
		}

		mu.Lock()
		t := total
		mu.Unlock()
		return &testReport{CasesProcessed: nCases, StepsProcessed: t}, nil
	}
}

// stubRunFuncInstant creates a RunFunc that completes instantly (like a stub adapter).
func stubRunFuncInstant(nCases int) mcp.RunFunc {
	return func(ctx context.Context) (any, error) {
		return &testReport{CasesProcessed: nCases, StepsProcessed: 0}, nil
	}
}

// newTestConfig creates a PipelineConfig for testing.
func newTestConfig(nCases, nSteps int, promptDir string) mcp.PipelineConfig {
	steps := []string{"STEP_A", "STEP_B", "STEP_C"}
	return mcp.PipelineConfig{
		Name:        "test-pipeline",
		Version:     "dev",
		StepSchemas: testStepSchemas,
		WorkerPreamble: "You are a test pipeline worker.",
		DefaultGetNextStepTimeout: 1000,
		DefaultSessionTTL:         300000,
		CreateSession: func(ctx context.Context, params mcp.StartParams, disp *dispatch.MuxDispatcher, bus *dispatch.SignalBus) (mcp.RunFunc, mcp.SessionMeta, error) {
			parallel := params.Parallel
			if parallel < 1 {
				parallel = 1
			}
			return stubRunFunc(disp, nCases, nSteps, parallel, steps, promptDir),
				mcp.SessionMeta{TotalCases: nCases, Scenario: "test-scenario"},
				nil
		},
		FormatReport: func(result any) (string, any, error) {
			r, ok := result.(*testReport)
			if !ok {
				return "", nil, fmt.Errorf("unexpected result type")
			}
			return fmt.Sprintf("Processed %d cases, %d steps", r.CasesProcessed, r.StepsProcessed), r, nil
		},
	}
}

func newTestConfigStub(nCases int) mcp.PipelineConfig {
	return mcp.PipelineConfig{
		Name:        "test-pipeline",
		Version:     "dev",
		StepSchemas: testStepSchemas,
		DefaultGetNextStepTimeout: 1000,
		DefaultSessionTTL:         300000,
		CreateSession: func(ctx context.Context, params mcp.StartParams, disp *dispatch.MuxDispatcher, bus *dispatch.SignalBus) (mcp.RunFunc, mcp.SessionMeta, error) {
			return stubRunFuncInstant(nCases),
				mcp.SessionMeta{TotalCases: nCases, Scenario: "test-stub"},
				nil
		},
		FormatReport: func(result any) (string, any, error) {
			r, ok := result.(*testReport)
			if !ok {
				return "", nil, fmt.Errorf("unexpected result type")
			}
			return fmt.Sprintf("Stub: %d cases", r.CasesProcessed), r, nil
		},
	}
}

func newTestServer(t *testing.T, cfg mcp.PipelineConfig) *mcp.PipelineServer {
	t.Helper()
	srv := mcp.NewPipelineServer(cfg)
	t.Cleanup(srv.Shutdown)
	return srv
}

func connectInMemory(t *testing.T, ctx context.Context, srv *mcp.PipelineServer) *sdkmcp.ClientSession {
	t.Helper()
	t1, t2 := sdkmcp.NewInMemoryTransports()
	serverSession, err := srv.MCPServer.Connect(ctx, t1, nil)
	if err != nil {
		t.Fatalf("server.Connect: %v", err)
	}
	t.Cleanup(func() { serverSession.Close() })

	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "test-client", Version: "v0.0.1"}, nil)
	session, err := client.Connect(ctx, t2, nil)
	if err != nil {
		t.Fatalf("client.Connect: %v", err)
	}
	return session
}

func callTool(t *testing.T, ctx context.Context, session *sdkmcp.ClientSession, name string, args map[string]any) map[string]any {
	t.Helper()
	res, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
		Name:      name,
		Arguments: args,
	})
	if err != nil {
		t.Fatalf("CallTool(%s): %v", name, err)
	}
	if res.IsError {
		for _, c := range res.Content {
			if tc, ok := c.(*sdkmcp.TextContent); ok {
				t.Fatalf("CallTool(%s) returned error: %s", name, tc.Text)
			}
		}
		t.Fatalf("CallTool(%s) returned error", name)
	}
	result := make(map[string]any)
	for _, c := range res.Content {
		if tc, ok := c.(*sdkmcp.TextContent); ok {
			if err := json.Unmarshal([]byte(tc.Text), &result); err != nil {
				t.Fatalf("unmarshal tool result: %v (text: %s)", err, tc.Text)
			}
			return result
		}
	}
	t.Fatalf("no text content in tool result")
	return nil
}

func callToolE(ctx context.Context, session *sdkmcp.ClientSession, name string, args map[string]any) (map[string]any, error) {
	res, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
		Name:      name,
		Arguments: args,
	})
	if err != nil {
		return nil, fmt.Errorf("CallTool(%s): %w", name, err)
	}
	if res.IsError {
		for _, c := range res.Content {
			if tc, ok := c.(*sdkmcp.TextContent); ok {
				return nil, fmt.Errorf("CallTool(%s) error: %s", name, tc.Text)
			}
		}
		return nil, fmt.Errorf("CallTool(%s) returned error", name)
	}
	for _, c := range res.Content {
		if tc, ok := c.(*sdkmcp.TextContent); ok {
			result := make(map[string]any)
			if err := json.Unmarshal([]byte(tc.Text), &result); err != nil {
				return nil, fmt.Errorf("unmarshal %s result: %w", name, err)
			}
			return result, nil
		}
	}
	return nil, fmt.Errorf("no text content in %s result", name)
}

func containsCI(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

func containsAll(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if !containsCI(s, sub) {
			return false
		}
	}
	return true
}

// --- Tests ---

func TestPipelineServer_ToolDiscovery(t *testing.T) {
	srv := newTestServer(t, newTestConfigStub(3))
	ctx := context.Background()
	session := connectInMemory(t, ctx, srv)
	defer session.Close()

	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}

	want := map[string]bool{
		"start_pipeline":    false,
		"get_next_step":     false,
		"submit_step":       false,
		"submit_artifact":   false,
		"get_report":        false,
		"emit_signal":       false,
		"get_signals":       false,
		"get_worker_health": false,
	}
	for _, tool := range tools.Tools {
		if _, ok := want[tool.Name]; ok {
			want[tool.Name] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("tool %q not found in ListTools", name)
		}
	}
}

func TestPipelineServer_StubFullLoop(t *testing.T) {
	srv := newTestServer(t, newTestConfigStub(3))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	session := connectInMemory(t, ctx, srv)
	defer session.Close()

	startResult := callTool(t, ctx, session, "start_pipeline", map[string]any{})
	sessionID, ok := startResult["session_id"].(string)
	if !ok || sessionID == "" {
		t.Fatalf("expected non-empty session_id, got %v", startResult["session_id"])
	}
	totalCases, _ := startResult["total_cases"].(float64)
	if int(totalCases) != 3 {
		t.Fatalf("expected total_cases=3, got %v", totalCases)
	}

	time.Sleep(300 * time.Millisecond)
	stepResult := callTool(t, ctx, session, "get_next_step", map[string]any{
		"session_id": sessionID,
	})
	done, _ := stepResult["done"].(bool)
	if !done {
		t.Fatalf("expected done=true for stub RunFunc, got %v", stepResult)
	}

	reportResult := callTool(t, ctx, session, "get_report", map[string]any{
		"session_id": sessionID,
	})
	status, _ := reportResult["status"].(string)
	if status != "done" {
		t.Fatalf("expected status=done, got %s", status)
	}
	report, _ := reportResult["report"].(string)
	if report == "" {
		t.Fatal("expected non-empty report string")
	}
	t.Logf("report: %s", report)
}

func TestPipelineServer_GetNextStep_NoSession(t *testing.T) {
	srv := newTestServer(t, newTestConfigStub(1))
	ctx := context.Background()
	session := connectInMemory(t, ctx, srv)
	defer session.Close()

	res, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
		Name:      "get_next_step",
		Arguments: map[string]any{"session_id": "nonexistent"},
	})
	if err != nil {
		t.Fatalf("expected tool error, got transport error: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected IsError=true for missing session")
	}
}

func TestPipelineServer_DoubleStart_WhileRunning(t *testing.T) {
	cfg := newTestConfig(3, 3, "")
	srv := newTestServer(t, cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	session := connectInMemory(t, ctx, srv)
	defer session.Close()

	callTool(t, ctx, session, "start_pipeline", map[string]any{})

	res, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
		Name:      "start_pipeline",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("expected tool error, got transport error: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected IsError=true for double start while running")
	}
}

func TestPipelineServer_ForceStart_ReplacesRunning(t *testing.T) {
	cfg := newTestConfig(3, 3, "")
	srv := newTestServer(t, cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	session := connectInMemory(t, ctx, srv)
	defer session.Close()

	start1 := callTool(t, ctx, session, "start_pipeline", map[string]any{})
	sid1 := start1["session_id"].(string)

	start2 := callTool(t, ctx, session, "start_pipeline", map[string]any{"force": true})
	sid2 := start2["session_id"].(string)

	if sid2 == sid1 {
		t.Fatal("force-started session should have a different ID")
	}
	t.Logf("force-started session %s replaced %s", sid2, sid1)
}

// TestStartPipeline_WorkerPrompt verifies parallel>1 returns worker_prompt.
func TestStartPipeline_WorkerPrompt(t *testing.T) {
	cfg := newTestConfig(3, 3, "")
	srv := newTestServer(t, cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	session := connectInMemory(t, ctx, srv)
	defer session.Close()

	startResult := callTool(t, ctx, session, "start_pipeline", map[string]any{
		"parallel": 4,
	})
	sessionID := startResult["session_id"].(string)

	workerPrompt, _ := startResult["worker_prompt"].(string)
	if workerPrompt == "" {
		t.Fatal("expected non-empty worker_prompt for parallel>1")
	}
	if !containsAll(workerPrompt, sessionID, "get_next_step", "submit_step",
		"worker_started", "worker_stopped", "mode", "stream") {
		t.Errorf("worker_prompt missing required protocol keywords")
	}

	workerCount, _ := startResult["worker_count"].(float64)
	if int(workerCount) != 4 {
		t.Errorf("expected worker_count=4, got %v", workerCount)
	}

	if !containsCI(workerPrompt, "STEP_A") || !containsCI(workerPrompt, "STEP_B") || !containsCI(workerPrompt, "STEP_C") {
		t.Error("worker_prompt missing step schema names")
	}
	t.Logf("worker_prompt length: %d chars", len(workerPrompt))
}

// TestStartPipeline_WorkerPrompt_Serial verifies parallel=1 omits worker_prompt.
func TestStartPipeline_WorkerPrompt_Serial(t *testing.T) {
	cfg := newTestConfig(3, 3, "")
	srv := newTestServer(t, cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	session := connectInMemory(t, ctx, srv)
	defer session.Close()

	startResult := callTool(t, ctx, session, "start_pipeline", map[string]any{
		"parallel": 1,
	})

	workerPrompt, _ := startResult["worker_prompt"].(string)
	if workerPrompt != "" {
		t.Errorf("expected empty worker_prompt for parallel=1, got %d chars", len(workerPrompt))
	}
}

// TestCapacityWarning_ProtocolAgnostic verifies capacity warning text.
func TestCapacityWarning_ProtocolAgnostic(t *testing.T) {
	cfg := newTestConfig(3, 3, "")
	srv := newTestServer(t, cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	session := connectInMemory(t, ctx, srv)
	defer session.Close()

	startResult := callTool(t, ctx, session, "start_pipeline", map[string]any{
		"parallel": 4,
	})
	sessionID := startResult["session_id"].(string)

	step := callTool(t, ctx, session, "get_next_step", map[string]any{
		"session_id": sessionID,
	})
	warning, _ := step["capacity_warning"].(string)

	if warning == "" {
		t.Fatal("expected capacity_warning when only 1/4 workers active")
	}

	forbidden := []string{"launch", "subagent", "pull more", "MUST"}
	for _, word := range forbidden {
		if containsCI(warning, word) {
			t.Errorf("capacity_warning contains v1 language %q: %s", word, warning)
		}
	}

	required := []string{"under capacity", "workers active"}
	for _, word := range required {
		if !containsCI(warning, word) {
			t.Errorf("capacity_warning missing %q: %s", word, warning)
		}
	}
	t.Logf("capacity_warning: %s", warning)
}

// TestCapacityGate_ProtocolAgnostic verifies gate error message.
func TestCapacityGate_ProtocolAgnostic(t *testing.T) {
	sess := &mcp.PipelineSession{DesiredCapacity: 4}
	sess.AgentPull()
	gateErr := sess.CheckCapacityGate()
	if gateErr == nil {
		t.Fatal("expected gate error with 1/4 capacity")
	}
	msg := gateErr.Error()

	forbidden := []string{"CAPACITY GATE ADVISORY", "Pull", "bring more workers", "TTL watchdog"}
	for _, word := range forbidden {
		if containsCI(msg, word) {
			t.Errorf("gate message contains v1 language %q: %s", word, msg)
		}
	}

	required := []string{"capacity gate", "workers observed", "expects"}
	for _, word := range required {
		if !containsCI(msg, word) {
			t.Errorf("gate message missing %q: %s", word, msg)
		}
	}
	t.Logf("gate message: %s", msg)
}

// TestWorkerMode_StreamRegistration verifies worker_started signal tracking.
func TestWorkerMode_StreamRegistration(t *testing.T) {
	cfg := newTestConfig(3, 3, "")
	srv := newTestServer(t, cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	session := connectInMemory(t, ctx, srv)
	defer session.Close()

	startResult := callTool(t, ctx, session, "start_pipeline", map[string]any{
		"parallel": 4,
	})
	sessionID := startResult["session_id"].(string)

	for i := 0; i < 4; i++ {
		callTool(t, ctx, session, "emit_signal", map[string]any{
			"session_id": sessionID,
			"event":      "worker_started",
			"agent":      "worker",
			"meta":       map[string]any{"worker_id": fmt.Sprintf("w%d", i), "mode": "stream"},
		})
	}

	signals := callTool(t, ctx, session, "get_signals", map[string]any{
		"session_id": sessionID,
	})
	signalList, _ := signals["signals"].([]any)

	var workerStarted int
	for _, s := range signalList {
		sig, _ := s.(map[string]any)
		if sig["event"] == "worker_started" {
			workerStarted++
			meta, _ := sig["meta"].(map[string]any)
			if meta["mode"] != "stream" {
				t.Errorf("worker_started signal missing mode=stream: %v", meta)
			}
		}
	}
	if workerStarted != 4 {
		t.Errorf("expected 4 worker_started signals, got %d", workerStarted)
	}
}

// TestWorkerMode_NoWorkerID_Ignored verifies graceful handling.
func TestWorkerMode_NoWorkerID_Ignored(t *testing.T) {
	cfg := newTestConfig(3, 3, "")
	srv := newTestServer(t, cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	session := connectInMemory(t, ctx, srv)
	defer session.Close()

	startResult := callTool(t, ctx, session, "start_pipeline", map[string]any{
		"parallel": 2,
	})
	sessionID := startResult["session_id"].(string)

	callTool(t, ctx, session, "emit_signal", map[string]any{
		"session_id": sessionID,
		"event":      "worker_started",
		"agent":      "worker",
		"meta":       map[string]any{"mode": "stream"},
	})

	callTool(t, ctx, session, "emit_signal", map[string]any{
		"session_id": sessionID,
		"event":      "worker_started",
		"agent":      "worker",
	})

	t.Log("worker_started without worker_id accepted without panic")
}

// TestV2Workers_FullDrain_Deterministic is the definitive v2 choreography test.
func TestV2Workers_FullDrain_Deterministic(t *testing.T) {
	cfg := newTestConfig(4, 2, "")
	srv := newTestServer(t, cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	session := connectInMemory(t, ctx, srv)
	defer session.Close()

	startResult := callTool(t, ctx, session, "start_pipeline", map[string]any{
		"parallel": 4,
	})
	sessionID := startResult["session_id"].(string)

	if wp, _ := startResult["worker_prompt"].(string); wp == "" {
		t.Fatal("expected worker_prompt in start_pipeline response")
	}
	if wc, _ := startResult["worker_count"].(float64); int(wc) != 4 {
		t.Fatalf("expected worker_count=4, got %v", wc)
	}

	type stepRecord struct {
		CaseID     string
		Step       string
		DispatchID int64
	}

	var mu sync.Mutex
	workLog := make(map[int][]stepRecord)
	seenDispatchIDs := make(map[int64]bool)

	var wg sync.WaitGroup
	errCh := make(chan error, 4)

	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			_, err := callToolE(ctx, session, "emit_signal", map[string]any{
				"session_id": sessionID,
				"event":      "worker_started",
				"agent":      "worker",
				"meta":       map[string]any{"worker_id": fmt.Sprintf("w%d", workerID), "mode": "stream"},
			})
			if err != nil {
				errCh <- fmt.Errorf("w%d emit worker_started: %w", workerID, err)
				return
			}

			for {
				res, err := callToolE(ctx, session, "get_next_step", map[string]any{
					"session_id": sessionID,
					"timeout_ms": 300,
				})
				if err != nil {
					errCh <- fmt.Errorf("w%d get_next_step: %w", workerID, err)
					return
				}

				if done, _ := res["done"].(bool); done {
					break
				}
				if avail, _ := res["available"].(bool); !avail {
					continue
				}

				caseID, _ := res["case_id"].(string)
				step, _ := res["step"].(string)
				dispatchID, _ := res["dispatch_id"].(float64)

				artifact := testArtifact(step, workerID)
				_, err = callToolE(ctx, session, "submit_artifact", map[string]any{
					"session_id":    sessionID,
					"artifact_json": artifact,
					"dispatch_id":   int64(dispatchID),
				})
				if err != nil {
					errCh <- fmt.Errorf("w%d submit(%s/%s): %w", workerID, caseID, step, err)
					return
				}

				mu.Lock()
				workLog[workerID] = append(workLog[workerID], stepRecord{
					CaseID: caseID, Step: step, DispatchID: int64(dispatchID),
				})
				if seenDispatchIDs[int64(dispatchID)] {
					errCh <- fmt.Errorf("w%d: duplicate dispatch_id %d", workerID, int64(dispatchID))
				}
				seenDispatchIDs[int64(dispatchID)] = true
				mu.Unlock()
			}

			_, _ = callToolE(ctx, session, "emit_signal", map[string]any{
				"session_id": sessionID,
				"event":      "worker_stopped",
				"agent":      "worker",
				"meta":       map[string]any{"worker_id": fmt.Sprintf("w%d", workerID)},
			})
		}(i)
	}

	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Errorf("worker error: %v", err)
	}

	for i := 0; i < 4; i++ {
		if len(workLog[i]) == 0 {
			t.Errorf("worker-%d got zero steps (starvation)", i)
		} else {
			t.Logf("worker-%d processed %d steps", i, len(workLog[i]))
		}
	}

	var totalSteps int
	for _, records := range workLog {
		totalSteps += len(records)
	}
	if totalSteps == 0 {
		t.Fatal("pipeline produced zero steps")
	}
	t.Logf("total steps: %d across 4 workers", totalSteps)

	signals := callTool(t, ctx, session, "get_signals", map[string]any{
		"session_id": sessionID,
	})
	signalList, _ := signals["signals"].([]any)

	startedWorkers := make(map[string]bool)
	stoppedWorkers := make(map[string]bool)
	for _, s := range signalList {
		sig, _ := s.(map[string]any)
		event, _ := sig["event"].(string)
		meta, _ := sig["meta"].(map[string]any)
		wid, _ := meta["worker_id"].(string)
		switch event {
		case "worker_started":
			startedWorkers[wid] = true
		case "worker_stopped":
			stoppedWorkers[wid] = true
		}
	}
	if len(startedWorkers) != 4 {
		t.Errorf("expected 4 worker_started signals, got %d", len(startedWorkers))
	}
	if len(stoppedWorkers) != 4 {
		t.Errorf("expected 4 worker_stopped signals, got %d", len(stoppedWorkers))
	}

	reportResult := callTool(t, ctx, session, "get_report", map[string]any{
		"session_id": sessionID,
	})
	status, _ := reportResult["status"].(string)
	if status != "done" {
		t.Fatalf("expected status=done, got %s", status)
	}
}

// TestV2Workers_ConcurrencyTiming_Deterministic measures concurrent throughput.
func TestV2Workers_ConcurrencyTiming_Deterministic(t *testing.T) {
	cfg := newTestConfig(8, 3, "")
	srv := newTestServer(t, cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	session := connectInMemory(t, ctx, srv)
	defer session.Close()

	startResult := callTool(t, ctx, session, "start_pipeline", map[string]any{
		"parallel": 4,
	})
	sessionID := startResult["session_id"].(string)

	const perStepDelay = 20 * time.Millisecond
	var mu sync.Mutex
	var totalSteps int64

	var wg sync.WaitGroup
	errCh := make(chan error, 4)

	start := time.Now()
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			_, _ = callToolE(ctx, session, "emit_signal", map[string]any{
				"session_id": sessionID,
				"event":      "worker_started",
				"agent":      "worker",
				"meta":       map[string]any{"worker_id": fmt.Sprintf("w%d", workerID), "mode": "stream"},
			})

			for {
				res, err := callToolE(ctx, session, "get_next_step", map[string]any{
					"session_id": sessionID,
					"timeout_ms": 300,
				})
				if err != nil {
					errCh <- err
					return
				}
				if done, _ := res["done"].(bool); done {
					break
				}
				if avail, _ := res["available"].(bool); !avail {
					continue
				}

				time.Sleep(perStepDelay)

				step, _ := res["step"].(string)
				dispatchID, _ := res["dispatch_id"].(float64)

				artifact := testArtifact(step, workerID)
				_, err = callToolE(ctx, session, "submit_artifact", map[string]any{
					"session_id":    sessionID,
					"artifact_json": artifact,
					"dispatch_id":   int64(dispatchID),
				})
				if err != nil {
					errCh <- err
					return
				}
				mu.Lock()
				totalSteps++
				mu.Unlock()
			}

			_, _ = callToolE(ctx, session, "emit_signal", map[string]any{
				"session_id": sessionID,
				"event":      "worker_stopped",
				"agent":      "worker",
				"meta":       map[string]any{"worker_id": fmt.Sprintf("w%d", workerID)},
			})
		}(i)
	}

	wg.Wait()
	close(errCh)
	elapsed := time.Since(start)

	for err := range errCh {
		t.Fatalf("worker error: %v", err)
	}

	serialEstimate := time.Duration(totalSteps) * perStepDelay
	speedup := float64(serialEstimate) / float64(elapsed)

	t.Logf("timing: steps=%d, elapsed=%v, serial=%v, speedup=%.2fx",
		totalSteps, elapsed, serialEstimate, speedup)

	if elapsed > time.Duration(float64(serialEstimate)*0.75) {
		t.Errorf("concurrent execution too slow: elapsed=%v > 75%% of serial=%v (speedup=%.2fx)",
			elapsed, serialEstimate, speedup)
	}
}

// TestWorkerPrompt_StepSchemas verifies the generated prompt mentions all steps.
func TestWorkerPrompt_StepSchemas(t *testing.T) {
	cfg := newTestConfig(1, 1, "")
	sess := &mcp.PipelineSession{
		ID:              "test-session",
		DesiredCapacity: 4,
	}

	prompt := sess.WorkerPrompt(&cfg)

	for _, schema := range testStepSchemas {
		if !containsCI(prompt, schema.Name) {
			t.Errorf("worker prompt missing step %s", schema.Name)
		}
	}

	if !containsCI(prompt, "test-session") {
		t.Error("worker prompt missing session_id")
	}

	keywords := []string{"get_next_step", "submit_step", "worker_started", "worker_stopped", "mode", "stream"}
	for _, kw := range keywords {
		if !containsCI(prompt, kw) {
			t.Errorf("worker prompt missing keyword %q", kw)
		}
	}
}

// TestWorkerPrompt_SessionIDEmbedded verifies session ID is concrete, not a placeholder.
func TestWorkerPrompt_SessionIDEmbedded(t *testing.T) {
	cfg := newTestConfig(1, 1, "")
	sess := &mcp.PipelineSession{
		ID:              "s-1234567890",
		DesiredCapacity: 2,
	}

	prompt := sess.WorkerPrompt(&cfg)

	if !containsCI(prompt, "s-1234567890") {
		t.Error("worker prompt does not contain the actual session ID")
	}

	if containsCI(prompt, "%s") || containsCI(prompt, "{session_id}") || containsCI(prompt, "%[1]s") {
		t.Error("worker prompt contains unresolved template placeholders")
	}
}

// TestSignalBus_EmitAndGet tests basic signal bus flow.
func TestSignalBus_EmitAndGet(t *testing.T) {
	srv := newTestServer(t, newTestConfigStub(3))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	session := connectInMemory(t, ctx, srv)
	defer session.Close()

	startResult := callTool(t, ctx, session, "start_pipeline", map[string]any{})
	sessionID := startResult["session_id"].(string)
	time.Sleep(300 * time.Millisecond)

	emitResult := callTool(t, ctx, session, "emit_signal", map[string]any{
		"session_id": sessionID,
		"event":      "dispatch",
		"agent":      "main",
		"case_id":    "C01",
		"step":       "STEP_A",
		"meta":       map[string]any{"detail": "test"},
	})
	if emitResult["ok"] != "signal emitted" {
		t.Fatalf("expected ok='signal emitted', got %v", emitResult)
	}

	getResult := callTool(t, ctx, session, "get_signals", map[string]any{
		"session_id": sessionID,
	})
	total, _ := getResult["total"].(float64)
	if total < 2 {
		t.Fatalf("expected at least 2 signals, got %v", total)
	}

	signals, ok := getResult["signals"].([]any)
	if !ok || len(signals) == 0 {
		t.Fatal("expected signals array")
	}

	found := false
	for _, s := range signals {
		sig, ok := s.(map[string]any)
		if !ok {
			continue
		}
		if sig["event"] == "dispatch" && sig["agent"] == "main" && sig["case_id"] == "C01" {
			found = true
			break
		}
	}
	if !found {
		t.Error("agent-emitted dispatch signal not found in bus")
	}
}

// TestSignalBus_EmitRejectsEmpty verifies validation.
func TestSignalBus_EmitRejectsEmpty(t *testing.T) {
	srv := newTestServer(t, newTestConfigStub(1))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	session := connectInMemory(t, ctx, srv)
	defer session.Close()

	startResult := callTool(t, ctx, session, "start_pipeline", map[string]any{})
	sessionID := startResult["session_id"].(string)
	time.Sleep(300 * time.Millisecond)

	res, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
		Name: "emit_signal",
		Arguments: map[string]any{
			"session_id": sessionID, "event": "", "agent": "main",
		},
	})
	if err != nil {
		t.Fatalf("transport error: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected IsError=true for empty event")
	}

	res, err = session.CallTool(ctx, &sdkmcp.CallToolParams{
		Name: "emit_signal",
		Arguments: map[string]any{
			"session_id": sessionID, "event": "test", "agent": "",
		},
	})
	if err != nil {
		t.Fatalf("transport error: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected IsError=true for empty agent")
	}
}

// TestSession_TTL_Abort verifies the TTL watchdog aborts stale sessions.
func TestSession_TTL_Abort(t *testing.T) {
	cfg := newTestConfig(3, 3, "")
	srv := newTestServer(t, cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	session := connectInMemory(t, ctx, srv)
	defer session.Close()

	startResult := callTool(t, ctx, session, "start_pipeline", map[string]any{
		"parallel": 1,
	})
	sessionID := startResult["session_id"].(string)

	step := callTool(t, ctx, session, "get_next_step", map[string]any{
		"session_id": sessionID,
	})
	if done, _ := step["done"].(bool); done {
		t.Fatal("expected a step, got done=true")
	}

	srv.SetSessionTTL(200 * time.Millisecond)
	time.Sleep(500 * time.Millisecond)

	res := callTool(t, ctx, session, "get_next_step", map[string]any{
		"session_id": sessionID,
	})
	done, _ := res["done"].(bool)
	if !done {
		t.Fatalf("expected done=true after TTL abort, got %v", res)
	}
	t.Log("TTL abort verified")
}

// TestCleanArtifactJSON verifies markdown fence stripping.
func TestCleanArtifactJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain JSON", `{"key":"value"}`, `{"key":"value"}`},
		{"json fenced", "```json\n{\"key\":\"value\"}\n```", `{"key":"value"}`},
		{"generic fenced", "```\n{\"key\":\"value\"}\n```", `{"key":"value"}`},
		{"whitespace", "  \n{\"key\":\"value\"}\n  ", `{"key":"value"}`},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(mcp.CleanArtifactJSON([]byte(tt.input)))
			if got != tt.want {
				t.Errorf("CleanArtifactJSON(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestPipelineServer_GetWorkerHealth(t *testing.T) {
	srv := newTestServer(t, newTestConfig(2, 1, ""))
	ctx := context.Background()
	session := connectInMemory(t, ctx, srv)
	defer session.Close()

	startResult := callTool(t, ctx, session, "start_pipeline", map[string]any{
		"parallel": 1,
	})
	sessionID := startResult["session_id"].(string)

	// Emit worker signals manually
	callTool(t, ctx, session, "emit_signal", map[string]any{
		"session_id": sessionID,
		"event":      "worker_started",
		"agent":      "worker",
		"meta": map[string]any{
			"worker_id": "test-w1",
		},
	})
	callTool(t, ctx, session, "emit_signal", map[string]any{
		"session_id": sessionID,
		"event":      "error",
		"agent":      "worker",
		"case_id":    "C01",
		"step":       "STEP_A",
		"meta": map[string]any{
			"worker_id": "test-w1",
			"error":     "test error",
		},
	})

	health := callTool(t, ctx, session, "get_worker_health", map[string]any{
		"session_id": sessionID,
	})

	workers, ok := health["workers"]
	if !ok {
		t.Fatal("health response missing 'workers' field")
	}
	workerList, ok := workers.([]any)
	if !ok || len(workerList) == 0 {
		t.Fatal("expected at least one worker in health summary")
	}

	w := workerList[0].(map[string]any)
	if w["worker_id"] != "test-w1" {
		t.Errorf("expected worker_id=test-w1, got %v", w["worker_id"])
	}
	if w["error_count"].(float64) != 1 {
		t.Errorf("expected error_count=1, got %v", w["error_count"])
	}
	if w["last_error"] != "test error" {
		t.Errorf("expected last_error='test error', got %v", w["last_error"])
	}

	srv.Shutdown()
}

// --- submit_step tests ---

func TestStepSchema_ValidateFields(t *testing.T) {
	schema := mcp.StepSchema{
		Name: "TEST_STEP",
		Defs: []mcp.FieldDef{
			{Name: "name", Type: "string", Required: true},
			{Name: "score", Type: "float", Required: true},
			{Name: "notes", Type: "string", Required: false},
		},
	}

	t.Run("valid with all fields", func(t *testing.T) {
		err := schema.ValidateFields(map[string]any{
			"name": "foo", "score": 0.9, "notes": "ok",
		})
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})

	t.Run("valid with optional missing", func(t *testing.T) {
		err := schema.ValidateFields(map[string]any{
			"name": "foo", "score": 0.9,
		})
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})

	t.Run("missing required field", func(t *testing.T) {
		err := schema.ValidateFields(map[string]any{"name": "foo"})
		if err == nil {
			t.Fatal("expected error for missing required field")
		}
		if !strings.Contains(err.Error(), "score") {
			t.Errorf("error should mention 'score': %v", err)
		}
	})

	t.Run("null required field", func(t *testing.T) {
		err := schema.ValidateFields(map[string]any{
			"name": "foo", "score": nil,
		})
		if err == nil {
			t.Fatal("expected error for null required field")
		}
	})

	t.Run("no defs passes anything", func(t *testing.T) {
		legacy := mcp.StepSchema{Name: "LEGACY", Fields: map[string]string{"x": "any"}}
		err := legacy.ValidateFields(map[string]any{"whatever": 42})
		if err != nil {
			t.Errorf("legacy schema with no defs should pass: %v", err)
		}
	})
}

func TestPipelineConfig_FindSchema(t *testing.T) {
	cfg := mcp.PipelineConfig{
		StepSchemas: testStepSchemas,
	}

	t.Run("found", func(t *testing.T) {
		s, err := cfg.FindSchema("STEP_B")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.Name != "STEP_B" {
			t.Errorf("expected STEP_B, got %s", s.Name)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := cfg.FindSchema("NO_SUCH_STEP")
		if err == nil {
			t.Fatal("expected error for unknown step")
		}
		if !strings.Contains(err.Error(), "STEP_A") {
			t.Errorf("error should list valid steps: %v", err)
		}
	})
}

func TestSubmitStep_FullLoop(t *testing.T) {
	srv := newTestServer(t, newTestConfig(1, 1, ""))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	session := connectInMemory(t, ctx, srv)
	defer session.Close()

	startResult := callTool(t, ctx, session, "start_pipeline", map[string]any{
		"parallel": 1,
	})
	sessionID := startResult["session_id"].(string)

	res := callTool(t, ctx, session, "get_next_step", map[string]any{
		"session_id": sessionID,
		"timeout_ms": 2000,
	})

	if done, _ := res["done"].(bool); done {
		t.Fatal("expected a step, got done=true")
	}

	step, _ := res["step"].(string)
	dispatchID, _ := res["dispatch_id"].(float64)

	result := callTool(t, ctx, session, "submit_step", map[string]any{
		"session_id":  sessionID,
		"dispatch_id": int64(dispatchID),
		"step":        step,
		"fields":      testFieldsForStep(step),
	})

	if ok, _ := result["ok"].(string); ok != "step accepted" {
		t.Errorf("expected 'step accepted', got %q", ok)
	}
}

func TestSubmitStep_UnknownStep(t *testing.T) {
	srv := newTestServer(t, newTestConfig(1, 1, ""))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	session := connectInMemory(t, ctx, srv)
	defer session.Close()

	startResult := callTool(t, ctx, session, "start_pipeline", map[string]any{
		"parallel": 1,
	})
	sessionID := startResult["session_id"].(string)

	res := callTool(t, ctx, session, "get_next_step", map[string]any{
		"session_id": sessionID,
		"timeout_ms": 2000,
	})
	dispatchID, _ := res["dispatch_id"].(float64)

	_, err := callToolE(ctx, session, "submit_step", map[string]any{
		"session_id":  sessionID,
		"dispatch_id": int64(dispatchID),
		"step":        "NONEXISTENT",
		"fields":      map[string]any{"x": 1},
	})
	if err == nil {
		t.Fatal("expected error for unknown step")
	}
	if !strings.Contains(err.Error(), "unknown step") {
		t.Errorf("error should mention 'unknown step': %v", err)
	}
}

func TestSubmitStep_MissingRequiredField(t *testing.T) {
	srv := newTestServer(t, newTestConfig(1, 1, ""))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	session := connectInMemory(t, ctx, srv)
	defer session.Close()

	startResult := callTool(t, ctx, session, "start_pipeline", map[string]any{
		"parallel": 1,
	})
	sessionID := startResult["session_id"].(string)

	res := callTool(t, ctx, session, "get_next_step", map[string]any{
		"session_id": sessionID,
		"timeout_ms": 2000,
	})
	dispatchID, _ := res["dispatch_id"].(float64)
	step, _ := res["step"].(string)

	_, err := callToolE(ctx, session, "submit_step", map[string]any{
		"session_id":  sessionID,
		"dispatch_id": int64(dispatchID),
		"step":        step,
		"fields":      map[string]any{},
	})
	if err == nil {
		t.Fatal("expected error for missing required field")
	}
	if !strings.Contains(err.Error(), "missing required field") {
		t.Errorf("error should mention 'missing required field': %v", err)
	}
}

func TestSubmitStep_ZeroDispatchID(t *testing.T) {
	srv := newTestServer(t, newTestConfig(1, 1, ""))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	session := connectInMemory(t, ctx, srv)
	defer session.Close()

	startResult := callTool(t, ctx, session, "start_pipeline", map[string]any{
		"parallel": 1,
	})
	sessionID := startResult["session_id"].(string)

	_, err := callToolE(ctx, session, "submit_step", map[string]any{
		"session_id":  sessionID,
		"dispatch_id": 0,
		"step":        "STEP_A",
		"fields":      map[string]any{"value": "x", "score": 1.0},
	})
	if err == nil {
		t.Fatal("expected error for dispatch_id=0")
	}
}

func testFieldsForStep(step string) map[string]any {
	switch step {
	case "STEP_A":
		return map[string]any{"value": "test", "score": 0.95}
	case "STEP_B":
		return map[string]any{"result": true}
	case "STEP_C":
		return map[string]any{"summary": "done"}
	default:
		return map[string]any{"data": step}
	}
}
