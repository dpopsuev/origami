package mcp

import (
	"context"
	"fmt"

	"github.com/dpopsuev/origami/dispatch"
)

// FieldDef describes a single field in a step's artifact schema.
type FieldDef struct {
	Name     string // field name, e.g. "confidence"
	Type     string // type hint: "string", "bool", "float", "object", "array"
	Required bool   // if true, submit_step rejects artifacts missing this field
	Desc     string // optional human-readable description
}

// StepSchema declares what a single circuit step expects in its artifact.
// Used for runtime validation in submit_step and to auto-generate worker
// prompt step-schema tables.
type StepSchema struct {
	Name   string            // e.g. "F0_RECALL", "scan"
	Fields map[string]string // field name -> type/description (legacy, used for prompt rendering)
	Defs   []FieldDef        // structured field definitions for runtime validation
}

// ValidateFields checks that fields satisfies the schema's Defs.
// Returns nil if Defs is empty (legacy schemas pass without validation).
func (s StepSchema) ValidateFields(fields map[string]any) error {
	if len(s.Defs) == 0 {
		return nil
	}
	for _, def := range s.Defs {
		v, ok := fields[def.Name]
		if !ok && def.Required {
			return fmt.Errorf("step %s: missing required field %q", s.Name, def.Name)
		}
		if ok && v == nil && def.Required {
			return fmt.Errorf("step %s: field %q is null", s.Name, def.Name)
		}
	}
	return nil
}

// CircuitConfig is the domain-injection entry point. Implementations register
// three hooks (session creation, step schemas, report formatting) and the
// generic CircuitServer handles all protocol mechanics.
type CircuitConfig struct {
	Name    string // server implementation name (e.g. "asterisk", "achilles")
	Version string // server version (e.g. "dev")

	// StepSchemas declares the artifact schema for each circuit step.
	// The worker prompt auto-generates a step-schema table from these.
	StepSchemas []StepSchema

	// WorkerPreamble is domain-specific instruction text prepended to the
	// auto-generated worker prompt. For example: "You are an Asterisk
	// calibration worker."
	WorkerPreamble string

	// CreateSession wires up a domain-specific circuit run. It receives
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
	// artifact submission arrives for this duration, the session aborts.
	// Defaults to 300s (5min) if zero.
	DefaultSessionTTL int // milliseconds
}

// FindSchema returns the StepSchema for the given step name, or an error
// listing valid step names. Used by the submit_step handler.
func (c *CircuitConfig) FindSchema(step string) (StepSchema, error) {
	var names []string
	for _, s := range c.StepSchemas {
		if s.Name == step {
			return s, nil
		}
		names = append(names, s.Name)
	}
	return StepSchema{}, fmt.Errorf("unknown step %q; valid steps: %v", step, names)
}

// RunFunc is the goroutine body that runs the domain circuit. It receives
// a context (cancelled on session abort) and returns the domain result
// plus any error.
type RunFunc func(ctx context.Context) (result any, err error)

// SessionMeta carries initial metadata from the domain session factory
// back to the start_circuit response.
type SessionMeta struct {
	TotalCases int
	Scenario   string
}

// StartParams are the parsed parameters from a start_circuit tool call.
// Domain-specific fields live in Extra.
type StartParams struct {
	Parallel int
	Force    bool
	Extra    map[string]any // domain-specific params (scenario, backend, rp_base_url, etc.)
}
