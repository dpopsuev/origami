# Adapter → Component Rename: Evaluation Decision

**Date:** 2026-03-02
**Contract:** naming-taxonomy (Phase 4)
**Decision:** Defer — rename not justified during PoC.

## Audit

| Metric | Count |
|--------|-------|
| Files referencing `Adapter` | 33 |
| Total occurrences | ~200+ |
| Framework root (`adapter.go`, `adapter_test.go`, `dsl.go`, `run.go`) | 4 files, ~60 refs |
| Core transformers (`transformers/core_adapter.go`) | 1 file, 9 refs |
| CLI (`cmd/origami/cmd_adapter.go`) | 1 file, 8 refs |
| Modules (`modules/rca/adapter.go`, `adapter_test.go`, etc.) | ~20 files, ~100 refs |
| Studio, calibrate, dispatch | 5 files, ~15 refs |

## Analysis

**"Adapter" is semantically accurate.** In the Origami DSL, an `Adapter` bundles transformers, extractors, and hooks into a namespace — it _adapts_ a domain's processing logic to the framework's interfaces. This is textbook Adapter pattern.

**"Component" is marginally more intuitive** for newcomers who think of a "component" as a self-contained unit. But "Adapter" already communicates the relationship: the Adapter adapts domain logic to the framework.

**Risk/reward during PoC:**
- ~200 occurrences across 33 files = high mechanical risk.
- Zero functional or API change.
- The `Adapter` struct, registry, and CLI surface would all need coordinated renaming across Origami + Asterisk + Achilles.
- PoC API stability rule says "delete over deprecate, no shim layers" — a rename would be a large diff with no shim, but also no user-facing benefit.

## Decision

Defer the rename. Re-evaluate when the PoC-Done flag is raised and naming conventions are locked for external consumers. If the rename happens post-PoC, it should be a standalone atomic contract with a migration window.
