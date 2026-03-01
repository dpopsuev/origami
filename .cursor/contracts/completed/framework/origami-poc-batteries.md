# Contract — origami-poc-batteries

**Status:** complete  
**Goal:** Every BYO interface has at least one trivial PoC-ready implementation so new users can prototype in 5 minutes.  
**Serves:** Framework Maturity (current goal)

## Contract rules

Global rules only, plus:

- **Minimal, not production-grade.** Each battery is the simplest possible implementation. Enough to prototype, not enough to deploy. The consumer replaces it when serious.
- **One per BYO.** Deliver exactly one battery per BYO principle: BYOB, BYOA, BYOS. BYOD and BYOI are covered by existing primitives (`ProcessWalker`, built-in transformers, `origami run` CLI).

## Context

- `strategy/origami-vision.mdc` — "Batteries for PoC" table defines what ships.
- `notes/framework-maturity-assessment.md` — Gap #3: PoC batteries incomplete.
- `dispatch/` — Existing dispatchers: `StdinDispatcher`, `FileDispatcher`, `MuxDispatcher`, `BatchDispatcher`.
- `curate/` — Existing `curate.Store` interface with `FileStore` reference implementation.
- `walker.go` — `WalkerState` (in-memory, not serializable).

### Deliverables

| BYO | Battery | Description |
|---|---|---|
| BYOB | `HTTPDispatcher` | POST to OpenAI-compatible `/v1/chat/completions`. Reads `OPENAI_API_KEY` from env. Returns assistant message content as string. |
| BYOA | `StaticTokenAuth` | Middleware that injects a bearer token from env var into HTTP dispatchers. |
| BYOS (checkpoint) | `JSONCheckpointer` | Dumps `WalkerState` to a JSON file between nodes for resume-from-failure. |
| BYOS (store) | `MemoryStore` | In-memory implementation of `curate.Store` for tests and prototypes. |

## FSC artifacts

| Artifact | Target | Compartment |
|----------|--------|-------------|
| PoC batteries catalog update | `docs/` | domain |

## Execution strategy

Implement each battery independently. Each has its own unit test. Order: HTTPDispatcher, StaticTokenAuth, JSONCheckpointer, MemoryStore. Then validate all together.

## Coverage matrix

| Layer | Applies | Rationale |
|-------|---------|-----------|
| **Unit** | yes | Each battery: happy path, error handling, edge cases |
| **Integration** | yes | HTTPDispatcher with a mock HTTP server; JSONCheckpointer round-trip with Walk |
| **Contract** | yes | Each battery implements its respective BYO interface |
| **E2E** | no | These are PoC stubs, not production paths |
| **Concurrency** | no | Single-threaded usage |
| **Security** | yes | HTTPDispatcher handles API keys; StaticTokenAuth handles bearer tokens |

## Tasks

- [x] Implement `HTTPDispatcher` in `dispatch/http.go` — POST to OpenAI-compatible API, parse response
- [x] Implement `StaticTokenAuth` in `dispatch/auth.go` — middleware injecting bearer token from env var
- [x] Implement `JSONCheckpointer` in `checkpoint.go` — serialize/deserialize `WalkerState` to JSON file
- [x] Implement `MemoryStore` in `curate/memory.go` — in-memory `curate.Store`
- [x] Unit tests for all four batteries
- [x] Integration test: HTTPDispatcher with httptest.Server
- [x] Validate (green) — all tests pass
- [x] Tune (blue) — refactor for quality
- [x] Validate (green) — all tests still pass after tuning

## Acceptance criteria

**Given** a new user wanting to prototype an Origami circuit,  
**When** they use only framework-provided batteries (no custom code),  
**Then**:
- `HTTPDispatcher` sends prompts to an OpenAI-compatible API and returns responses
- `StaticTokenAuth` injects a bearer token from `$OPENAI_API_KEY` (or configurable env var)
- `JSONCheckpointer` persists walk state to disk and restores it for resume
- `MemoryStore` provides a working `curate.Store` for tests
- All batteries implement their respective BYO interfaces
- `go build ./...` and `go test ./...` pass in Origami

## Security assessment

| OWASP | Finding | Mitigation |
|-------|---------|------------|
| A02 Cryptographic Failures | `StaticTokenAuth` reads tokens from env vars. Tokens could leak in logs. | Never log token values. Redact in debug output. Document as PoC-only, not production-grade. |
| A07 Authentication | `HTTPDispatcher` sends API keys over HTTP if misconfigured. | Default to HTTPS. Warn (or reject) plain HTTP URLs. |
| A10 SSRF | `HTTPDispatcher` sends requests to user-specified URLs. | Validate URL scheme (https only by default). Document allowed hosts pattern. |

## Notes

2026-02-18 — Contract created. Quick-win for Framework Maturity goal. Closes gap #3 from the maturity assessment.
