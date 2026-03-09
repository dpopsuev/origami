# Project Standards

## Product definition

- Origami: Go library (`github.com/dpopsuev/origami`) for graph-based agentic circuit orchestration.
- Primitives: Node, Edge, Graph, Walker, Extractor, Element, Persona, Mask, Adversarial Dialectic, Ouroboros.
- Design: low floor, high ceiling, wide walls (Papert Paradigm).

## Methodology

- Start stories with Gherkin (Given/When/Then).
- Red-Orange-Green-Yellow-Blue cycle.
- Run `go build ./...` and `go test ./...` after every change.
- **Deterministic first.** LLM only for genuine reasoning. See deterministic-first.mdc for D/S boundary.
- Zero domain imports: never import from Asterisk, Achilles, or other consumers.

## API stability (PoC only)

When PoC Done: require deprecation-then-removal with migration window. See goals/poc-done-flips.md.

- Breaking changes allowed. Single consumer (Asterisk).
- **Delete over deprecate.** No `// Deprecated:` markers or wrappers.
- No shim layers. Remove superseded types/functions; update consumer in same session.

## Scope

- Framework-only. Domain logic in consumers.
- Batteries-included extractors for JSON, YAML, structured text. Domain extractors built by consumers.

## Data handling

- External data: partial/absent. Return partial results with clear errors.
- Redact secrets and PII by default.
- External calls: timeouts + retry with exponential backoff.
