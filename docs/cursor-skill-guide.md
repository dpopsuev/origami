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

## Agent Bus Protocol (Papercup v2)

The agent bus follows the **Papercup v2 choreography** protocol. The parent
agent is a **supervisor** — it never calls `get_next_step` or `submit_artifact`.
Workers own the full pipeline loop independently.

| Responsibility | Owner |
|----------------|-------|
| `start_calibration`, `get_report` | Supervisor (parent) |
| Launch worker Tasks with `worker_prompt` | Supervisor (parent) |
| Monitor via `get_signals` | Supervisor (parent) |
| `get_next_step`, `submit_artifact` | Worker (subagent) |
| Read `prompt_content`, generate artifact | Worker (subagent) |
| `emit_signal` (worker_started, worker_stopped) | Worker (subagent) |

### Supervisor workflow

1. Call `start_calibration(parallel=N)` — returns `session_id`, `worker_prompt`, `worker_count`.
2. Launch `worker_count` Task subagents with the server-generated `worker_prompt` (verbatim).
3. Monitor via `get_signals(since=N)` — observe `worker_started`, `artifact_submitted`, `worker_stopped`.
4. When all workers stop or `session_done` appears, call `get_report`.

### Worker protocol

Each worker (launched as a Task subagent) follows this sequence:

1. Emit `worker_started` signal with `meta.mode = "stream"` and `meta.worker_id`.
2. Loop: `get_next_step` → read `prompt_content` (inline) → `submit_artifact` → repeat.
3. Exit when `get_next_step` returns `done=true`.
4. Emit `worker_stopped` signal.

Workers MUST NOT wait for siblings. Each worker processes steps independently
as they become available (competing consumers pattern). The MCP server routes
`submit_artifact` to the correct dispatch via `dispatch_id`.

### Signal protocol

| Event | Emitter | When |
|-------|---------|------|
| `worker_started` | Worker | First action; includes `meta.mode` and `meta.worker_id` |
| `start` | Worker | Before processing a step |
| `done` | Worker | After submitting artifact for a step |
| `worker_stopped` | Worker | After loop exits (pipeline drained) |
| `error` | Worker or Supervisor | On failure |

### Server-generated worker prompt

The `worker_prompt` field in the `start_calibration` response contains
complete instructions for the worker loop, including:
- The `session_id` (embedded, not a placeholder)
- Step schemas (F0-F6 with required fields)
- Signal emission protocol
- Calibration rules (blind evaluation constraints)

Workers receive `prompt_content` inline in each `get_next_step` response —
no file I/O needed. This eliminates file read latency and permission issues.

### Capacity detection

The MCP server detects the v2 pattern via `peakPullers` — when N workers call
`get_next_step` concurrently, the server knows N independent workers exist.
The capacity gate uses this to verify the system has enough workers.

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
