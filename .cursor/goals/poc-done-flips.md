# PoC Done — Policy Flips

When the "PoC Done" flag is raised, execute every flip in this checklist. Each item references the rule or contract that contains a time-bound PoC policy.

## Checklist

- [ ] **API stability (Origami)** — `rules/domain/project-standards.mdc` § "API stability"
  - **Current (PoC):** Breaking changes allowed. Delete over deprecate. No shim layers.
  - **Flip to:** Deprecate-then-remove with a migration window (minimum 1 minor version). Backward-compat wrappers are required for public API changes. Consumer migration guide accompanies every breaking change.

- [ ] **API stability (Asterisk)** — Asterisk `rules/domain/project-standards.mdc` § "API stability (Origami dependency)"
  - **Current (PoC):** Origami breaking changes are expected. Update immediately. No shims.
  - **Flip to:** Origami provides a migration window. Asterisk pins Origami at a tested version and upgrades deliberately, not reactively.

- [ ] **Contract rule** — `contracts/draft/ouroboros-seed-circuit.md` § "Contract rules"
  - **Current (PoC):** "Breaking API allowed" note.
  - **Flip to:** Remove the note. Standard API stability applies.

## When to raise the flag

The "PoC Done" flag is raised when **all** of the following are true:

1. Presentation delivered (CHECKPOINT E passes).
2. Asterisk BasicAdapter and CursorAdapter baselines documented.
3. Origami public API surface is reviewed and intentionally frozen.
4. Both repos have clean README positioning.

## Who raises the flag

The project owner (dpopsuev) explicitly declares "PoC Done" in a conversation or commit message. No agent auto-promotes.
