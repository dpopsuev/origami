# Case Study: Electronic Symbols & Circuit Diagrams — A Pictogram Language for Washi

**Date:** 2026-03-02
**Subject:** Electronic schematic symbols and circuit diagram conventions as a pattern for Washi's visual circuit editor.
**Source:** `en.wikipedia.org/wiki/Electronic_symbol`, `en.wikipedia.org/wiki/Circuit_diagram`, IEC 60617, IEEE 315
**Purpose:** Design a standardized pictogram vocabulary for Origami circuits rendered in Washi. Electronic schematics solved this problem 80+ years ago: hundreds of component types, one universal visual language that any engineer can read instantly. Origami's agentic circuits need the same.

**Prerequisite:** `electronic-circuit-theory.md` — establishes the component-level mapping (transistor = Node, op-amp = Dialectic, etc.). This study extends that mapping into the visual domain: if a transistor has a standardized symbol, what is the standardized symbol for a Node?

---

## 1. How Electronic Symbols Work

Electronic symbols are **pictograms** — small, iconic pictures that represent components in a schematic diagram. They are not photographs or detailed drawings. They are deliberately abstract: a resistor is a zig-zag (ANSI) or rectangle (IEC), not a picture of a physical resistor. The abstraction serves three purposes:

1. **Recognition speed.** An engineer scanning a schematic identifies components in milliseconds. The symbol is faster than reading a label.
2. **Composability.** Symbols combine on a canvas with wires (edges) between them. The schematic is a graph with symbols at the nodes and wires at the edges — exactly what Washi renders.
3. **Universality.** IEC 60617 standardized symbols internationally. An engineer in Tokyo reads a schematic drawn in Munich without translation.

### Symbol design principles (from IEC 60617 and IEEE 315)

| Principle | Description | Origami implication |
|-----------|-------------|---------------------|
| **Iconic, not literal** | A diode symbol (triangle + bar) abstracts the physical device into its functional essence: one-way flow. | Origami symbols should represent *what the component does*, not what it looks like in YAML. |
| **Distinguishable at small scale** | Symbols must be readable on dense schematics. A resistor and capacitor are visually distinct even at 10px height. | Washi symbols must work at semantic zoom levels — recognizable even when the graph has 50+ nodes. |
| **Composable with standard wires** | Every symbol has input/output terminals (pins) where wires attach. Terminal placement is standardized: inputs left, outputs right. | Origami symbols need consistent port placement for edge connections. |
| **Annotatable** | Reference designators (R1, C2, Q3) and values (10kΩ, 100nF) are placed beside symbols, not inside them. | Node names, transformer bindings, and element affinities annotate the symbol externally. |
| **Variant-aware** | A base symbol can have variants: a capacitor with a + mark is polarized; without is non-polarized. Same family, visual modifier. | A Node symbol with a "D" badge is deterministic; with an "S" badge is stochastic. Same base, visual modifier. |

### Organization conventions (from circuit diagram standards)

| Convention | Electronic | Washi equivalent |
|------------|-----------|-----------------|
| **Signal flow left-to-right** | Input at left edge, output at right edge of the schematic. | Circuit entry node at left, `_done` at right. |
| **Power at top, ground at bottom** | VCC rail at top, GND at bottom. | Input context / walker identity at top, `_done` sink at bottom (or right). |
| **Zones as dashed boxes** | Functional blocks enclosed in dashed rectangles with labels. | Zone subgraphs rendered as labeled regions with background color. |
| **T-junction for connections** | Wires meeting at a T with a dot indicate connection; crossing without a dot indicates no connection. | Edge connection points on node ports; edge crossings are layout artifacts, not junctions. |
| **Reference designators** | R1, C2, Q3 — type prefix + sequential number. | Node names serve as reference designators. Transformer bindings are the "value" annotation. |

---

## 2. The Origami Pictogram Vocabulary

Each Origami primitive maps to a pictogram. The design follows IEC principles: iconic, distinguishable at small scale, composable with edges, annotatable.

### Tier 1: Core Components (always visible)

These appear on every circuit. Their symbols must be the most recognizable.

| Origami Primitive | Electronic Analogue | Proposed Pictogram | Rationale |
|-------------------|--------------------|--------------------|-----------|
| **Node** (generic) | Transistor | Rectangle with rounded corners. Input port (left), output port (right). | The transistor is the fundamental active element. The rectangle is the simplest shape that accommodates a label. Rounded corners distinguish from zone boxes. |
| **Node** (deterministic) | Digital logic gate | Rectangle with a **gear icon** (⚙) badge in the top-right corner. | Gears = mechanical, predictable, deterministic. The badge is the variant marker (like the + on a polarized capacitor). |
| **Node** (stochastic) | Analog amplifier | Rectangle with a **sparkle icon** (✦) badge in the top-right corner. | Sparkle = non-deterministic, probabilistic, AI-powered. Visual contrast with the gear. |
| **Edge** (normal) | Wire/trace | Solid line with arrowhead. | Universal convention for directed flow. |
| **Edge** (shortcut) | Diode | Dashed line with arrowhead + small triangle on the line. | The diode allows flow in one direction above a threshold. The dashed line signals "conditional bypass." The triangle is the diode's iconic shape. |
| **Edge** (loop) | Feedback path | Curved line returning to an earlier node, with a circular arrow icon. | Feedback loops in schematics curve back. The circular arrow signals iteration. |
| **Zone** | Functional block (dashed box) | Rounded rectangle with semi-transparent background fill. Zone label at top. | Matches the standard schematic convention for functional grouping. Background color encodes data domain (see Pattern 2 from electronic-circuit-theory). |
| **`_start`** | Power supply (VCC) | Filled circle with a play icon (▶). | Power supply initiates the circuit. Play = "begin." |
| **`_done`** | Ground (GND) | Filled circle with a stop icon (■). | Ground is the universal sink. Stop = "terminate." |

### Tier 2: Processing Components (visible at medium zoom)

| Origami Primitive | Electronic Analogue | Proposed Pictogram | Rationale |
|-------------------|--------------------|--------------------|-----------|
| **Extractor** | ADC | Small downward arrow icon (↓) inside the node, or a badge: `[E]` with a funnel shape. | ADC converts analog to digital (unstructured to structured). The funnel represents narrowing free-form into typed schema. |
| **Transformer** | IC / functional module | Hexagonal badge overlaid on the node. Hexagon is the traditional IC package outline (DIP top-view). | Transformers are the "ICs" of Origami — self-contained processing units with defined inputs and outputs. |
| **Hook** (pre/post) | Inductor / ferrite bead | Small diamond (◆) on the edge entering or leaving a node. Pre-hook diamond on the input edge; post-hook diamond on the output edge. | Inductors and ferrites condition the signal at component boundaries. Hooks condition artifacts at node boundaries. The diamond is small enough to not clutter but visible enough to signal "something happens here." |
| **Mask** | Signal conditioning chain | Stacked diamonds (◆◆) on the edge — one per mask layer. | Multiple masks compose like a filter chain. Stacking communicates nesting depth. |
| **Walker** | Current source | Small person icon (stick figure) or element-colored dot that moves along edges during animation. | The walker is the "current" flowing through the circuit. A moving dot is the oscilloscope trace — it shows where the signal is now. |

### Tier 3: Advanced Components (visible at high zoom or on hover)

| Origami Primitive | Electronic Analogue | Proposed Pictogram | Rationale |
|-------------------|--------------------|--------------------|-----------|
| **Adversarial Dialectic** | Op-Amp | Triangle (the op-amp symbol) with `+` and `−` inputs. Thesis enters at `+`, antithesis enters at `−`, synthesis exits at the apex. | The op-amp case study in `electronic-circuit-theory.md` is the most detailed component mapping. The triangle is instantly recognizable to any engineer. |
| **Element affinity** | Component rating | Colored ring around the node, using element colors from ODS (`--el-fire`, `--el-water`, etc.). | Component ratings (voltage, wattage) are annotated on schematics. Element affinity is the node's "rating" — its behavioral spec. |
| **Persona** | Named IC package | Small avatar badge with element color. | ICs have part numbers (LM741, NE555). Personas have names and element types. The avatar distinguishes which "personality" is assigned. |
| **Subgraph / Marble** | IC with internal schematic | Collapsed: single node with a chevron fold indicator (▸). Expanded: reveals internal graph in a nested container. | ICs can be shown as a black box (pin diagram) or as a full internal schematic. Subgraph fold/unfold mirrors this exactly. |
| **Circuit Breaker** | Fuse / MCCB | Small lightning bolt icon (⚡) badge on the node. | Fuses and circuit breakers protect against overcurrent. The lightning bolt is universally associated with electrical protection. |
| **Context Filter** (zone boundary) | Decoupling capacitor | Two parallel lines (capacitor symbol) on the zone boundary edge. | The decoupling capacitor case study (Pattern 6) directly maps to context filtering at zone boundaries. The capacitor symbol on the boundary communicates "filtering happens here." |

---

## 3. The Origami Schematic Standard (OSS)

Following IEC 60617's structure, we propose a formalized symbol standard for Origami circuits:

### OSS-1: Symbol categories

| Category | Prefix | Components |
|----------|--------|------------|
| **Nodes** | N | Generic, deterministic, stochastic, dialectic, subgraph |
| **Edges** | E | Normal, shortcut, loop, zone-crossing |
| **Zones** | Z | Unstructured, structured, hybrid |
| **Walkers** | W | By element type (Fire, Water, Earth, Air, Void, Lightning) |
| **Processors** | P | Extractor, transformer, hook, mask, renderer |
| **Control** | C | Circuit breaker, rate limiter, context filter |
| **Terminals** | T | Start, done, error |

### OSS-2: Reference designators

Following the R1/C2/Q3 convention from electronics:

| Component | Designator format | Example |
|-----------|------------------|---------|
| Node | `N-<name>` | `N-triage`, `N-investigate` |
| Edge | `E-<from>-<to>` | `E-triage-investigate` |
| Zone | `Z-<name>` | `Z-backcourt`, `Z-frontcourt` |
| Walker | `W-<element><N>` | `W-Fire1`, `W-Water2` |
| Extractor | `X-<name>` | `X-json-triage` |
| Transformer | `TF-<name>` | `TF-llm-extract` |

### OSS-3: Layout rules

1. **Signal flows left-to-right.** Entry nodes at the left edge, terminal nodes at the right edge.
2. **Zones are horizontal bands.** Backcourt (unstructured) at top, frontcourt (structured) at bottom. Zone boundaries are horizontal dividers.
3. **Feedback loops curve below.** Loop edges route below the main signal path to avoid crossing forward edges.
4. **Deterministic nodes cluster together.** D-nodes (gear badge) should be adjacent when possible, creating a visual "digital section" of the circuit.
5. **Stochastic nodes cluster together.** S-nodes (sparkle badge) form the "analog section."
6. **The D/S boundary is visually prominent.** Where a deterministic edge connects to a stochastic node (or vice versa), the edge is rendered with a gradient or color transition. This is the "mixed-signal boundary" — the most critical design point.

---

## 4. Washi Integration: Component Palette

The pictogram vocabulary directly populates Washi's **Component Palette** (Phase 2, task B3). The palette is organized by OSS category:

```
┌─────────────────────────────────┐
│ Component Palette               │
├─────────────────────────────────┤
│ ▸ Nodes                         │
│   ☐ Generic Node                │
│   ⚙ Deterministic Node          │
│   ✦ Stochastic Node             │
│   △ Dialectic Node              │
│   ▸ Subgraph (collapsed)        │
│                                 │
│ ▸ Edges                         │
│   → Normal Edge                 │
│   ⇢ Shortcut Edge               │
│   ↻ Loop Edge                   │
│                                 │
│ ▸ Control                       │
│   ⚡ Circuit Breaker             │
│   ∥ Context Filter               │
│   ⏱ Rate Limiter                │
│                                 │
│ ▸ Zones                         │
│   □ Unstructured Zone            │
│   ■ Structured Zone              │
│   ◧ Hybrid Zone                  │
│                                 │
│ ▸ Terminals                     │
│   ▶ Start                       │
│   ■ Done                        │
│   ⚠ Error                       │
└─────────────────────────────────┘
```

### Adapters and Modules extend the palette

When an Origami adapter or module is installed, its transformers and extractors appear as additional palette entries under a namespace section:

```
│ ▸ origami.modules.rca            │
│   ⬡ llm-extract                  │
│   ⬡ context-builder              │
│   ⬡ persist                      │
│   ⬡ score                        │
│   ⬡ report                       │
│ ▸ origami.adapters.rp            │
│   ⬡ rp-fetch                     │
│   ⬡ rp-push                      │
```

The hexagon (⬡) is the transformer/extractor icon — the IC package shape from Tier 2.

---

## 5. Semantic Zoom and Symbol Detail Levels

Electronic schematics have one zoom level. Washi has semantic zoom (Phase 2.7, DO10). The pictogram vocabulary adapts to zoom level:

### Low zoom (overview) — "block diagram"

Only Tier 1 symbols visible. Nodes are small colored rectangles with D/S badges. Edges are thin lines. Zones are background regions. No labels except zone names.

This is the equivalent of a **block diagram** in electronics — shows functional blocks and signal flow, no component details.

```
┌─ Z-backcourt ────────────┐    ┌─ Z-frontcourt ──────────┐
│  [⚙]──[✦]──[✦]──[⚙]     │───▸│  [⚙]──[✦]──[⚙]         │──▸ ■
│                           │    │                          │
└───────────────────────────┘    └──────────────────────────┘
```

### Medium zoom (working view) — "schematic"

Tier 1 + Tier 2 symbols visible. Node names appear inside. Transformer hexagons and hook diamonds visible. Edge labels show condition summaries. Walker dots animate during runs.

This is the **schematic diagram** — the working engineer's view.

```
┌─ Backcourt ────────────────────────┐    ┌─ Frontcourt ──────────────┐
│  ⚙ recall ──▸ ✦ triage ──▸ ✦ inv  │───▸│  ⚙ correlate ──▸ ✦ judge │──▸ ■
│  ◆pre       ⬡llm-extract   ⬡llm   │  ∥ │  ⬡match          △dialectic│
│                              ◆post │    │                            │
└────────────────────────────────────┘    └────────────────────────────┘
```

### High zoom (inspection view) — "datasheet"

All three tiers visible. Full node configuration, element affinity rings, persona avatars, extractor schemas, recent artifact previews. Equivalent to reading the **component datasheet**.

---

## 6. The D/S Boundary as Mixed-Signal Boundary

The `electronic-circuit-theory.md` case study established that Origami circuits are mixed-signal: unstructured (analog) and structured (digital) domains connected by extractors (ADCs) and renderers (DACs). The D/S (Deterministic/Stochastic) boundary adds a second axis:

| | Deterministic | Stochastic |
|---|---|---|
| **Structured** | Pure digital logic (match rules, dedup, schema validation) | AI with typed output (LLM extraction, scored classification) |
| **Unstructured** | Template rendering, regex parsing | Free-form LLM generation, narrative synthesis |

The **most critical boundary** in the schematic is where all four quadrants meet — a node that converts unstructured stochastic input into structured deterministic output. This is the "ADC in a mixed-signal circuit" — and Washi should render it with maximum visual prominence:

- The edge crossing from S-domain to D-domain gets a **gradient transition** (amber to green, matching the zone colors from Phase 4 of origami-autodoc).
- The node at the boundary gets a **double badge**: gear (deterministic) + funnel (extractor).
- At medium zoom, a small label appears: "D/S boundary."

---

## 7. Comparison: EDA Tools vs Washi

Electronic Design Automation (EDA) tools — KiCad, Altium, Eagle, LTspice — are the precedent for schematic editors. Washi is an EDA for agentic circuits.

| EDA Feature | Electronic EDA | Washi |
|-------------|---------------|-------|
| Component library | Standard symbol libraries (IEC, ANSI) | OSS pictogram vocabulary + adapter/module palette |
| Schematic capture | Drag components, draw wires, annotate | Drag nodes, draw edges, configure conditions |
| Netlist generation | Schematic → netlist (connections list) | Graph → YAML (bidirectional sync, Phase 2 B2) |
| Simulation | SPICE simulation (voltage, current, timing) | Stub/dry calibration (artifact quality, confidence, path) |
| PCB layout | Netlist → physical board layout | YAML → `origami fold` → binary |
| Design Rule Check (DRC) | Verify electrical rules (clearances, connections) | `origami lint` + `origami validate` (schema, topology, D/S rules) |
| Bill of Materials (BOM) | List of all components with specs | Node catalog with transformer bindings, element affinities, cost estimates |
| Hierarchical design | Subsheets for complex circuits | Subgraph fold/unfold (Phase 2.5) |
| Collaborative editing | Multi-user schematic editing | Collaboration cursors (Phase 4, E8) |
| Version control | Schematic diffing | Circuit diff view (Phase 2.7, DO3) |

The parallel is structural, not superficial. Washi is doing for agentic circuits what KiCad does for electronic circuits: providing a visual design surface with standardized symbols, bidirectional synchronization with the textual format, simulation, validation, and hierarchical composition.

---

## 8. Actionable Takeaways

1. **Define the Origami Symbol Standard (OSS) in Washi Phase 0.** Before rendering any graph, define the pictogram vocabulary as SVG assets in the ODS package. Symbols are design tokens — they must be consistent across Washi and Kabuki.

2. **Three-tier semantic zoom vocabulary.** Low zoom = block diagram (shapes + colors). Medium zoom = schematic (symbols + labels + badges). High zoom = datasheet (full configuration). Each tier is a progressive disclosure of the same underlying graph.

3. **D/S badges on every node.** The gear (⚙ deterministic) and sparkle (✦ stochastic) badges are the most important visual signals in the schematic. They answer the question "does this node cost money?" at a glance.

4. **D/S boundary rendering.** The mixed-signal boundary (where deterministic edges meet stochastic nodes) must be visually prominent: gradient edges, double badges, boundary labels at medium zoom. This is the most critical design point in any circuit.

5. **Adapter/module palette entries use the hexagon.** The hexagon is the IC package shape — it communicates "self-contained processing unit with defined I/O." All transformer and extractor entries in the palette use the hexagon, regardless of origin.

6. **Dialectic node uses the op-amp triangle.** The op-amp is the most detailed analogy in the electronic-circuit-theory case study. Rendering the dialectic as a triangle with +/− inputs is instantly meaningful to anyone who has read that study.

7. **Context filter uses the capacitor symbol.** Two parallel lines on a zone boundary edge communicate "filtering happens here" — directly from the decoupling capacitor pattern (Pattern 6).

8. **Washi's Component Palette is a "component library."** Organize it by OSS category (Nodes, Edges, Zones, Control, Terminals), with adapter/module namespaces as expandable sections. This mirrors how KiCad organizes its symbol libraries.

9. **Reference designators in the YAML.** Consider generating OSS-style reference designators (`N-triage`, `E-triage-investigate`, `Z-backcourt`) as metadata in `origami autodoc` output. These appear as labels in the schematic view and in generated documentation.

10. **Theme system (Phase 5.5, PP1) extends the symbol vocabulary.** Domain-specific themes can override default pictograms. A security-domain theme (Achilles) might use shield icons instead of gears for deterministic nodes. The base vocabulary is standardized; themes customize it.

---

## 9. Washi as an Agentic Schematic Editor

The synthesis of this case study and `electronic-circuit-theory.md`:

> **Washi is not a graph editor. Washi is a schematic editor for agentic circuits.**

The distinction matters:
- A **graph editor** renders nodes and edges. Any shape, any layout, no semantic meaning in the visuals.
- A **schematic editor** renders components using standardized symbols with semantic meaning baked into the visual representation. The shape tells you what the component does before you read any label.

When an engineer opens KiCad, they don't see "rectangles connected by lines." They see resistors, capacitors, op-amps, and transistors — each instantly recognizable from its pictogram. When a circuit designer opens Washi, they should not see "boxes connected by arrows." They should see deterministic nodes (gear), stochastic nodes (sparkle), dialectic engines (triangle), shortcut edges (dashed + triangle), context filters (capacitor), and circuit breakers (lightning bolt) — each instantly recognizable from its pictogram.

The Origami Symbol Standard (OSS) is the visual language that makes this possible. It transforms Washi from a generic graph tool into a domain-specific schematic editor — the KiCad of agentic circuits.

---

## References

- Electronic symbol standards: `en.wikipedia.org/wiki/Electronic_symbol`
- Circuit diagram conventions: `en.wikipedia.org/wiki/Circuit_diagram`
- IEC 60617:2025 — International standard for electronic symbols
- IEEE 315-1975 — ANSI standard for graphic symbols in electrical/electronics diagrams
- Prerequisite case study: `electronic-circuit-theory.md` — component-level mapping
- Washi contract: `contracts/draft/washi.md` — full product specification
- Origami Design System: referenced in Washi Phase 0a (ODS tokens)
- EDA tools: KiCad (`kicad.org`), Altium Designer, LTspice
- Schematic capture: `en.wikipedia.org/wiki/Schematic_capture`
