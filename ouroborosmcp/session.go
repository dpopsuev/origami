package ouroborosmcp

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/dpopsuev/origami"
	fwmcp "github.com/dpopsuev/origami/mcp"
	"github.com/dpopsuev/origami/ouroboros"
)

// SessionState tracks the lifecycle of a discovery session.
type SessionState string

const (
	StateRunning SessionState = "running"
	StateDone    SessionState = "done"
	StateError   SessionState = "error"
)

// Session holds the state for a single discovery run driven by MCP tool calls.
type Session struct {
	ID           string
	Bus          *fwmcp.SignalBus
	Config       ouroboros.DiscoveryConfig
	ProbeHandler *ProbeHandler

	mu         sync.Mutex
	state      SessionState
	seen       map[string]ouroboros.DiscoveryResult
	seenOrder  []framework.ModelIdentity
	iteration  int
	report     *ouroboros.RunReport
	startTime  time.Time
	termReason string
}

// NewSession creates a discovery session with the given config and probe handler.
// If handler is nil, the session falls back to the legacy refactor-only behavior.
func NewSession(config ouroboros.DiscoveryConfig, handler *ProbeHandler) *Session {
	if config.MaxIterations == 0 {
		config = ouroboros.DefaultConfig()
	}
	s := &Session{
		ID:           fmt.Sprintf("mc-%d", time.Now().UnixNano()),
		Bus:          fwmcp.NewSignalBus(),
		Config:       config,
		ProbeHandler: handler,
		state:        StateRunning,
		seen:         make(map[string]ouroboros.DiscoveryResult),
		startTime:    time.Now(),
	}
	s.Bus.Emit("session_started", "server", "", "", map[string]string{
		"max_iterations": fmt.Sprintf("%d", config.MaxIterations),
		"probe_id":       config.ProbeID,
	})
	return s
}

// GetState returns the current session state.
func (s *Session) GetState() SessionState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state
}

// NextPrompt returns the discovery prompt for the current iteration.
// If the session is terminated, done=true and prompt is empty.
// Uses the probe handler's prompt if available; falls back to the default
// refactor probe for backward compatibility.
func (s *Session) NextPrompt() (prompt string, done bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state != StateRunning {
		return "", true
	}
	if s.iteration >= s.Config.MaxIterations {
		s.finalizeLocked("max_iterations_reached")
		return "", true
	}
	if s.ProbeHandler != nil {
		return ouroboros.BuildFullPromptWith(s.seenOrder, s.ProbeHandler.Prompt()), false
	}
	return ouroboros.BuildFullPrompt(s.seenOrder), false
}

// SubmitResponse parses a raw subagent response, scores the probe, checks for
// repeats, and advances the iteration. Returns the result and whether the model
// was a repeat (which terminates the session if TerminateOnRepeat is set).
//
// Scoring is probe-aware: if ProbeHandler is set and NeedsCodeBlock is false,
// the scorer receives the raw text (minus identity JSON); otherwise the legacy
// code-block extraction + refactor scorer is used.
func (s *Session) SubmitResponse(raw string) (ouroboros.DiscoveryResult, bool, error) {
	mi, err := ouroboros.ParseIdentityResponse(raw)
	if err != nil {
		return ouroboros.DiscoveryResult{}, false, fmt.Errorf("parse identity: %w", err)
	}

	if framework.IsWrapperName(mi.ModelName) {
		s.Bus.Emit("identity_rejected", "server", "", "", map[string]string{
			"model":  mi.ModelName,
			"reason": "wrapper",
		})
		return ouroboros.DiscoveryResult{}, false, fmt.Errorf(
			"wrapper identity rejected: model_name=%q is a known wrapper, not a foundation model", mi.ModelName)
	}

	var probeOutput string
	var score ouroboros.ProbeScore
	var dimScores map[ouroboros.Dimension]float64

	if s.ProbeHandler != nil && !s.ProbeHandler.NeedsCodeBlock {
		probeOutput = ouroboros.ExtractProbeText(raw)
		dimScores = s.ProbeHandler.Score(probeOutput)
	} else {
		code, parseErr := ouroboros.ParseProbeResponse(raw)
		if parseErr != nil {
			return ouroboros.DiscoveryResult{}, false, fmt.Errorf("parse probe: %w", parseErr)
		}
		probeOutput = code
		score = ouroboros.ScoreRefactorOutput(code)
		if s.ProbeHandler != nil {
			dimScores = s.ProbeHandler.Score(code)
		}
	}

	key := ouroboros.ModelKey(mi)

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state != StateRunning {
		return ouroboros.DiscoveryResult{}, false, fmt.Errorf("session is not running (state=%s)", s.state)
	}

	result := ouroboros.DiscoveryResult{
		Iteration:       s.iteration,
		Model:           mi,
		ExclusionPrompt: ouroboros.BuildExclusionPrompt(s.seenOrder),
		Probe: ouroboros.ProbeResult{
			ProbeID:         s.Config.ProbeID,
			RawOutput:       probeOutput,
			Score:           score,
			DimensionScores: dimScores,
		},
		Timestamp: time.Now(),
	}

	if _, exists := s.seen[key]; exists {
		s.Bus.Emit("model_repeated", "server", "", "", map[string]string{
			"model":     mi.ModelName,
			"iteration": fmt.Sprintf("%d", s.iteration),
		})
		if s.Config.TerminateOnRepeat {
			s.finalizeLocked(fmt.Sprintf("repeat_%s_at_iteration_%d", key, s.iteration))
		}
		return result, true, nil
	}

	s.seen[key] = result
	s.seenOrder = append(s.seenOrder, mi)
	s.iteration++

	s.Bus.Emit("model_discovered", "server", "", "", map[string]string{
		"model":     mi.ModelName,
		"provider":  mi.Provider,
		"iteration": fmt.Sprintf("%d", s.iteration-1),
		"score":     fmt.Sprintf("%.2f", score.TotalScore),
	})

	return result, false, nil
}

// Finalize terminates the session with the given reason and builds the report.
func (s *Session) Finalize(reason string) *ouroboros.RunReport {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.finalizeLocked(reason)
}

func (s *Session) finalizeLocked(reason string) *ouroboros.RunReport {
	if s.report != nil {
		return s.report
	}
	s.state = StateDone
	s.termReason = reason

	results := make([]ouroboros.DiscoveryResult, 0, len(s.seenOrder))
	for _, mi := range s.seenOrder {
		key := ouroboros.ModelKey(mi)
		if r, ok := s.seen[key]; ok {
			results = append(results, r)
		}
	}

	s.report = &ouroboros.RunReport{
		RunID:        s.ID,
		StartTime:    s.startTime,
		EndTime:      time.Now(),
		Config:       s.Config,
		Results:      results,
		UniqueModels: append([]framework.ModelIdentity{}, s.seenOrder...),
		TermReason:   reason,
	}

	s.Bus.Emit("session_done", "server", "", "", map[string]string{
		"unique_models": fmt.Sprintf("%d", len(s.seenOrder)),
		"term_reason":   reason,
	})

	return s.report
}

// GetReport returns the run report, or nil if the session hasn't been finalized.
func (s *Session) GetReport() *ouroboros.RunReport {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.report
}

// UniqueCount returns the number of unique models discovered so far.
func (s *Session) UniqueCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.seenOrder)
}

// ModelNames returns a comma-separated list of discovered model names.
func (s *Session) ModelNames() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	names := make([]string, len(s.seenOrder))
	for i, mi := range s.seenOrder {
		names[i] = mi.ModelName
	}
	return strings.Join(names, ", ")
}
