# Cursor Skill Guide

How to build Cursor Skills for Origami-based tools.

## Overview

Every Origami-based tool can ship Cursor Skills — structured instructions
that enable the Cursor AI agent to orchestrate pipeline execution. Skills
use MCP tool calls (zero file writes, zero approval gates) and follow the
agent bus protocol for subagent delegation.

This guide covers: the Three Skills pattern, scaffolding from pipeline YAML,
MCP integration, and provider routing.

## Three Skills Pattern

Every Origami-based tool should ship three Cursor Skills, mirroring the
Three CLIs pattern:

| Skill | CLI equivalent | Purpose |
|-------|---------------|---------|
| `<tool>-run` | `<tool> run` | Execute the pipeline on real input |
| `<tool>-calibrate` | `<tool> calibrate` | Measure accuracy against ground truth |
| `<tool>-dataset` | `<tool> dataset` | Build and curate ground truth datasets |

Start with `<tool>-calibrate` — it exercises the full pipeline and produces
measurable metrics.

## Scaffolding

Generate a SKILL.md from any pipeline YAML:

```bash
origami skill scaffold --tool myapp pipelines/rca.yaml
```

This creates `.cursor/skills/myapp-calibrate/SKILL.md` with:
- Trigger patterns
- Pipeline step table (derived from YAML nodes)
- Edge flow diagram
- MCP tool call instructions (start, pull loop, report)
- Subagent delegation protocol
- Error handling and security guardrails

### Options

| Flag | Default | Description |
|------|---------|-------------|
| `--tool NAME` | pipeline name | Tool name used in skill name and paths |
| `--out DIR` | `.cursor/skills/<tool>-calibrate/` | Output directory |

## MCP Integration

Skills communicate with the tool's CLI through MCP tool calls. The tool
runs as an MCP server (`<tool> serve`) and exposes these tools:

| Tool | Description |
|------|-------------|
| `start_calibration` | Start a run; returns session_id |
| `get_next_step` | Pull next pipeline step; blocks until ready |
| `submit_artifact` | Submit artifact JSON for the current step |
| `get_report` | Get final metrics scorecard |
| `emit_signal` | Emit observability event to the signal bus |
| `get_signals` | Read events from the signal bus |

### Why MCP, not FileDispatcher

The file-based signal protocol (`signal.json`) requires Cursor to approve
every file write. For calibration with N cases × M steps, that means
N×M manual approvals. MCP tool calls have zero approval gates — the agent
calls tools directly.

### Configuration

Add the MCP server to `.cursor/mcp.json`:

```json
{
  "mcpServers": {
    "myapp": {
      "command": "/path/to/bin/myapp",
      "args": ["serve"],
      "cwd": "/path/to/project"
    }
  }
}
```

## Agent Bus Protocol

Every pipeline step MUST be delegated to a Task subagent. The main agent
acts as a dispatcher, not an executor.

| Responsibility | Owner |
|----------------|-------|
| `get_next_step`, `submit_artifact` | Main agent |
| Read prompt, generate artifact | Subagent |
| `emit_signal` (dispatch) | Main agent |
| `emit_signal` (start, done, error) | Subagent |

### Signal protocol

| Event | Emitter | When |
|-------|---------|------|
| `dispatch` | Main agent | Before launching subagent |
| `start` | Subagent | First action |
| `done` | Subagent | Artifact ready |
| `error` | Subagent | On failure |

### Parallel mode

Cursor supports up to 4 concurrent Task subagents in a single message.
For parallel calibration:

1. Pull N steps via `get_next_step`
2. Launch N Task subagents in one message
3. Collect results and call `submit_artifact` for each

The MCP server tracks capacity and warns if you under-pull.

## Dispatchers

Origami provides multiple dispatch mechanisms. Skills should use MCP
(via MuxDispatcher), but tools can also wire other dispatchers:

| Dispatcher | Transport | Use case |
|------------|-----------|----------|
| `MuxDispatcher` | In-memory channels | MCP server ↔ calibration runner |
| `CLIDispatcher` | Subprocess stdin/stdout | External CLI tools (Codex, Claude) |
| `HTTPDispatcher` | OpenAI-compatible API | Direct LLM API calls |
| `StdinDispatcher` | Terminal I/O | Manual/interactive mode |
| `FileDispatcher` | File polling | Non-Cursor automation (deprecated for Cursor) |

### CLIDispatcher

For tools that want to shell out to external LLM CLIs:

```go
d, err := dispatch.NewCLIDispatcher("codex",
    dispatch.WithCLIArgs("--model", "o3"),
    dispatch.WithCLITimeout(2 * time.Minute),
)
```

The dispatcher reads the prompt file, pipes it to the CLI's stdin,
captures stdout as the artifact, and writes it to the artifact path.

## Provider Routing

Pipeline YAML nodes can specify a `provider:` field to route steps to
different LLM backends:

```yaml
nodes:
  - name: triage
    transformer: llm
    provider: cursor
  - name: investigate
    transformer: llm
    provider: codex
```

| Provider | Dispatcher | Description |
|----------|-----------|-------------|
| `cursor` | MuxDispatcher | Cursor agent via MCP |
| `codex` | CLIDispatcher | OpenAI Codex CLI |
| `claude` | CLIDispatcher | Anthropic Claude CLI |
| `openai` | HTTPDispatcher | OpenAI API |
| (empty) | Default | Uses the run's default dispatcher |

## Customizing the Scaffold

The generated SKILL.md is a starting point. Common customizations:

1. **Artifact schemas** — Add step-specific JSON schemas for subagent guidance.
2. **Sticky subagents** — Maintain `case_id → agent_id` map for context continuity across steps.
3. **Domain-specific instructions** — Add tool-specific analysis guidance per step.
4. **Progress reporting** — Customize how metrics are presented.

## Reference Implementations

| Tool | Skill | Status |
|------|-------|--------|
| Asterisk | `asterisk-calibrate` | Production — full MCP-based calibration |
| Asterisk | `asterisk-analyze` | Production — single-launch RCA |
| Achilles | (planned) | Scaffold-ready |
