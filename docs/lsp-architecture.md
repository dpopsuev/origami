# Origami LSP Architecture

## Overview

The Origami Language Server provides IDE-grade editing support for pipeline YAML files. It runs as a subprocess spawned by the editor, communicating over stdin/stdout using the Language Server Protocol (JSON-RPC 2.0).

## Components

```
┌─────────────────────┐    stdio     ┌──────────────────┐
│  VS Code Extension  │◄──────────►│  origami lsp      │
│  (extension.ts)     │   JSON-RPC  │  (Go binary)      │
└─────────────────────┘             └──────┬───────────┘
                                           │
                              ┌────────────┼────────────┐
                              │            │            │
                        ┌─────▼─────┐ ┌────▼────┐ ┌────▼────┐
                        │ Diagnostics│ │Completion│ │  Hover  │
                        │ (lint)     │ │(DSL keys)│ │(elements│
                        └───────────┘ └─────────┘ │personas)│
                                                   └─────────┘
                        ┌───────────┐ ┌─────────┐ ┌──────────┐
                        │ Definition │ │Semantic │ │Inlay     │
                        │ (go-to)    │ │Tokens   │ │Hints     │
                        └───────────┘ └─────────┘ └────┬─────┘
                                                        │
                                                   ┌────▼─────┐
                                                   │Kami Bridge│
                                                   │(SSE live) │
                                                   └──────────┘
```

## Features

### Diagnostics (Phase 1)

Integrates `origami-lint` to report YAML structure errors, invalid elements, orphan nodes, and missing edges in real time. Severity levels map to LSP `DiagnosticSeverity`.

### Completion (Phase 1)

Context-aware suggestions:
- **Top level**: `pipeline`, `nodes`, `edges`, `walkers`, `start`, `done`, etc.
- **Node fields**: `name`, `element`, `family`, `extractor`, etc.
- **Element values**: `fire`, `water`, `earth`, `air`, `diamond`, `lightning`, `iron`
- **Persona values**: `herald`, `seeker`, `sentinel`, `weaver`, `arbiter`, `catalyst`, `oracle`, `phantom`
- **Node references**: in `from:`, `to:`, `start:` fields

### Hover (Phase 1)

Rich markdown hover documentation for elements (description, traits, color), personas (identity and quote), edge expression context (`output`, `state`, `config`), node references (family, element), and top-level keys.

### Go-to-Definition (Phase 1)

Jump from `from:`, `to:`, or `start:` values to the corresponding `- name:` declaration in the nodes section.

### Semantic Tokens (Phase 2)

Seven custom token types, one per Origami element:

| Token Type | Element | Default Color |
|---|---|---|
| `origami-fire` | Fire | Crimson (#DC143C) |
| `origami-water` | Water | Cerulean (#007BA7) |
| `origami-earth` | Earth | Cobalt (#0047AB) |
| `origami-air` | Air | Amber (#FFBF00) |
| `origami-diamond` | Diamond | Sapphire (#0F52BA) |
| `origami-lightning` | Lightning | Crimson (#DC143C) |
| `origami-iron` | Iron | Iron (#48494B) |

Tokens are applied to `element:` values and zone names that carry an element.

### Inlay Hints (Phase 3)

Non-intrusive inline annotations:
- **Element traits**: compact speed/loops/shortcut summary after `element:` values
- **Persona descriptions**: first sentence of persona identity after `persona:` values
- **Expression classification**: `static`, `output-dep`, `state-dep`, or `expr` after `when:` values
- **Element flow**: `fire → water` on edge `id:` lines showing elemental transition
- **Start node**: `entry [element] family=X` after `start:` declaration

### Kami Bridge (Phase 4)

SSE client that connects to a running Kami debugger server and overlays live pipeline state onto inlay hints:
- **ACTIVE [agent]**: node currently being executed
- **PAUSED**: pipeline paused at a breakpoint
- **visited (Xs ago)**: previously visited nodes with timestamp

Configuration via `workspace/didChangeConfiguration`:
- `origami.kami.enabled` (boolean, default: `true`)
- `origami.kami.port` (number, default: `9800`)

Auto-reconnects with exponential backoff (1s → 30s max).

## Installation

### From Source

```bash
# Build the origami binary (includes lsp subcommand)
cd origami && go build -o origami ./cmd/origami

# Install the VS Code extension
cd lsp/vscode && npm install && npm run compile
# Then "Install from VSIX" or link for development
```

### VS Code Settings

```json
{
  "origami.lsp.path": "/path/to/origami",
  "origami.kami.enabled": true,
  "origami.kami.port": 9800
}
```

## Protocol Support

| Capability | Method | Status |
|---|---|---|
| Diagnostics | `textDocument/publishDiagnostics` | ✅ |
| Completion | `textDocument/completion` | ✅ |
| Hover | `textDocument/hover` | ✅ |
| Go-to-definition | `textDocument/definition` | ✅ |
| Semantic tokens | `textDocument/semanticTokens/full` | ✅ |
| Inlay hints | `textDocument/inlayHint` | ✅ |
| Configuration | `workspace/didChangeConfiguration` | ✅ |
| Document sync | Full (`didOpen`, `didChange`, `didClose`) | ✅ |
