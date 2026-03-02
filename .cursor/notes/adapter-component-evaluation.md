# Adapter → Component Rename: Evaluation Decision

**Date:** 2026-03-02
**Contract:** naming-taxonomy (Phase 4)
**Decision:** Executed — rename completed.

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

**"Component" was selected** over "Adapter", "Plugin", "Extension", and "Provider" based on:

1. **Electronics metaphor alignment** — a component is a discrete functional block on a PCB; matches Origami's circuit vocabulary (Node, Edge, Zone, Circuit).
2. **Semantic accuracy** — Components bundle capabilities (transformers, extractors, hooks) under a namespace. They are self-contained units, not GoF adapters bridging incompatible interfaces.
3. **Reduced overloading** — "Adapter" was overloaded: `framework.Adapter`, `store.EnvelopeStoreAdapter` (GoF adapter), `adapter_routing.go` (wrapper). Renaming the framework concept to "Component" disambiguates.
4. **Developer intuition** — "Component" is immediately understood as "a unit I install/import".

## Rename Scope

| Before | After | Scope |
|--------|-------|-------|
| `framework.Adapter` | `framework.Component` | Struct, types, functions |
| `adapter.go` | `component.go` | File rename |
| `adapters/` | `components/` | Directory rename + import paths |
| `adapter.yaml` | `component.yaml` | YAML manifest files |
| `origami adapter` | `origami component` | CLI command |
| `calibrate.ModelAdapter` | `calibrate.ModelBackend` | Interface |
| `--adapter` flag | `--backend` flag | CLI flags |

**Excluded (GoF adapter pattern, correct as-is):**
- `store.EnvelopeStoreAdapter` — true GoF adapter bridging store interfaces
- `adapter_routing.go` / `RoutingRecorder` — wraps transformers, not framework.Component
- `framework_adapters.go` — bridge functions between framework types and RCA domain

## Outcome

Rename completed across Origami and Asterisk. All tests pass including race detector.
