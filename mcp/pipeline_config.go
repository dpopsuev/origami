package mcp

import (
	"context"

	"github.com/dpopsuev/origami/dispatch"
)

// StepSchema declares what a single pipeline step expects in its artifact.
// Used to auto-generate worker prompt step-schema tables.
type StepSchema struct {
	Name   string            // e.g. "F0_RECALL", "scan"
	Fields map[string]string // field name -> type/description
}

// PipelineConfig is the domain-injection entry point. Implementations register
// three hooks (session creation, step schemas, report formatting) and the
// generic PipelineServer handles all protocol mechanics.
type PipelineConfig struct {
	Name    string // server implementation name (e.g. "asterisk", "achilles")
	Version string // server version (e.g. "dev")

	// StepSchemas declares the artifact schema for each pipeline step.
	// The worker prompt auto-generates a step-schema table from these.
	StepSchemas []StepSchema

	// WorkerPreamble is domain-specific instruction text prepended to the
	// auto-generated worker prompt. For example: "You are an Asterisk
	// calibration worker."
	WorkerPreamble string

	// CreateSession wires up a domain-specific pipeline run. It receives
	// the parsed start parameters, a pre-created MuxDispatcher, and the
	// session's SignalBus for domain-specific observability signals.
	// Returns a RunFunc (executed in a goroutine), initial metadata
	// (total_cases, scenario name), and an error.
	CreateSession func(ctx context.Context, params StartParams, disp *dispatch.MuxDispatcher, bus *dispatch.SignalBus) (RunFunc, SessionMeta, error)

	// FormatReport converts domain-specific run result into human-readable
	// text and optional structured data. Called by get_report.
	FormatReport func(result any) (formatted string, structured any, err error)

	// DefaultGetNextStepTimeout is the server-side timeout for get_next_step
	// when the caller doesn't specify timeout_ms. Defaults to 10s if zero.
	DefaultGetNextStepTimeout int // milliseconds

	// DefaultSessionTTL is the inactivity TTL for sessions. When no
	// submit_artifact arrives for this duration, the session aborts.
	// Defaults to 300s (5min) if zero.
	DefaultSessionTTL int // milliseconds
}

// RunFunc is the goroutine body that runs the domain pipeline. It receives
// a context (cancelled on session abort) and returns the domain result
// plus any error.
type RunFunc func(ctx context.Context) (result any, err error)

// SessionMeta carries initial metadata from the domain session factory
// back to the start_pipeline response.
type SessionMeta struct {
	TotalCases int
	Scenario   string
}

// StartParams are the parsed parameters from a start_pipeline tool call.
// Domain-specific fields live in Extra.
type StartParams struct {
	Parallel int
	Force    bool
	Extra    map[string]any // domain-specific params (scenario, adapter, rp_base_url, etc.)
}
