# Contract — code-housekeeping

**Status:** complete  
**Goal:** All three repos (Origami, Asterisk, Achilles) are free of stale pre-distillation references, dead code, and incorrect paths.  
**Serves:** Framework Maturity (current goal)

## Contract rules

- Cross-repo contract: changes span Origami, Asterisk, and Achilles.
- No behavioral changes — only comments, paths, dead code removal, and CLI consolidation.
- Each repo must build and test green independently before committing.

## Context

After the Asterisk-to-Origami distillation, all three repos carry stale references to `pkg/framework/`, `Provider: "asterisk"`, dead functions, and a standalone `cmd/metacal` binary that belongs as an `origami` subcommand.

## FSC artifacts

Code only — no FSC artifacts. Index updates in affected `.cursor/` directories.

## Execution strategy

Work repo by repo: Origami first (upstream), then Asterisk, then Achilles. Build + test each before committing.

## Coverage matrix

| Layer | Applies | Rationale |
|-------|---------|-----------|
| **Unit** | yes | Existing tests must stay green; provider rename touches identity_test.go |
| **Integration** | yes | CLI consolidation requires metacal subcommand test |
| **Contract** | no | No API schema changes |
| **E2E** | no | No pipeline behavior changes |
| **Concurrency** | no | No shared state changes |
| **Security** | no | No trust boundaries affected |

## Tasks

### Origami

- [x] O1: Consolidate `cmd/metacal` into `cmd/origami` as `origami metacal` subcommand group
- [x] O2: Fix default `runs-dir` from `pkg/framework/metacal/runs` to `metacal/runs`
- [x] O3: Rename `Provider: "asterisk"` to `"origami"` in `known_models.go` and `identity_test.go`
- [x] O4: Reword `curate/record.go` doc comment to remove Asterisk references

### Asterisk

- [x] A1: Fix `examples/framework/main.go` stale `pkg/framework/` and docs paths
- [x] A2: Remove dead `pkg/framework/metacal/runs` and `/metacal` from `.gitignore`
- [x] A3: Delete `papert-paradigm.mdc` from Asterisk (belongs in Origami); verify Origami has it
- [x] A4: Update `.cursor/rules/index.mdc` after papert rule removal

### Achilles

- [x] C1: Fix `main.go` doc comment (`pkg/framework/` -> `github.com/dpopsuev/origami`)
- [x] C2: Remove unused `RegisterExtractors()` function
- [x] C3: Remove nonexistent `edges.go` from `project-standards.mdc`

### Finalize

- [x] Build + test all three repos (`go build ./...` && `go test -race ./...`)
- [ ] Ship commits across all three repos
- [x] Validate (green) — all tests pass, acceptance criteria met.
- [x] Tune (blue) — refactor for quality. No behavior changes.
- [x] Validate (green) — all tests still pass after tuning.

## Acceptance criteria

- Given any `.go` file across all three repos, when grepped for `pkg/framework`, then zero matches in code (comments referencing history are acceptable in `.cursor/` docs).
- Given `origami metacal prompt`, when run, then it produces the same output as the old `metacal prompt`.
- Given `go build ./...` in each repo, when run, then exit code 0.
- Given `go test -race ./...` in each repo, when run, then all tests pass.
- Given `known_models.go`, when inspected, then no `Provider: "asterisk"` entries exist.

## Security assessment

No trust boundaries affected.

## Notes

2026-02-23 22:10 — Contract created from cross-repo audit. All items are cosmetic/structural, no behavioral changes.
