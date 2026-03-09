---
id: agent-bus
title: Papercup Agent-Bus Protocol
description: Mandatory signaling protocol for agent/subagent coordination during circuit execution
labels: [protocol, agentic]
---

# Papercup Agent-Bus Protocol

## Delegation mandate

Every circuit step MUST be processed by a worker. The parent agent is a **supervisor**, not an executor.

| Responsibility | Owner |
|---|---|
| `start_calibration`, `get_report` | Parent (supervisor) |
| `get_next_step`, `submit_step` | Worker subagent |
| Read prompt, generate artifact | Worker subagent |
| Monitor health, replace failures | Parent (supervisor) |

Processing a step inline (parent doing the work) is a violation.

## Worker loop

```
emit_signal(session_id, "worker_started", "worker", meta={worker_id})
while true:
    response = get_next_step(session_id, timeout_ms: 30000)
    if response.done: break
    if not response.available: continue
    artifact = generate_artifact(read(response.prompt_path))
    submit_step(session_id, dispatch_id, step, fields)
emit_signal(session_id, "worker_stopped", "worker", meta={worker_id})
```

## Supervisor pattern

```
session = start_calibration(scenario, adapter, parallel=N)
launch N Task worker subagents
while not done:
    signals = get_signals(session_id, since=last_index)
    if worker_error: launch replacement
    if all workers stopped: break
report = get_report(session_id)
```

## Signal protocol

Worker-emitted: `worker_started`, `start`, `done` (with `bytes`), `error`, `worker_stopped`.

Auto-emitted (server-side, do not duplicate): `session_started`, `step_ready`, `artifact_submitted`, `circuit_done`, `session_done`, `session_error`.
