# De Bono × Origami — Design Lineage Case Study

Case study comparing Edward de Bono's Six Thinking Hats / Parallel Thinking
with Origami's persona system. Identifies structural parallels, philosophical
tensions, and design insights.

**Sources:** [Six Thinking Hats](https://en.wikipedia.org/wiki/Six_Thinking_Hats),
[Parallel Thinking](https://en.wikipedia.org/wiki/Edward_de_Bono#Parallel_thinking)

## Colored Roles for Structured Thinking

Both systems assign colored identities to constrain and direct cognitive effort:

| De Bono Hat         | Role                    | Origami Persona           | Role                               |
| ------------------- | ----------------------- | ------------------------- | ---------------------------------- |
| White (facts)       | Neutral information     | Seeker (Cerulean/Water)   | Deep evidence gatherer             |
| Red (intuition)     | Gut reaction, 30s only  | —                         | No direct equivalent               |
| Black (critical)    | Risks, what's wrong     | Challenger (Scarlet/Fire) | Rejects weak triage                |
| Yellow (optimistic) | Benefits, value         | Herald (Crimson/Fire)     | Fast, optimistic classification    |
| Green (creative)    | New ideas, alternatives | Weaver (Amber/Air)        | Cross-repo correlator, synthesizer |
| Blue (meta)         | Process management      | Ouroboros                 | Metacalibration                    |

Origami adds 4 Antithesis personas (Challenger, Abyss, Bulwark, Specter) that
have no de Bono counterpart — because de Bono explicitly rejects adversarial
thinking.

## Philosophical Tension: Parallel vs. Adversarial

De Bono's entire thesis is that adversarial debate (Socratic dialectic — what he
calls "the Greek Gang of Three") is harmful. Parallel Thinking means everyone
focuses in the same direction simultaneously. No one argues; everyone explores
together. He positions this explicitly against the dialectic method.

Origami does the opposite. The Adversarial Dialectic (D0-D4) is a structured
dialectic: Thesis holder frames a charge, Antithesis holder challenges, they
debate in rounds, a Synthesis verdict emerges.

However — Origami's dialectic is much closer to de Bono's controlled hat
sequences than to free-form debate:

- **Bounded:** `MaxTurns`, `MaxNegations`, `GapClosureThreshold` — debate cannot spiral
- **Structured:** D0→D1→D2→D3→D4, each step has a schema — not free-form arguing
- **Conditional:** `NeedsAntithesis()` only fires in the uncertain confidence band (0.50–0.85)
- **Synthesizing:** D4 Verdict produces Affirm/Amend/Acquit/Remand — it resolves, not just critiques

De Bono would likely approve of the bounded, structured nature but object to the
adversarial framing. A de Bono-aligned version would frame D0-D4 as "switching
hats" rather than "prosecution vs. defense."

## Design Insights

### Hat Discipline = Mask Discipline

De Bono's cardinal rule: everyone wears the same hat at the same time. You don't
have one person being emotional (Red) while another is being factual (White).

Origami's Mask system implements this principle at the node level:

- `ValidNodes()` constrains where a capability activates
- At any given node, all walkers experience the same injected context
- This is node-scoped hat enforcement — the circuit structure is the hat sequence

The Mask system is de Bono's "everyone wears the same hat" translated into
circuit middleware.

### The Missing Red Hat

De Bono's Red Hat: 30 seconds, no justification required, pure gut reaction. It
surfaces intuition that would otherwise be suppressed by the demand for evidence.

Origami has no equivalent. Confidence scores are computed, not intuitive. Every
conclusion requires evidence citations (`project-standards.mdc`: "Every AI
inference must cite evidence").

This is correct for RCA (evidence-first is the right design). But it raises a
question: is there value in a fast, cheap, zero-evidence "pre-filter" step? F0
Recall (Herald persona) is the closest — fast intake — but it still requires
structured output.

### Sequence Programs vs. Affinity Scheduling

De Bono prescribes fixed hat sequences for different activities:

- Problem solving: Blue → White → Green → Red → Yellow → Black → Green → Blue
- Strategic planning: Blue → Yellow → Black → White → Blue → Green → Blue

These are chosen upfront and followed rigidly.

Origami's `AffinityScheduler` is more adaptive: `StepAffinity` scores determine
which walker handles which node, `computeMismatch()` dynamically matches walker
capabilities to node needs, and edge heuristics determine the actual path at
runtime.

This is an evolution: de Bono's static programs → Origami's dynamic scheduling
with affinity scores. The circuit graph encodes the space of possible hat
sequences; the walker/scheduler chooses the actual sequence per case.

## Verdict

The Origami persona system is a computational descendant of de Bono's framework
with two key mutations:

1. **Embraces adversarial dialectic** (de Bono rejects it) — but bounds it so
   tightly it functions more like a structured hat-switch than a free debate
2. **Dynamic scheduling** replaces fixed sequences — the graph topology +
   affinity scores create emergent hat programs per case

The Mask system is the closest direct implementation of de Bono's "hat
discipline" principle.
