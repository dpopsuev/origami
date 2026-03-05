# Contract — decoupled-schematics

**Status:** active  
**Goal:** Schematics run as independent processes or containers, communicating via MCP, with a unified knowledge layer replacing the Git-only CodeReader.  
**Serves:** API Stabilization — multi-schematic composition is the final framework primitive before surface freeze.

## Contract rules

- **Test-first.** Every phase begins with integration tests that define the expected behavior.
- **Each commit leaves the build green.** Incomplete Go source (e.g. hooks_inject.go WIP) stays unstaged.
- **Origami breaking changes are expected** (PoC era). Delete over deprecate.
- **Deterministic first.** Clone, search, file read, and container lifecycle are deterministic. Only LLM reasoning over fetched knowledge is stochastic.

## Context

Predecessor: Asterisk `knowledge-sources-github.md` — established source packs + shallow-clone cache. User feedback identified two problems:
1. **OCP-specific leaks** in `SourcePack` (Operator field, hardcoded `release-` branch patterns, GitHub URI template).
2. **No web doc support** — OCP docs are HTTP, not Git.

This led to a broader architectural redesign:
- Unified `knowledge.Reader` interface with pluggable drivers (Git, HTTP docs, future: local files).
- Knowledge as its own Origami schematic, decoupled from RCA.
- Extend `fold/codegen` for multi-schematic composition.
- Schematics as independent OS processes or OCI containers, communicating via MCP.

### Current architecture

```mermaid
graph TD
    subgraph origami_yaml [origami.yaml]
        Imports["imports:\n  - origami.schematics.rca"]
        Bindings["bindings:\n  rca.source: origami.connectors.rp\n  rca.code: origami.connectors.github"]
    end

    subgraph fold [origami fold]
        Codegen["codegen.go\n(single schematic)"]
    end

    subgraph rca [RCA Schematic]
        CodeReader["CodeReader interface\n(Git-only, in-process)"]
        Hooks["inject.code.* hooks"]
    end

    subgraph github [connectors/github]
        RepoCache["RepoCache\nshallow clone + ripgrep"]
    end

    Imports --> Codegen
    Codegen --> rca
    CodeReader --> github
    Hooks --> CodeReader
```

### Desired architecture

```mermaid
graph TD
    subgraph origami_yaml [origami.yaml]
        Imports["imports:\n  - origami.schematics.rca\n  - origami.schematics.knowledge"]
        Bindings["bindings:\n  rca.source: origami.connectors.rp\n  knowledge.git: origami.connectors.github\n  knowledge.docs: origami.connectors.docs"]
        Deploy["deploy:\n  knowledge: {mode: subprocess}"]
    end

    subgraph fold [origami fold]
        Codegen["codegen.go\n(multi-schematic)"]
        Dockerfile["dockerfile.go\n(per-schematic OCI image)"]
    end

    subgraph knowledge_proc [Knowledge Schematic ⚙️ subprocess/container]
        Router["AccessRouter"]
        GitDriver["Git Driver\n(RepoCache + ripgrep)"]
        DocsDriver["Docs Driver\n(docs.redhat.com search)"]
        Router --> GitDriver
        Router --> DocsDriver
    end

    subgraph rca_proc [RCA Schematic ⚙️ main process]
        KnowledgeSocket["knowledge socket\n(Reader interface)"]
        Hooks["inject hooks"]
        KnowledgeSocket --> Hooks
    end

    subgraph subprocess [subprocess/]
        Server["Server\n(stdio MCP transport)"]
        Orchestrator["Orchestrator\n(multi-process + hot-swap)"]
        ContainerMgr["ContainerManager\n(podman/docker lifecycle)"]
    end

    Imports --> Codegen
    Deploy --> Codegen
    Codegen -->|"in-process or subprocess"| rca_proc
    Codegen -->|"subprocess or container"| knowledge_proc
    KnowledgeSocket -->|"MCP over stdio/TCP"| Router
    Orchestrator --> Server
    Orchestrator --> ContainerMgr
```

## FSC artifacts

| Artifact | Target | Compartment |
|----------|--------|-------------|
| Multi-schematic codegen design | `docs/multi-schematic-composition.md` | domain |
| Knowledge Reader/Driver interfaces | code (`schematics/knowledge/`) | domain |
| Subprocess/Container lifecycle | code (`subprocess/`) | domain |

## Execution strategy

Six phases, strictly ordered. Each phase begins with tests, then implementation. Each completed phase leaves the build green.

- **Phase 0** — MCP subprocess transport (independent package, no cross-deps)
- **Phase 1** — Multi-schematic fold/codegen (framework extension)
- **Phase 2** — Knowledge schematic + Git/Docs drivers
- **Phase 3** — Hot-swap (orchestrator, graceful restart)
- **Phase 4** — Containerization (Dockerfile gen, container lifecycle, OCI images)
- **Phase 5** — RCA integration + Asterisk wiring (delete CodeReader, update hooks, wire knowledge schematic)

## Coverage matrix

| Layer | Applies | Rationale |
|-------|---------|-----------|
| **Unit** | yes | AccessRouter dispatch, Dockerfile generation, driver handle/search/read/list |
| **Integration** | yes | MCP tool call round-trip over stdio, subprocess lifecycle, hot-swap binary replacement, container start/stop, codegen template output |
| **Contract** | yes | knowledge.Reader and knowledge.Driver interface contracts; component.yaml socket/satisfies declarations |
| **E2E** | yes | Full fold → build → run with secondary schematic in subprocess mode |
| **Concurrency** | yes | Orchestrator concurrent CallTool, subprocess restart under load, race detector on all tests |
| **Security** | yes | Container port binding, subprocess binary path validation, no secrets in MCP payloads |

## Tasks

### Phase 0 — MCP subprocess transport ✅

- [x] P0.1: Write integration tests — two Go processes communicating via MCP over stdio, tool call round-trip, reconnection after process restart
- [x] P0.2: Implement `subprocess.Server` — child process lifecycle (start/stop/restart), MCP client via `sdkmcp.CommandTransport`, health check (ping), `CallTool`
- [x] P0.3: Build + test gate — `go test -race ./subprocess/...` passes

### Phase 1 — Multi-schematic fold/codegen ✅

- [x] P1.1: Write tests — codegen produces `main.go` that constructs secondary schematic and passes it to primary, both in-process and subprocess modes
- [x] P1.2: Add `Deploy map[string]*DeployConfig` to `fold/Manifest`, `Schematic` field to `SocketEntry`, `Factory`/`Serve` fields to `ComponentMeta`
- [x] P1.3: Extend `buildTemplateContext` — partition bindings by schematic namespace, resolve secondaries, generate construction code
- [x] P1.4: Extend `mainTemplate` — conditional subprocess imports, secondary schematic instantiation or `subprocess.Server` wiring
- [x] P1.5: Build + test gate — `go test -race ./fold/...` passes (including integration build)

### Phase 2 — Knowledge schematic + drivers ✅

- [x] P2.1: Define `knowledge.Reader` and `knowledge.Driver` interfaces in `schematics/knowledge/reader.go`
- [x] P2.2: Implement `AccessRouter` in `schematics/knowledge/access_router.go` — dispatches to registered drivers by `SourceKind`
- [x] P2.3: Write `AccessRouter` unit tests — driver dispatch, unknown kind error, delegation correctness
- [x] P2.4: Create `schematics/knowledge/component.yaml` — factory, serve entrypoint, git + docs sockets
- [x] P2.5: Implement `connectors/github/git_driver.go` — wraps existing `RepoCache`, maps `rca.SearchResult`/`TreeEntry` to knowledge types
- [x] P2.6: Create `connectors/docs/` package — HTTP documentation driver for `docs.redhat.com/search/`, HTML-to-text, local cache with TTL
- [x] P2.7: Write `connectors/docs/driver_test.go` — unit tests for Handles, Ensure, Search, Read, List
- [x] P2.8: Update `connectors/github/component.yaml` — add `satisfies: - socket: git, factory: NewGitDriver`
- [x] P2.9: Add `knowledge` socket to `schematics/rca/component.yaml` with `schematic: origami-knowledge`
- [x] P2.10: Build + test gate — `go test -race ./schematics/knowledge/... ./connectors/docs/... ./connectors/github/...` passes

### Phase 3 — Hot-swap ✅

- [x] P3.1: Write tests — replace subprocess binary while orchestrator is running, verify reconnection and behavior change
- [x] P3.2: Implement `subprocess.Orchestrator` — Register, Start, Stop, Swap, CallTool, Healthy, StopAll
- [x] P3.3: Build + test gate — `go test -race ./subprocess/...` passes

### Phase 4 — Containerization 🔶 (partial)

- [x] P4.1: Implement `fold.GenerateDockerfile` — template-based Dockerfile for schematics with `serve` entrypoint
- [x] P4.2: Write Dockerfile generation tests — content assertions, error for missing serve path, default Go version
- [x] P4.3: Implement `subprocess.ContainerManager` — OCI lifecycle via podman/docker (start/stop/swap, port mapping)
- [x] P4.4: Write `ContainerManager` unit tests — default runtime, custom runtime, unknown image error
- [ ] P4.5: Implement TCP MCP transport — extend MCP client for TCP connections to containerized schematics
- [ ] P4.6: Write integration tests — build schematic OCI image from test binary, start via podman, MCP tool call over TCP, hot-swap container image
- [ ] P4.7: Add `fold --container` flag to generate Dockerfile alongside `main.go`
- [ ] P4.8: Build + test gate — `go test -race ./subprocess/... ./fold/...` passes with container tests

### Phase 5 — RCA integration + Asterisk wiring

- [ ] P5.1: Generalize `SourcePack` — rename `Operator` → `Domain`, add `Docs []SourcePackDoc`, remove hardcoded OCP conventions
- [ ] P5.2: Delete `schematics/rca/code_reader.go` (superseded by `knowledge.Reader`)
- [ ] P5.3: Fix `schematics/rca/hooks_inject.go` — replace `WorkspaceParams`/`buildWorkspaceParams` with knowledge-based injection
- [ ] P5.4: Update RCA inject hooks to use `knowledge.Reader` for code access instead of `CodeReader`
- [ ] P5.5: Update Asterisk `origami.yaml` — add knowledge schematic import + bindings, source packs with doc entries
- [ ] P5.6: Create `schematics/knowledge/cmd/serve/main.go` — MCP server entrypoint for subprocess/container mode
- [ ] P5.7: Full end-to-end test — Asterisk with decoupled knowledge schematic in subprocess mode
- [ ] P5.8: Validate (green) — all tests pass, build green across Origami + Asterisk
- [ ] P5.9: Tune (blue) — refactor for quality, no behavior changes
- [ ] P5.10: Validate (green) — all tests still pass after tuning

## Acceptance criteria

**Given** an `origami.yaml` with two imports (`origami.schematics.rca` + `origami.schematics.knowledge`) and bindings for `knowledge.git` and `knowledge.docs`,  
**When** `origami fold` runs,  
**Then** the generated `main.go` constructs the knowledge schematic with both drivers and passes it to the RCA schematic's `WithKnowledgeReader` option.

**Given** `deploy: { knowledge: { mode: subprocess } }` in `origami.yaml`,  
**When** the generated binary starts,  
**Then** the knowledge schematic runs as a child process communicating via MCP over stdio, and the RCA schematic calls `knowledge.Reader` methods transparently.

**Given** a running knowledge subprocess,  
**When** a new binary is swapped in via `Orchestrator.Swap()`,  
**Then** in-flight requests drain gracefully and subsequent calls use the new binary.

**Given** `fold --container` is invoked,  
**When** the knowledge schematic has a `serve` entrypoint,  
**Then** a Dockerfile is generated that builds a distroless OCI image with the schematic binary.

**Given** the RCA schematic with a `knowledge` socket,  
**When** `knowledge.Reader.Search(ctx, src, "holdover", 10)` is called with a Git source,  
**Then** the AccessRouter dispatches to the Git driver which searches the local shallow clone via ripgrep.

**Given** a `SourceKindDoc` source pointing to `docs.redhat.com`,  
**When** `knowledge.Reader.Search(ctx, src, "ptp grandmaster", 5)` is called,  
**Then** the Docs driver queries the search endpoint and returns parsed results.

## Security assessment

| OWASP | Finding | Mitigation |
|-------|---------|------------|
| A01 Broken Access Control | Subprocess binary path could be manipulated | Binary path validated at registration. No user-supplied paths at runtime. |
| A03 Injection | Container image name from YAML used in `podman run` | Image name passed as argument, not interpolated in shell. Only manifest-declared images accepted. |
| A05 Security Misconfiguration | Container port binding exposes MCP on localhost | Bind to `127.0.0.1` only. Container network isolated by default. |
| A07 SSRF | Docs driver fetches arbitrary URLs from source config | URLs must match configured domain patterns. Source packs are operator-controlled YAML, not user input. |
| A09 Logging/Monitoring | MCP payloads between processes could contain sensitive data | No secrets in MCP tool arguments. Knowledge content (code, docs) stays in-process memory, not logged. |

## Notes

2026-03-05 — Retroactive contract. Phases 0–3 and partial Phase 4 were implemented and shipped before this contract was written. Committed as `3f33b71` (subprocess package) and `bece4c5` (fold multi-schematic + knowledge + drivers + testdata). Phase 4 containerization has Dockerfile generation and ContainerManager scaffolding but lacks TCP MCP transport and full integration tests. Phase 5 (RCA integration) not started — `hooks_inject.go` has pre-existing compilation errors from the workspace-to-sources rename that remain unstaged.

2026-03-04 — User feedback: "knowledge source pack is overly optimized for OCP & OCP Operators" and "OCP docs aren't a GitHub repo but just an HTTP website." This triggered the generalization from Git-only CodeReader to unified Reader/Driver with pluggable backends. User selected "schematic-now" (knowledge as its own schematic), "extend-origami" (fold multi-schematic codegen), and "container-now" (OCI support in scope).
