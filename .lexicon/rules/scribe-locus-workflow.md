---
id: scribe-locus-workflow
title: Scribe + Locus Workflow
description: How to use Scribe and Locus MCP tools for governance and architecture context
labels: [process, mcp]
always_apply: true
---

# Scribe + Locus Workflow

## Session start

1. `motd` -- read current goal and reminders.
2. `list_artifacts(kind=contract)` -- see active/draft contracts.
3. If a sprint is active, `contract_tree(id)` for the full board.

## Before changes

- `list_artifacts` with status/priority filters to find the next contract.
- `scan_project` (Locus) for architecture context: coupling hotspots, dependencies.

## During work

- `set_field(id, field, value)` to update contract status (draft -> active -> complete).
- `attach_section(id, name, text)` for diagrams, notes, or mermaid charts.
- `create_artifact` for new contracts, sprints, goals. Use `parent` for hierarchies, `depends_on` for sequencing.

## Cross-scope

- `list_artifacts` defaults to home scope. Pass explicit `scope` for other projects.
- `get_artifact(id)` works across all scopes.

## Locus tools

- `scan_project` -- codograph (packages, imports, symbols, LOC, churn)
- `diff_branches` -- structural diff between branches
- `get_codograph_history` -- past scans and diffs
