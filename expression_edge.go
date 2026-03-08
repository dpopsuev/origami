package framework

// Category: Core Primitives

import (
	"encoding/json"
	"fmt"
	"path"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
)

// ExprContext is the evaluation context passed to when: expressions.
// Fields are lowercase to match YAML/expression conventions.
type ExprContext struct {
	Output  map[string]any   `expr:"output"`
	State   ExprState        `expr:"state"`
	Config  map[string]any   `expr:"config"`
	Signals SignalExprHelpers `expr:"signals"`
}

// ExprState exposes walker state to expressions.
type ExprState struct {
	Loops   map[string]int `expr:"loops"`
	Current string         `expr:"current"`
}

// expressionEdge evaluates a compiled expr-lang program against artifact + state.
// Created by BuildGraph when EdgeDef.When is non-empty.
type expressionEdge struct {
	def     EdgeDef
	program *vm.Program
	config  map[string]any
}

// CompileExpressionEdge compiles a When expression with a typed environment.
// Called at graph build time; compilation errors surface immediately.
// Optional config is stored and passed to the expression context at evaluation time.
func CompileExpressionEdge(def EdgeDef, config ...map[string]any) (*expressionEdge, error) {
	if def.When == "" {
		return nil, fmt.Errorf("edge %s: When expression is empty", def.ID)
	}

	program, err := expr.Compile(def.When,
		expr.Env(ExprContext{}),
		expr.AsBool(),
	)
	if err != nil {
		return nil, fmt.Errorf("edge %s: compile expression %q: %w", def.ID, def.When, err)
	}

	var cfg map[string]any
	if len(config) > 0 {
		cfg = config[0]
	}

	return &expressionEdge{def: def, program: program, config: cfg}, nil
}

func (e *expressionEdge) ID() string       { return e.def.ID }
func (e *expressionEdge) From() string     { return e.def.From }
func (e *expressionEdge) To() string       { return e.def.To }
func (e *expressionEdge) IsShortcut() bool { return e.def.Shortcut }
func (e *expressionEdge) IsLoop() bool     { return e.def.Loop }
func (e *expressionEdge) IsParallel() bool { return e.def.Parallel }

func (e *expressionEdge) Evaluate(artifact Artifact, state *WalkerState) *Transition {
	ctx := buildExprContext(artifact, state, e.config)

	result, err := expr.Run(e.program, ctx)
	if err != nil {
		return nil
	}

	matched, ok := result.(bool)
	if !ok || !matched {
		return nil
	}

	return &Transition{
		NextNode:    e.def.To,
		Explanation: fmt.Sprintf("when: %s", e.def.When),
	}
}

// runExprProgram runs a compiled expression program against a context.
func runExprProgram(program *vm.Program, ctx ExprContext) (any, error) {
	return expr.Run(program, ctx)
}

// buildExprContext creates the evaluation context from artifact and walker state.
// config is populated by circuit vars (C3); nil defaults to empty map.
// If state.Context contains a FindingCollector under FindingCollectorKey,
// the Signals helpers are populated for finding-aware edge conditions.
func buildExprContext(artifact Artifact, state *WalkerState, config map[string]any) ExprContext {
	output := artifactToMap(artifact)

	loops := make(map[string]int)
	current := ""
	var collector FindingCollector
	if state != nil {
		for k, v := range state.LoopCounts {
			loops[k] = v
		}
		current = state.CurrentNode
		collector, _ = state.Context[FindingCollectorKey].(FindingCollector)
	}

	if config == nil {
		config = make(map[string]any)
	}

	return ExprContext{
		Output:  output,
		State:   ExprState{Loops: loops, Current: current},
		Config:  config,
		Signals: SignalExprHelpers{collector: collector},
	}
}

// SignalExprHelpers exposes finding queries to when: expressions.
// Methods are callable as signals.HasFinding("error"), etc.
type SignalExprHelpers struct {
	collector FindingCollector
}

// HasFinding returns true if any finding is at or above the given severity.
func (h SignalExprHelpers) HasFinding(severity string) bool {
	if h.collector == nil {
		return false
	}
	threshold := FindingSeverity(severity)
	for _, f := range h.collector.Findings() {
		if SeverityAtOrAbove(f.Severity, threshold) {
			return true
		}
	}
	return false
}

// FindingCount returns the number of findings at or above the given severity.
func (h SignalExprHelpers) FindingCount(severity string) int {
	if h.collector == nil {
		return 0
	}
	threshold := FindingSeverity(severity)
	count := 0
	for _, f := range h.collector.Findings() {
		if SeverityAtOrAbove(f.Severity, threshold) {
			count++
		}
	}
	return count
}

// FindingDomain returns true if any finding matches the domain glob pattern.
func (h SignalExprHelpers) FindingDomain(domain string) bool {
	if h.collector == nil {
		return false
	}
	for _, f := range h.collector.Findings() {
		if matched, _ := path.Match(domain, f.Domain); matched {
			return true
		}
	}
	return false
}

// artifactToMap converts an Artifact's Raw() value to a map[string]any
// suitable for expression evaluation. Structs are marshaled through JSON.
func artifactToMap(artifact Artifact) map[string]any {
	if artifact == nil {
		return make(map[string]any)
	}

	raw := artifact.Raw()
	if raw == nil {
		return make(map[string]any)
	}

	if m, ok := raw.(map[string]any); ok {
		return m
	}

	data, err := json.Marshal(raw)
	if err != nil {
		return map[string]any{"_raw": raw}
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return map[string]any{"_raw": raw}
	}

	return m
}
