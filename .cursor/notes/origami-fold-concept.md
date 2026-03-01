# origami fold — Compile YAML Project to Standalone Binary

**Status:** Vision (design during `rca-pure-dsl` Phase 1, implement after)

## Concept

In origami, a fold transforms flat paper into a shaped form. `origami fold` transforms a flat YAML project into a shaped binary.

Consumer repositories (Asterisk, Achilles) contain **zero Go files** — only YAML circuits, scenarios, schemas, prompt templates, and configuration. The Go lives in Origami (the engine) and in Origami's adapters/marbles (the collections). `origami fold` compiles the YAML project into a standalone binary with a custom name.

## How it works

```
asterisk/                          # Zero .go files
├── origami.yaml                   # Project manifest
├── circuits/
│   ├── asterisk-rca.yaml
│   ├── asterisk-ingest.yaml
│   └── asterisk-calibration.yaml
├── scenarios/
├── scorecards/
├── prompts/
├── schema.yaml
└── vocabulary.yaml
```

The manifest (`origami.yaml`):

```yaml
name: asterisk
description: Evidence-based RCA for ReportPortal test failures

imports:
  - origami.adapters.rp
  - origami.adapters.sqlite
  - origami.marbles.rca

cli:
  commands:
    analyze:
      circuit: asterisk-rca
      flags:
        - {name: launch, type: string, required: true}
        - {name: adapter, type: string, default: llm}
    calibrate:
      circuit: asterisk-calibration
      flags:
        - {name: scenario, type: string, required: true}
    ingest:
      circuit: asterisk-ingest
      flags:
        - {name: project, type: string, required: true}
```

Build: `origami fold --output bin/asterisk`

The fold command:
1. Parses the manifest — extracts imports, circuits, CLI commands
2. Resolves adapter imports to Go package paths (FQCN resolution already exists)
3. Generates a temp `main.go` that embeds all YAML and registers adapters
4. Runs `go build` → outputs `bin/asterisk`

## Precedent

| System | Analog |
|--------|--------|
| **Terraform Provider Dev Kit** | Generates Go binary from provider schema |
| **Ansible Execution Environments** | Builds container images from collection requirements |
| **GraalVM native-image** | Compiles JVM app to native binary |
| **Go embed** | Embeds static files at compile time |

## What it eliminates in Asterisk

| Current Go | LOC | Replaced by |
|------------|-----|-------------|
| `cmd/asterisk/` (15 CLI commands) | 2,032 | `cli:` section in manifest |
| `internal/mcpconfig/` (MCP server wiring) | 1,249 | Declarative MCP config in manifest |
| `cmd/asterisk-analyze-rp-cursor/` | 100 | Becomes a CLI command entry in manifest |

Total: ~3,400 LOC of Go wiring replaced by YAML configuration.

## Framework prerequisites

- Adapter FQCN resolution (already exists)
- CLI scaffold via `origami.NewCLI()` (already exists)
- `origami.yaml` manifest spec (new)
- CLI flag → circuit var wiring (new)
- Go code generation template for `main.go` (new)
- MCP server configuration from manifest (new)

## Open questions

- Does `origami fold` require the Go toolchain at fold-time? (Yes, unless we pre-build for common platforms)
- Can the manifest declare MCP tools alongside CLI commands?
- Should `origami fold` also produce container images (`--format=docker`)?
