# Case Study: Ansible Collections — Package Distribution for Agentic Frameworks

**Date:** 2026-02-25  
**Subject:** Ansible Collections — the standard distribution format for Ansible content  
**Source:** `docs.ansible.com/projects/ansible/latest/collections_guide/`, `ansible-language-server.readthedocs.io/en/latest/`  
**Purpose:** Map Ansible's Collections + Galaxy model to Origami. Identify what reusable content consumers produce today, define a distribution format, and prevent the duplication problem as the consumer ecosystem grows.

---

## 1. What Ansible Collections Are

Ansible Collections are distribution packages bundling playbooks, roles, modules, and plugins into versioned, namespaced units. They solve the "everybody reinvents the same module" problem that plagued Ansible before Collections were introduced (Ansible 2.10+).

Key components:

- **Collection format**: A directory structure (`plugins/modules/`, `roles/`, `playbooks/`) with a `galaxy.yml` manifest declaring namespace, name, version, dependencies, and supported Ansible versions.
- **Galaxy (registry)**: A central server (`galaxy.ansible.com`) where collections are published, discovered, and installed. Red Hat also offers Automation Hub for certified, enterprise-grade collections.
- **ansible-galaxy CLI**: `install`, `list`, `remove`, `verify`, `download`. Supports `requirements.yml` for pinning collection versions.
- **FQCNs (Fully Qualified Collection Names)**: `namespace.collection.module_name` — eliminates name collisions. The `collections:` keyword in playbooks allows shorthand after declaration.
- **Ansible Language Server**: Provides IDE-native intelligence for playbooks — syntax highlighting, validation (YAML + ansible-lint), auto-completion of module names and options with FQCN awareness, documentation on hover, and go-to-definition for module implementations.

---

## 2. The Duplication Problem in Origami

Today, Origami has two consumers (Asterisk, Achilles) and a third planned (future operators). Already, duplication is visible:

| Pattern | Asterisk | Achilles | Shared? |
|---------|----------|----------|---------|
| JSON artifact extraction | `StepExtractor[T]` (generic) | `GovulncheckExtractor` (domain) | No — same pattern, different implementations |
| Node registry wiring | `buildNodeRegistry()` — 7 passthrough nodes | `NodeRegistry(repoPath)` — 4 real nodes | No — same ceremony, different factories |
| Circuit YAML embedding | `//go:embed circuit_rca.yaml` | `circuitPath()` via `runtime.Caller` | No — different embed strategies |
| Edge routing | `when:` expressions in YAML | `when:` expressions in YAML | Yes — both use the framework's expr-lang engine |
| Store/persistence hooks | `StoreHooks(st, caseData)` — 5 hooks | None | No — Achilles has no persistence |
| MCP server wiring | `mcpconfig.NewServer()` with domain hooks | None | No — Achilles has no MCP |
| LLM transformer | Wrapped in orchestrate runner | Not used | No |

As the consumer count grows (N operators across RAN, Core, edge, platform per the Red Hat telco strategy), each consumer will independently build:

1. An LLM transformer tailored to their domain prompt format
2. A JSON extractor for their artifact types
3. A persistence hook pattern for their store
4. A MCP server configuration for their calibration
5. A circuit YAML for their domain graph

Without a sharing mechanism, this is O(N) duplicated work. Collections reduce it to O(1) shared + O(N) domain-specific.

---

## 3. Concept Mapping: Ansible to Origami

| Ansible Concept | Origami Equivalent | Status |
|----------------|-------------------|--------|
| **Module** (task action) | `Transformer`, `Extractor`, `Node` factory | Exist as Go interfaces. No packaging. |
| **Role** (reusable task bundle) | Circuit YAML (reusable graph pattern) | Exist as files. No distribution. |
| **Plugin** (callback, filter, lookup) | `Hook`, `EdgeFactory`, `Mask` | Exist as Go interfaces. No packaging. |
| **Collection** (distribution bundle) | **Missing** — no bundle format | |
| **galaxy.yml** (manifest) | **Missing** — no manifest | |
| **Galaxy** (registry/discovery) | **Missing** — no registry | |
| **`ansible-galaxy install`** (CLI) | **Missing** — no install command | |
| **FQCNs** (`namespace.collection.module`) | **Missing** — no namespacing | Registry keys are flat strings |
| **`collections:` keyword** (playbook import) | **Missing** — no DSL import | |
| **Ansible Language Server** (IDE intelligence) | **Missing** — no LSP | Planned: `origami-lsp` contract |

---

## 4. Origami Collection Format Design

### 4.1 What a collection contains

A collection is a **Go module** exporting typed registries:

```go
type Collection struct {
    Namespace    string
    Name         string
    Version      string
    Description  string
    Transformers TransformerRegistry
    Extractors   ExtractorRegistry
    Nodes        NodeRegistry
    Hooks        HookRegistry
    Circuits    map[string][]byte // name → embedded YAML
}
```

### 4.2 Manifest (`collection.yaml`)

```yaml
collection: vuln-tools
namespace: achilles
version: 0.1.0
description: "Vulnerability scanning circuit, extractors, and nodes"

provides:
  circuits:
    - name: vuln-scan
      path: circuits/achilles.yaml
      description: "4-node scan→classify→assess→report circuit"
  extractors:
    - name: govulncheck-v1
      description: "Parses govulncheck JSON streaming output"
    - name: classify-v1
      description: "Deduplicates findings and assigns severity"
  nodes:
    - family: scan
    - family: classify
    - family: assess
    - family: report

requires:
  origami: ">=0.3.0"
```

### 4.3 FQCNs in circuit YAML

```yaml
nodes:
  - name: scan
    extractor: achilles.govulncheck-v1    # namespace.name
  - name: analyze
    transformer: core.llm                 # built-in core collection
```

The `imports:` section provides shorthand (like Ansible's `collections:` keyword):

```yaml
imports:
  - achilles.vuln-tools
  - core

nodes:
  - name: scan
    extractor: govulncheck-v1             # resolved via imports
```

### 4.4 Collection merge API

```go
func MergeCollections(base GraphRegistries, colls ...*Collection) GraphRegistries
```

Consumers call `MergeCollections` to combine framework registries with collection registries. Name collisions are detected and reported as errors (not silently overwritten).

### 4.5 CLI

```
origami collection list                            # installed collections
origami collection install github.com/org/name     # go get + register
origami collection inspect achilles.vuln-tools     # show manifest
origami collection validate                        # verify all provides are resolvable
```

Since Origami is Go, "install" means `go get` the module. The CLI wraps `go get` + manifest validation + registry registration.

---

## 5. Candidate Collections

### 5.1 `core` (ships with Origami)

| Type | Name | Source |
|------|------|--------|
| Transformer | `llm` | `transformers/llm.go` |
| Transformer | `http` | `transformers/http.go` |
| Transformer | `jq` | `transformers/jq.go` |
| Transformer | `file` | `transformers/file.go` |
| Extractor | `step-extractor` | Generic JSON → typed struct (from Asterisk's `StepExtractor`) |
| Circuit | `ouroboros-probe` | `ouroboros/circuits/ouroboros-probe.yaml` |

### 5.2 `rca-tools` (from Asterisk)

| Type | Name | Source |
|------|------|--------|
| Circuit | `rca-investigation` | `circuit_rca.yaml` |
| Circuit | `defect-dialectic` | `defect-dialectic.yaml` |
| Hook pattern | `store-hooks` | `StoreHooks` — per-step persistence |
| Node pattern | `passthrough-bridge` | `passthroughNode` for external runner delegation |

### 5.3 `vuln-tools` (from Achilles)

| Type | Name | Source |
|------|------|--------|
| Circuit | `vuln-scan` | `circuits/achilles.yaml` |
| Extractor | `govulncheck-v1` | `GovulncheckExtractor` |
| Extractor | `classify-v1` | `ClassifyExtractor` |
| Node pattern | `scan-classify-assess-report` | 4-node vuln circuit pattern |

---

## 6. Competitive Advantages

### 6.1 Go module system as distribution

Ansible Collections require a custom distribution format (tarballs + Galaxy API). Origami collections are Go modules — they use the existing Go module ecosystem (`go get`, `go.sum`, `proxy.golang.org`). No custom registry infrastructure needed for basic distribution.

### 6.2 Compile-time type safety

Ansible Collections discover modules at runtime (Python import). Origami collections are compiled — a missing transformer or extractor is caught at build time, not at circuit walk time. FQCN resolution can be validated statically.

### 6.3 Single binary

Collections are compiled into the consumer binary. No runtime dependency on collection files or a package manager. `go build` produces one binary with all collections embedded.

### 6.4 LSP integration

The planned `origami-lsp` can resolve FQCNs, validate collection references, and provide completion for collection-provided transformers, extractors, and node families — all in the editor, before running the circuit.

---

## 7. Gaps Compared to Ansible

### Gap 1: No registry / discovery

Ansible has Galaxy. Origami would rely on Go module discovery (pkg.go.dev, GitHub search). For enterprise use, a curated registry (like Automation Hub) may be needed eventually.

**Actionable:** Start without a registry. Use Go modules + a `README.md` listing known collections. Build a registry if the collection count exceeds a manageable list.

### Gap 2: No runtime loading

Go is compiled — collections can't be loaded at runtime from disk. Ansible modules can be discovered at runtime. This is a tradeoff: Origami gets type safety and performance; Ansible gets flexibility.

**Actionable:** Accept this tradeoff. Runtime loading is not aligned with Origami's design philosophy (compile-time safety, single binary). Collections are Go packages, imported at build time.

### Gap 3: No versioned dependency resolution

`go get` handles version resolution, but collection-level version constraints (`requires: origami >= 0.3.0`) need validation beyond what `go.mod` provides.

**Actionable:** The `origami collection validate` command checks that the Go module version satisfies the collection's `requires` field. Advisory, not enforced — Go's module system is the authority.

---

## 8. Actionable Takeaways

1. **Collection struct + manifest** — Define the `Collection` type and `collection.yaml` format. This is the foundation everything else builds on.

2. **FQCN resolution** — Add namespace-aware lookup to all registries (`TransformerRegistry.Get("achilles.govulncheck-v1")`). Backward-compatible: unqualified names resolve as today.

3. **MergeCollections API** — Helper to combine base registries with collection registries, with collision detection.

4. **Core collection** — Extract Origami's built-in transformers (llm, http, jq, file) into a `core` collection that ships with the framework.

5. **`imports:` in CircuitDef** — Add `Imports []string` field to `CircuitDef` for FQCN shorthand. `omitempty` ensures backward compatibility.

6. **CLI** — `origami collection list/install/inspect/validate` wrapping `go get` + manifest parsing.

7. **LSP awareness** — The planned LSP should resolve FQCNs, validate collection references, and complete collection-provided names.

---

## References

- Ansible Collections guide: `docs.ansible.com/projects/ansible/latest/collections_guide/`
- Ansible Language Server: `ansible-language-server.readthedocs.io/en/latest/`
- Ansible Galaxy: `galaxy.ansible.com`
- Red Hat Automation Hub: `console.redhat.com/ansible/automation-hub`
- Origami registries: `dsl.go` (NodeRegistry, EdgeFactory), `transformer.go` (TransformerRegistry), `extractor.go` (ExtractorRegistry), `hook.go` (HookRegistry)
- Origami circuit resolution: `resolve.go` (ResolveCircuitPath, RegisterEmbeddedCircuit)
- Origami built-in transformers: `transformers/` (llm, http, jq, file)
- Related contracts: `origami-lsp` (IDE intelligence), `consumer-ergonomics` (API polish), `e2e-dsl-testing` (E2E coverage)
