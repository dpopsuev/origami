# Case Study: Electronic Circuit Theory — Signal Processing as Pipeline Orchestration

**Date:** 2026-03-01
**Subject:** Electronic circuit theory — analog, digital, and mixed-signal design principles
**Source:** `en.wikipedia.org/wiki/Electronic_circuit`, classical EE textbooks (Jaeger, Horowitz & Hill, Sedra & Smith)
**Purpose:** Cross-domain pattern study. Electronic circuits and agentic pipelines solve the same fundamental problem: transforming, routing, and conditioning signals through a graph of processing elements. Map circuit theory onto Origami's architecture. Identify patterns the framework could formalize or adapt. The central analogy — analog-to-digital conversion mirrors unstructured-to-structured extraction — opens into a deeper structural isomorphism.

---

## 1. What Electronic Circuits Are

An electronic circuit is a graph of components connected by conductive paths through which electric current flows. Components transform signals: amplify, filter, modulate, convert, store, route. The graph topology determines what the circuit does.

Circuits fall into three categories:

**Analog circuits** process continuous signals — voltage and current vary smoothly over time. Components: resistors (limit current), capacitors (store charge), inductors (resist change), transistors (amplify/switch), diodes (one-way flow). Design concerns: gain, bandwidth, noise, impedance matching. Analysis tools: Kirchhoff's laws (conservation of current and voltage), Ohm's law, transfer functions.

**Digital circuits** process discrete signals — voltages represent binary 0 or 1. Built from transistors wired as logic gates (AND, OR, NOT, XOR). Logic gates compose into arbitrarily complex computational functions. Design concerns: timing, propagation delay, power dissipation, race conditions. The key property: each gate **regenerates** the binary signal, so noise accumulated in one stage does not propagate to the next.

**Mixed-signal circuits** contain both analog and digital sections connected by converters. An **ADC** (analog-to-digital converter) samples a continuous signal and quantizes it into discrete values. A **DAC** (digital-to-analog converter) reconstructs a continuous signal from discrete values. Most real-world systems are mixed-signal: sensors produce analog data, processors work digitally, actuators need analog output.

---

## 2. The Core Analogy: ADC/DAC as Extraction/Rendering

The deepest structural parallel between circuit theory and Origami is at the analog-digital boundary.

### The ADC side: Extractor

An ADC takes a continuous, noisy, information-rich analog signal and converts it into a discrete, clean, machine-processable digital representation. An Origami `Extractor` does the same: it takes unstructured LLM output (natural language, free-form JSON, log fragments) and converts it into a typed Go struct implementing `Artifact`.

The ADC parameters map precisely:

| ADC Parameter | Origami Equivalent | Implication |
|---|---|---|
| **Sampling rate** (samples/sec) | Extraction attempts / retries | Too few samples = missed information. Too many = wasted compute. The Nyquist rate sets the minimum: sample at least 2x the highest-frequency component to avoid aliasing. In pipeline terms: the extraction schema must capture at least the essential structure of the LLM output, or the result is a distorted representation. |
| **Resolution** (bit depth) | Schema granularity | 8-bit ADC captures 256 levels; 16-bit captures 65,536. A coarse schema (`{category: string, confidence: float}`) is 8-bit. A fine schema (`{category, subcategory, evidence[], confidence, reasoning, alternatives[], caveats[]}`) is 16-bit. More resolution = more faithful representation = more downstream utility. |
| **Quantization error** | Information loss during extraction | The irreducible gap between the continuous input and its discrete representation. A 3-paragraph LLM analysis reduced to `{confidence: 0.72}` has high quantization error. The `Raw()` method on `Artifact` is Origami's way of preserving the original signal alongside the quantized version — analogous to storing both the digital samples and the original analog waveform. |
| **Anti-aliasing filter** | Prompt engineering / persona preamble | Before an ADC samples, an anti-aliasing low-pass filter removes frequencies above Nyquist to prevent aliasing (high-frequency content masquerading as low-frequency). In pipelines, the prompt preamble and persona instructions shape the LLM output *before* extraction, ensuring it falls within the extractor's "bandwidth." A poorly prompted LLM produces output the extractor cannot faithfully represent — aliasing. |
| **Oversampling** | Multiple extraction with voting | Some ADCs sample at many times the Nyquist rate, then downsample with digital filtering for better effective resolution. A pipeline could extract multiple times from the same LLM output using different schema projections, then merge — trading compute for fidelity. |

### The DAC side: Prompt Renderer

A DAC converts discrete digital values back into a continuous analog signal. In Origami, `RenderPrompt` converts structured context (variables, prior outputs, walker state) back into natural language prompts that LLMs consume.

| DAC Parameter | Origami Equivalent | Implication |
|---|---|---|
| **Reconstruction filter** | Prompt template smoothing | Raw DAC output is a staircase waveform (discrete steps). A reconstruction filter interpolates between steps to produce a smooth signal. A raw template (`Category: {{.category}}, Confidence: {{.confidence}}`) is a staircase. A well-crafted prompt that weaves structured data into natural narrative is a smooth reconstruction. |
| **Dynamic range** | Prompt expressiveness | The range of output voltages a DAC can produce. A rigid template has low dynamic range. A template system with conditionals, loops, and context-aware sections has high dynamic range — it can express a wider variety of structured inputs as coherent prompts. |
| **Glitch energy** | Prompt artifacts | When a DAC transitions between values, brief voltage spikes (glitches) appear. In prompts, poorly interpolated structured data creates artifacts: dangling references, contradictory instructions, formatting breaks. A deglitcher (sample-and-hold) in circuits; prompt validation in pipelines. |

### The symmetry insight

In circuit design, ADC and DAC are treated as **symmetric, equally important** conversions. Neither is an afterthought. They have dedicated components, specifications, and design attention.

In Origami today, the `Extractor` interface is first-class: named, registered, DSL-wirable, with built-in implementations (`JSONExtractor[T]`, `RegexExtractor`, `CodeBlockExtractor`). The prompt renderer (`RenderPrompt`) is a utility function — unregistered, not DSL-wirable, no named implementations.

Circuit theory says this asymmetry is a design smell. The structured-to-unstructured conversion (DAC) deserves the same architectural weight as the unstructured-to-structured conversion (ADC). A `Renderer` interface symmetric to `Extractor` — with named implementations, a registry, and DSL integration — would close this gap.

```mermaid
flowchart LR
    subgraph unstructuredIn ["Unstructured"]
        raw["LLM Output / Logs / Free Text"]
    end

    raw --> ADC["Extractor (ADC)"]
    ADC --> structured

    subgraph structuredDomain ["Structured"]
        structured["Typed Artifact / Go Struct"]
    end

    structured --> DAC["Renderer (DAC)"]
    DAC --> prompt

    subgraph unstructuredOut ["Unstructured"]
        prompt["Prompt for Next Node"]
    end
```

The cycle is continuous: each node receives unstructured input (from an LLM or prior rendering), extracts structure (ADC), processes the structured data, then renders it back to unstructured form (DAC) for the next LLM call. The quality of both conversions determines the pipeline's overall signal fidelity.

---

## 3. Concept Mapping: Circuit Theory to Origami

| Circuit Concept | Origami Equivalent | Mapping Rationale |
|---|---|---|
| **Transistor** (active element) | `Node` | The fundamental active processing unit. Transistors amplify or switch signals; nodes process artifacts. Both are the building blocks everything else composes from. |
| **Wire / trace** | `Edge` | Passive signal path between active elements. Wires carry current; edges carry artifacts and transitions. Both are defined by their endpoints and their properties (impedance / conditions). |
| **Resistor** (limits current) | Edge `when:` condition | Controls and limits signal flow. A resistor drops voltage proportional to current; a `when:` expression gates transitions proportional to artifact properties. Both prevent uncontrolled flow. |
| **Capacitor** (stores charge) | `WalkerState.Context` | Accumulates energy (charge / context) over time and releases it when needed. A capacitor integrates current into voltage; walker context integrates node outputs into accumulated state. Both have memory — they remember what happened before. |
| **Inductor** (resists change) | `Mask` (pre/post hooks) | Opposes sudden changes in current / processing behavior. An inductor smooths transients; a mask's pre-hook normalizes input and post-hook validates output, smoothing the signal around the node. Both add stability at the cost of latency. |
| **Diode** (one-way valve) | Shortcut edge | Allows current in one direction only, with a threshold voltage. A shortcut edge allows traversal only when confidence exceeds a threshold. Both implement conditional, one-directional flow. |
| **Op-amp** (differential amplifier) | Adversarial Dialectic | Takes two inputs (inverting and non-inverting), amplifies the difference. The Dialectic takes thesis and antithesis, amplifies their disagreement into a resolved synthesis. Both produce a single output from two competing inputs. High open-loop gain (unchecked dialectic) causes saturation; negative feedback (convergence criteria) stabilizes the output. |
| **Ground** (reference / sink) | `_done` node | The universal reference point and signal sink. All circuits reference to ground; all pipeline walks terminate at `_done`. |
| **Power supply** (VCC/VDD) | Input context + Walker identity | The energy source that powers every component. Without VCC, nothing operates. Without input context and a walker, no node can process. |
| **Bus** (data + address + control) | Papercup signal bus | Shared communication channel with structured protocol. A data bus carries payloads (artifacts), an address bus carries routing (dispatch IDs), a control bus carries status signals (waiting/processing/done). Papercup's three-part protocol mirrors this exactly. |
| **Clock signal** | Scheduler tick / dispatch cycle | Synchronization pulse for sequential operations. Digital circuits advance on clock edges; the dispatcher advances on poll cycles. Both ensure orderly, synchronized progression. |
| **Test point** | `WalkObserver` | Designated measurement insertion point. A test point lets an oscilloscope probe the signal without affecting it; `WalkObserver` lets Kami observe walk events without affecting execution. Both are designed-in observability. |
| **PCB / schematic** | Pipeline YAML (DSL) | Declarative design artifact describing the circuit topology. A schematic shows components and connections; pipeline YAML shows nodes and edges. Both are the source of truth that gets "compiled" into an executable form (fabricated PCB / `BuildGraph`). |
| **Breadboard** | Stub calibration | Rapid prototyping platform. A breadboard lets you wire components without soldering for quick verification; stub calibration lets you verify pipeline machinery without LLM calls. Both validate structure before committing to production. |
| **Production PCB** | `origami fold` | The final, optimized, manufactured form. A PCB is the breadboard prototype compiled into a production artifact. `origami fold` is the YAML pipeline compiled into a standalone binary. |
| **Component library** | Registries (`NodeRegistry`, `ExtractorRegistry`, `TransformerRegistry`) | Catalog of reusable, characterized parts. Component libraries specify every part's parameters; registries map names to implementations. Both enable design by composition from known building blocks. |

---

## Visual Gallery: Component Parallels

Each core circuit component has a structural counterpart in the Origami framework. The following diagrams place them side by side.

### Transistor and Node

The transistor is the fundamental active element in electronics — everything else is built from it. The Node is the fundamental active element in Origami. Both take an input signal, apply a controlled transformation (set by a bias/affinity), and produce an amplified or processed output.

```mermaid
flowchart LR
    subgraph transistor ["Transistor"]
        vin["Vin"] --> tr["Transistor"]
        gate["Gate / Base"] -.->|"bias"| tr
        tr --> vout["Vout (amplified)"]
    end

    subgraph nodeOrigami ["Node"]
        artIn["Input Artifact"] --> nd["Node.Process"]
        elem["ElementAffinity"] -.->|"bias"| nd
        nd --> artOut["Output Artifact"]
    end
```

A transistor without a gate bias is uncontrolled. A node without an element affinity has no behavioral contract. Both need a control signal to produce useful output.

### Capacitor and WalkerState.Context

A capacitor accumulates charge over time — voltage builds as current flows in. `WalkerState.Context` accumulates state across nodes — context builds as each node adds its output. Both are the memory of the system.

```mermaid
flowchart TB
    subgraph capacitor ["Capacitor"]
        direction LR
        it1["I at t1"] --> cap["C"]
        it2["I at t2"] --> cap
        it3["I at t3"] --> cap
        cap --> voltage["V = Q/C"]
    end

    subgraph context ["WalkerState.Context"]
        direction LR
        n1out["Node 1 output"] --> ctx["Context map"]
        n2out["Node 2 output"] --> ctx
        n3out["Node 3 output"] --> ctx
        ctx --> downstream["Available downstream"]
    end
```

A capacitor that never discharges bloats the circuit. A context that grows without bound bloats the prompt window. Both need periodic discharge — in circuits via a bleed resistor, in pipelines via context filtering at zone boundaries (Pattern 6).

### Diode and Shortcut Edge

A diode allows current in one direction only, above a threshold voltage. A shortcut edge allows traversal only when a condition is met, skipping intermediate nodes. Both implement conditional, one-directional flow.

```mermaid
flowchart LR
    subgraph diode ["Diode"]
        srcA["Source"] -->|"V >= 0.7V"| d["Diode"] --> destA["Destination"]
        srcA -.->|"V < 0.7V: blocked"| normalA["Normal path"]
    end

    subgraph shortcut ["Shortcut Edge"]
        classify["classify"] -->|"confidence >= 0.8"| s["Shortcut Edge"] --> decide["decide"]
        classify -.->|"confidence < 0.8"| investigate["investigate"]
    end
```

The diode's forward voltage (0.7V) is the shortcut's confidence threshold (0.8). Below it, signal takes the long path. Above it, signal bypasses intermediate stages. Both trade thoroughness for speed when the signal is strong enough.

### Bus and Papercup Signal Bus

A system bus has three channels: data (payload), address (routing), control (status/commands). Papercup's signal protocol mirrors this exactly with three corresponding channels.

```mermaid
flowchart TB
    subgraph systemBus ["System Bus"]
        direction LR
        dataBus["Data Bus"]
        addrBus["Address Bus"]
        ctrlBus["Control Bus"]
    end

    dataBus <-->|"payload"| artifactCh
    addrBus <-->|"routing"| dispatchID
    ctrlBus <-->|"status"| statusSig

    subgraph papercup ["Papercup Signal Bus"]
        direction LR
        artifactCh["Artifact Channel"]
        dispatchID["Dispatch ID"]
        statusSig["Status Signal"]
    end
```

In circuits, a device that writes to the data bus without a valid address corrupts memory. In Papercup, an artifact submitted without a matching `dispatch_id` is a race condition. Both protocols require all three channels to be coherent for correct operation.

### Op-Amp and Adversarial Dialectic

The op-amp is the most instructive single-component analogy in this study. A real op-amp (e.g., the classic uA741) has open-loop gain of ~100,000, near-infinite input impedance, near-zero output impedance, and a fixed gain-bandwidth product. These characteristics make it useless in open-loop mode (it saturates instantly) but extraordinarily useful in closed-loop mode, where external feedback components determine all useful behavior. The Adversarial Dialectic follows the same architecture.

#### Open-loop vs Closed-loop

An op-amp without feedback is a **comparator** — it produces a binary output (HIGH or LOW) because any nonzero input difference, multiplied by 100,000, immediately saturates. With negative feedback, the same op-amp becomes a **linear amplifier** with predictable, controlled gain.

```mermaid
flowchart LR
    subgraph openLoop ["Open-Loop: Comparator"]
        inOL["Signal"] --> ampOL["Op-Amp (A=100k)"] --> outOL["Saturated: HIGH or LOW"]
    end

    subgraph closedLoop ["Closed-Loop: Linear Amplifier"]
        inCL["Signal"] --> ampCL["Op-Amp (A=100k)"] --> outCL["Proportional Output"]
        outCL -->|"Rf / Rg feedback"| ampCL
    end
```

```mermaid
flowchart LR
    subgraph noFeedback ["No Feedback: Snap Judgment"]
        thesis1["Thesis"] --> dial1["Dialectic"] --> verdict1["Binary: Affirm or Acquit"]
    end

    subgraph withFeedback ["With Feedback: Calibrated Analysis"]
        thesis2["Thesis"] --> dial2["Dialectic"] --> verdict2["Calibrated Verdict"]
        verdict2 -->|"MaxTurns / Threshold"| dial2
    end
```

Without convergence criteria, the Dialectic is a comparator: whichever argument is slightly stronger wins instantly with maximum confidence. With feedback (convergence threshold, max turns), the Dialectic produces proportional, calibrated output where confidence reflects the actual weight of evidence.

#### Non-Inverting Amplifier Topology

The classic non-inverting amplifier is the op-amp's most common configuration. Two resistors (Rf and Rg) form a voltage divider in the feedback path. The closed-loop gain is entirely determined by these passive components: `Gain = 1 + Rf/Rg`. The op-amp's own gain (100,000) becomes irrelevant — only the feedback network matters.

```mermaid
flowchart TB
    signal["+V Signal"] --> opamp["Op-Amp"]
    opamp --> vout["Vout"]
    vout -->|"Rf"| divider["Voltage Divider"]
    divider -->|"to inverting input"| opamp
    divider -->|"Rg"| gnd["GND"]
```

```mermaid
flowchart TB
    thesis["+ Thesis"] --> engine["Dialectic Engine"]
    engine --> synth["Synthesis"]
    synth -->|"MaxTurns (Rf)"| check["Convergence Check"]
    check -->|"to antithesis input"| engine
    check -->|"Threshold (Rg)"| baseline["Baseline"]
```

The mapping: **Rf = MaxTurns** (how many debate rounds feed back) and **Rg = ConvergenceThreshold** (the reference level against which convergence is measured). Just as `Gain = 1 + Rf/Rg`, the dialectic's effective "amplification" of evidence quality scales with more rounds and a lower threshold. And just as in circuit design, **the useful behavior is set by the feedback parameters, not by the raw LLM capability** — an LLM with higher raw ability (higher open-loop gain) produces no better output if the feedback network is identical.

#### The Two Golden Rules

The ideal op-amp obeys two golden rules (Horowitz & Hill, *The Art of Electronics*):

1. **In negative feedback, the output does whatever is necessary to make the voltage difference between the inputs zero.**
2. **The inputs draw zero current.**

These translate directly to Dialectic design rules:

1. **The synthesis does whatever is necessary to make the disagreement between thesis and antithesis zero.** A synthesis that leaves unresolved contradictions between Thesis and Antithesis is like an op-amp that hasn't settled — it's still in transient, not at equilibrium. The convergence check should verify that the gap has closed, not just that N rounds have passed.

2. **The dialectic draws zero evidence.** The debate process should not consume, alter, or destroy the original evidence. Thesis and antithesis observe the same evidence; they interpret it differently. This is the "high input impedance" principle: the dialectic probes the evidence without loading it (changing it). If the dialectic process itself corrupts or selectively omits evidence, input impedance is too low.

#### Extended Parallel

| Op-Amp Characteristic | Dialectic Equivalent | Design Implication |
|---|---|---|
| Non-inverting input (+V) | Thesis Path | The primary signal to be processed |
| Inverting input (-V) | Antithesis Path | The challenging signal fed back from output |
| Differential stage | D0-D3: structured debate | Amplifies the disagreement between inputs |
| Output (Vout) | D4 Synthesis verdict | Single reconciled output |
| Feedback network (Rf, Rg) | MaxTurns, ConvergenceThreshold | Passive components that determine all useful behavior |
| Open-loop gain (100k) | Raw LLM capability | Enormous but useless without feedback; irrelevant to closed-loop behavior |
| Closed-loop gain (1 + Rf/Rg) | Effective quality amplification | Entirely determined by feedback parameters, not raw capability |
| **Golden Rule 1**: Vdiff -> 0 | Synthesis closes the thesis-antithesis gap | Convergence should verify gap closure, not just round count |
| **Golden Rule 2**: zero input current | Dialectic does not consume evidence | Evidence is observed, never altered by the debate process |
| **CMRR** (common-mode rejection) | Reject shared assumptions | The dialectic should amplify *disagreement*, not shared biases. If both thesis and antithesis share a faulty assumption, the synthesis inherits it. High CMRR = the dialectic surfaces and challenges shared premises |
| **GBWP** (gain-bandwidth product) | Quality-speed product is constant | More calibrated confidence (gain) requires more debate rounds (time). Faster convergence produces less calibrated output. The product is fixed for a given model |
| **Compensation** (dominant pole) | MaxNegations limit | Op-amps add a dominant pole capacitor to prevent high-frequency oscillation. MaxNegations prevents the dialectic from oscillating between thesis and antithesis indefinitely. Both sacrifice bandwidth for stability |
| **Slew rate** (max dV/dt) | Complexity processing limit | An op-amp distorts signals that change faster than its slew rate. An LLM distorts analysis when input complexity exceeds its processing capacity. Both produce triangle waves (oversimplified output) instead of faithful reproduction |
| **Input offset voltage** | Persona bias | A real op-amp has a small inherent bias that shifts the output. Each persona has an inherent element bias. Both are measurable, both can be compensated (Ouroboros measures persona bias; op-amps have offset null pins) |
| **Saturation** (output rails) | Confidence floor/ceiling | Output cannot exceed supply rails (0.0 to 1.0 confidence). A dialectic that always produces 0.95 or 0.15 is railing — the feedback isn't working |
| **Phase reversal** (input overdrive) | Confirmation bias lock-in | When an op-amp input is overdriven beyond the common-mode range, positive feedback can lock the output in the wrong state. When a thesis is overwhelmingly strong, the dialectic can lock into confirmation bias — the antithesis cannot overcome the thesis regardless of evidence |
| **Noise** (thermal, flicker) | Hallucination, prompt sensitivity | Intrinsic output noise even with zero input. LLMs hallucinate even with clean prompts. Both require low-noise designs for high-precision work |
| **PSRR** (power supply rejection) | Prompt template stability | A good op-amp rejects power supply noise. A good dialectic rejects variations in prompt formatting, template wording, and irrelevant context changes |

#### Design Improvements from Op-Amp Theory

**1. Convergence verification should check gap closure, not just round count.** Golden Rule 1 says the output settles when `V+ - V- = 0`. Currently, `MaxTurns` limits rounds but doesn't verify that the thesis-antithesis gap actually closed. A dialectic that runs 3 rounds but leaves major contradictions unresolved is like an op-amp that hasn't settled. The convergence check should measure the remaining disagreement between the last thesis and antithesis — not just whether the budget is exhausted.

**2. Evidence immutability during dialectic.** Golden Rule 2 says inputs draw zero current. The dialectic should guarantee that the original evidence (walker context, prior artifacts) is read-only during D0-D4. If the debate process modifies, filters, or selectively presents evidence to thesis or antithesis holders, input impedance is compromised and the output is biased by the process, not just the evidence.

**3. Shared-assumption detection (CMRR).** The most dangerous failure mode in a dialectic isn't that thesis and antithesis disagree — it's that they agree on something wrong. High CMRR means the system detects and challenges premises shared by both sides. A CMRR check in the dialectic would explicitly ask: "What assumptions do thesis and antithesis share? Are any of them unwarranted?" This is the one place where agreement should raise suspicion, not confidence.

**4. Quality-speed product as a model constant.** GBWP is fixed for a given op-amp. For a given model, the product of confidence calibration and convergence speed may be approximately constant. Ouroboros could measure this empirically: run the same dialectic at different MaxTurns values and plot confidence accuracy vs. rounds. The resulting curve characterizes the model's GBWP equivalent — pipeline designers can then choose the right operating point on the curve for their latency/quality tradeoff.

**5. Compensation for oscillation prevention.** MaxNegations is the dominant pole capacitor. Without it, a dialectic between two strong personas can oscillate: thesis refuted, antithesis refuted, thesis reinstated, antithesis reinstated. MaxNegations breaks this oscillation by forcing a decision after N rejections. The value should be tuned per element pair: Water vs Fire (high conflict = needs more compensation) vs Earth vs Air (low conflict = less compensation needed).

**6. Offset calibration via Ouroboros.** Real op-amps have offset null pins to trim inherent bias. Ouroboros already measures per-model behavioral dimensions — this is offset measurement. The next step is offset *compensation*: when a persona's measured bias is known, the prompt preamble can include a corrective instruction ("you tend toward over-confidence in classification; weight counter-evidence 10% more heavily"). This is the dialectic equivalent of trimming the offset null potentiometer.

---

## 4. Six Transferable Patterns

### Pattern 1: Signal Conditioning Chain

**Circuit principle:** Raw analog signals are never fed directly into an ADC. A signal conditioning chain — filter (remove noise), amplify (boost weak signals), level-shift (match voltage range) — prepares the signal for faithful conversion.

**Origami mapping:** The Mask pipeline (`MaskA.pre -> MaskB.pre -> Node.Process -> MaskB.post -> MaskA.post`) already implements signal conditioning. Masks before an extraction node are **anti-aliasing filters**: they shape the input to fall within the extractor's representable range.

**Insight:** This vocabulary helps pipeline designers reason about *why* certain masks exist. A `RecallMask` on an investigation node isn't just "adding context" — it's **amplifying a weak signal** so the extractor downstream can resolve it. A `CorrelationMask` isn't just "cross-referencing" — it's **filtering noise** by removing uncorrelated evidence. The signal conditioning metaphor makes mask placement a principled design decision rather than ad-hoc attachment.

**Possible adaptation:** Document mask placement guidelines using signal conditioning vocabulary. A pipeline design checklist: "Before every extraction boundary, verify the signal conditioning chain: noise filtered? signal amplified? level-shifted to match schema range?"

```mermaid
flowchart LR
    subgraph circuit ["Analog Signal Conditioning"]
        sensor["Sensor"] --> lpf["Filter"] --> amp["Amplifier"] --> ls["Level Shift"] --> adc["ADC"] --> digital["Digital Out"]
    end

    subgraph origami ["Mask Pipeline"]
        llmOut["LLM Output"] --> m1["CorrelationMask .pre"] --> m2["RecallMask .pre"] --> proc["Node.Process"] --> m2post["RecallMask .post"] --> m1post["CorrelationMask .post"] --> ext["Extractor"] --> typed["Typed Artifact"]
    end
```

### Pattern 2: Mixed-Signal Architecture

**Circuit principle:** Real-world systems are almost never pure analog or pure digital. They are **mixed-signal**: analog sections for interfacing with the physical world, digital sections for computation, and converters (ADC/DAC) at the boundaries. Each domain has different design rules. Analog design cares about noise, bandwidth, impedance. Digital design cares about timing, logic correctness, propagation delay. The boundary between domains is the most critical design point.

**Origami mapping:** Origami pipelines are hybrid systems. Early pipeline stages (recall, investigation) operate in the **unstructured** domain — they deal with natural language, free-form JSON, noisy LLM output. Later stages (judgment, synthesis) operate in the **structured** domain — they work with typed artifacts, validated schemas, boolean decisions. The `Extractor` sits at the unstructured-to-structured boundary; `RenderPrompt` sits at the structured-to-unstructured boundary.

**Zones** map naturally to data domains:
- **Unstructured zones** — Nodes that primarily consume and produce free-form data (backcourt / intake)
- **Structured zones** — Nodes that primarily consume and produce typed artifacts (frontcourt / synthesis)
- **Hybrid zones** — Nodes that convert between domains (the extraction / rendering boundary)

**Insight:** Treating zones as data domains changes how pipeline designers think about node placement. Moving a schema-validated node into an unstructured zone is like putting a digital IC on an analog board without proper decoupling — it will work, but suboptimally. The framework could warn when a node with `schema:` (structured) is placed in a zone dominated by free-form processing (unstructured), or vice versa.

**Possible adaptation:** An optional `domain:` annotation on zones (`unstructured`, `structured`, `hybrid`) that feeds into pipeline linting. The linter checks that extraction nodes sit at unstructured-to-structured zone boundaries, and prompt rendering happens at structured-to-unstructured boundaries.

```mermaid
flowchart LR
    subgraph unstructuredZone ["Unstructured Zone (Backcourt)"]
        recall["recall"] --> investigate["investigate"]
    end

    investigate -->|"Extract"| ext["Extractor"]
    ext --> judge

    subgraph structuredZone ["Structured Zone (Frontcourt)"]
        judge["judge"] --> synthesize["synthesize"]
    end

    synthesize -->|"Render"| rend["Renderer"]
    rend --> nextLLM["Next LLM call"]
```

The zone boundary is the most critical design point. Placing an extractor inside an unstructured zone or a renderer inside a structured zone is like placing an ADC in the middle of an analog filter chain — it quantizes the signal before conditioning is complete.

### Pattern 3: Impedance Matching

**Circuit principle:** Maximum power transfer between a source and load occurs when their impedances are conjugate-matched. Mismatched impedance causes signal reflection — energy bounces back instead of being absorbed. In RF design, impedance mismatch is measured as VSWR (voltage standing wave ratio): 1:1 is perfect, higher ratios mean more reflection and less useful power transfer.

**Origami mapping:** The `AffinityScheduler` already implements impedance matching: it selects walkers whose `Element` best matches a node's `ElementAffinity`. Fire walkers on Fire nodes = matched impedance = maximum "power transfer" (processing effectiveness). A Water walker on a Lightning node = mismatched impedance = signal reflection (the walker's deep, methodical nature fights the node's need for speed).

**Insight:** Circuit theory quantifies mismatch as a ratio, not a boolean. The `AffinityScheduler` currently picks the "best" match, but doesn't quantify *how much* quality degrades from a suboptimal match. An **impedance mismatch score** (0.0 = perfect match, 1.0 = total mismatch) on each walker-node assignment would let the framework:
- Log mismatch warnings when assignments exceed a threshold
- Feed mismatch data into calibration metrics (does high mismatch correlate with lower M1?)
- Let pipeline designers tune `stickiness` based on empirical mismatch data

**Possible adaptation:** Add a `Mismatch(walker, node) float64` method to `AffinityScheduler` that returns a quantified impedance ratio. Expose it via `WalkObserver` events so Kami can visualize mismatched assignments in the graph.

```mermaid
flowchart LR
    subgraph matched ["Matched: mismatch = 0.0"]
        fireW["Walker (Fire)"] -->|"Z matched"| fireN["Node (Fire affinity)"]
        fireN --> goodArt["High-quality artifact"]
    end

    subgraph mismatched ["Mismatched: mismatch = 0.8"]
        waterW["Walker (Water)"] -->|"Z mismatched"| fireN2["Node (Fire affinity)"]
        fireN2 --> poorArt["Degraded artifact"]
    end
```

A Water walker assigned to a Fire node is like connecting a high-impedance source to a low-impedance load: most of the "energy" (processing capability) is wasted as reflection (behavioral friction) rather than transferred into useful work.

### Pattern 4: Negative Feedback for Stability

**Circuit principle:** An op-amp without feedback has open-loop gain of ~100,000. Any tiny input difference drives the output to the supply rails (saturation). **Negative feedback** — feeding a fraction of the output back to the inverting input — trades gain for stability. The closed-loop gain becomes predictable: `G = 1/β` where β is the feedback fraction. The system self-corrects: if the output drifts high, the feedback drives it back down.

**Origami mapping:** Loop edges with convergence thresholds are negative feedback circuits. Each iteration through the loop compares the current output (confidence, completeness) against a target. If the output hasn't converged, the loop iterates again with corrective input. `Element.ConvergenceThreshold` is the feedback fraction β: it determines how much "error" (distance from target) is tolerable before the loop exits.

**Insight:** Circuit theory provides precise vocabulary for loop tuning:
- **Underdamped** (β too low, gain too high): the loop oscillates — successive iterations swing between overconfident and underconfident without converging. This is a pipeline that loops 3 times and produces wildly different answers each time.
- **Overdamped** (β too high, gain too low): the loop converges too slowly — it takes many iterations to reach an adequate answer, wasting compute. This is a pipeline with overly strict convergence criteria.
- **Critically damped** (β optimal): the loop converges in the minimum number of iterations without oscillation. This is the calibration target.
- **Instability** (positive feedback): if the loop amplifies rather than corrects errors, the system runs away. This is a pipeline where each iteration makes the output *worse* — a signal to break the loop and escalate to the Dialectic.

**Possible adaptation:** Track convergence trajectory across loop iterations. If confidence oscillates (increases then decreases then increases), flag as underdamped. If confidence barely changes per iteration, flag as overdamped. Log these as calibration signals.

### Pattern 5: Kirchhoff's Current Law — Data Conservation

**Circuit principle:** At every node in a circuit, the sum of currents entering equals the sum of currents leaving (KCL). No charge is created or destroyed. This is a conservation law — it holds unconditionally and is the basis of all circuit analysis.

**Origami mapping:** Every piece of evidence entering a node should be accounted for in the output — preserved, transformed, or explicitly discarded with justification. If a recall node surfaces 5 evidence items and the investigation node's artifact references only 2, what happened to the other 3? Were they irrelevant (legitimate filtering) or were they lost (information leak)?

**Insight:** KCL is not enforced in Origami today. A node can silently drop evidence. The `ArtifactSchema` validates output structure but not output completeness relative to input. Circuit theory says this is a fundamental gap: every junction must conserve current.

The pipeline equivalent of KCL: for every evidence item in a node's input, the output artifact must either (a) reference it, (b) transform it into a new form, or (c) explicitly declare it irrelevant with rationale. Option (c) is the "evidence drain" — current flowing to ground. It's legitimate, but it must be explicit.

**Possible adaptation:** An optional `evidence_conservation: strict` flag on nodes that activates input/output evidence tracking. The framework counts evidence items in and evidence items out (referenced + transformed + explicitly drained). A conservation violation triggers a warning via `WalkObserver`. Not a hard gate (too rigid for early pipeline stages), but a measurable signal for calibration tuning.

```mermaid
flowchart LR
    subgraph kcl ["KCL: current in = current out"]
        i1["I1 = 3A"] --> junction["Junction"]
        i2["I2 = 2A"] --> junction
        junction --> i3["I3 = 2A"]
        junction --> i4["I4 = 3A"]
    end

    subgraph conservation ["Evidence: items in = items accounted"]
        logs["3 log entries"] --> node["Investigation Node"]
        commits["2 commit refs"] --> node
        node --> referenced["2 referenced"]
        node --> transformed["1 transformed"]
        node --> drained["2 drained (explicit)"]
    end
```

In the circuit: 3A + 2A in = 2A + 3A out. Conservation holds. In the pipeline: 3 + 2 = 5 evidence items in = 2 referenced + 1 transformed + 2 explicitly drained = 5 accounted. The "drained" items are current flowing to ground — legitimate, but the drain must be explicit, not silent.

### Pattern 6: Decoupling Capacitors — Context Isolation

**Circuit principle:** Every IC has decoupling capacitors (typically 100nF ceramic) placed physically close to its power pins. Their purpose: prevent high-frequency noise generated by one chip from propagating through the power rail to affect other chips. They act as local energy reservoirs that absorb transient current demands, keeping the power supply clean for neighboring components.

**Origami mapping:** `WalkerState.Context` is a shared power rail — context accumulated at one node is available at all subsequent nodes. This is powerful (any node can access any prior context) but dangerous (noise from one node can pollute another's processing). A node that adds verbose, partially-relevant context to the walker state is injecting noise onto the shared rail.

**Insight:** Zone boundaries should act as decoupling capacitors. When a walker crosses from one zone to another, the context should be filtered: persistent, validated context (DC component — stable, always-relevant facts) passes through, while transient, speculative context (AC component — intermediate hypotheses, raw LLM fragments) is blocked or attenuated.

**Possible adaptation:** A `context_filter` field on zone definitions that specifies which context keys propagate across the zone boundary. Keys not listed are available within the zone but stripped when the walker exits. This prevents investigation-phase speculation from leaking into judgment-phase processing.

```mermaid
flowchart LR
    subgraph zoneA ["Investigation Zone"]
        n1["investigate"] --> n2["correlate"]
    end

    n2 --> filter["Context Filter"]
    filter -->|"pass: evidence, timeline"| n3
    filter -.->|"block: raw LLM, hypotheses"| stripped["stripped"]

    subgraph zoneB ["Judgment Zone"]
        n3["judge"] --> n4["synthesize"]
    end
```

In the circuit, a decoupling capacitor passes DC (stable power) while blocking AC (high-frequency noise). At a zone boundary, the context filter passes stable facts (evidence, timeline) while blocking transient noise (raw LLM fragments, speculative hypotheses). Both prevent upstream noise from corrupting downstream processing.

```yaml
zones:
  investigation:
    nodes: [recall, investigate, correlate]
    context_filter:
      pass: [evidence, artifacts, timeline]
      block: [raw_llm_output, intermediate_hypotheses]
  judgment:
    nodes: [judge, synthesize]
```

---

## 5. Gaps Illuminated by the Analogy

### Gap 1: DAC is not first-class

`RenderPrompt` is a utility function. `Extractor` is a registered, named, DSL-wirable interface with built-in implementations. In circuit design, treating the DAC as less important than the ADC would be a fundamental engineering error — both conversions are critical to system performance.

A `Renderer` interface symmetric to `Extractor` would:
- Be named and registered (`RendererRegistry`)
- Be DSL-wirable (`renderer: narrative-v1` on node definitions)
- Have built-in implementations (`TemplateRenderer`, `StructuredRenderer`, `NarrativeRenderer`)
- Participate in pipeline validation (`Validate()` checks renderer references)

This closes the ADC/DAC symmetry gap and elevates prompt construction from ad-hoc string formatting to a principled, testable, swappable pipeline component.

### Gap 2: No signal integrity metric

Circuits measure **signal-to-noise ratio** (SNR) at every stage. A signal chain with 60dB SNR at the input and 20dB SNR at the output has lost 40dB of signal quality — something is wrong.

Origami has `Confidence()` on artifacts, but no measure of **evidence preservation** through the pipeline. A node might output high confidence while silently discarding half the input evidence. Confidence measures the node's self-assessed certainty; SNR would measure how much of the input signal survived processing.

A pipeline-level evidence SNR metric would track: (evidence items referenced in output) / (evidence items available in input). Monotonically decreasing SNR across the pipeline is expected (each stage focuses the signal). A sudden drop at a specific node flags it as a lossy stage worth investigating.

### Gap 3: No power budget equivalent

Circuits have strict power budgets. Each component's power consumption is specified, and the total must not exceed the supply's capacity. Thermal analysis ensures no component overheats.

Origami has the `Safety > Speed` principle and the 50,000x ROI argument, but no per-node cost tracking. A `token_budget` or `cost_ceiling` on nodes would not gate execution (accuracy wins unconditionally) but would provide visibility: "This node consumed 15,000 tokens — 3x its typical budget. Is the prompt too verbose, or is the input unusually complex?"

This is an observability feature, not a throttling feature — consistent with `Safety > Speed` while adding cost awareness.

### Gap 4: No thermal throttling / backpressure

When a circuit's junction temperature exceeds its rating, thermal protection kicks in: the component reduces its operating frequency or shuts down to prevent damage. This is **backpressure** — the system protects itself from overload.

When an LLM API rate-limits or a dispatcher queue fills up, Origami has no backpressure mechanism. The framework could formalize:
- **Rate limiting** — maximum dispatch frequency per node or zone
- **Circuit breaker** — after N consecutive failures at a node, pause and escalate rather than retry indefinitely
- **Thermal budget** — track cumulative latency per walk; if it exceeds a threshold, the walk signals distress via `WalkObserver`

These patterns are standard in distributed systems (Hystrix, resilience4j) and universal in circuit design. They would complement the existing timeout SLAs in `agent-operations.mdc`.

---

## 6. Architectural Reflection

Electronic circuits and agentic pipelines are different instances of the same abstract architecture: **signal processing graphs**. Both route signals through active processing elements connected by conditional paths, with feedback loops for stability and converters at domain boundaries.

The key differences:

| Dimension | Electronic Circuit | Origami Pipeline |
|---|---|---|
| Signal type | Electrical (voltage/current) | Informational (artifacts/context) |
| Processing | Deterministic (physics) | Stochastic (LLM) |
| Noise source | Thermal, electromagnetic | Hallucination, ambiguity, prompt sensitivity |
| Feedback speed | Nanoseconds | Seconds to minutes |
| Design tool | SPICE simulation | Stub/dry/wet calibration |
| Failure mode | Smoke, shorts, oscillation | Wrong answers, loops, confidence collapse |

The stochastic nature of LLM processing is the fundamental difference. A resistor always obeys Ohm's law. An LLM node might produce different output for identical input. This means circuit-inspired patterns must be adapted with tolerance for non-determinism:
- Impedance matching becomes probabilistic affinity, not exact conjugate match
- KCL becomes evidence accounting, not exact conservation
- Convergence becomes statistical trend, not monotonic decrease
- Signal conditioning becomes prompt shaping, not precise filtering

Despite this, the structural patterns transfer remarkably well. The mixed-signal architecture pattern in particular reframes pipeline design: stop thinking of pipelines as uniform processing chains and start thinking of them as systems with distinct signal domains, critical conversion boundaries, and domain-specific design rules.

---

## 7. Actionable Takeaways

1. **Renderer interface (DAC symmetry)** — Define a `Renderer` interface symmetric to `Extractor`: `Name() string`, `Render(ctx context.Context, data any) (string, error)`. Add `RendererRegistry`. Wire into DSL via `renderer:` field on nodes. This closes the most significant gap the analogy reveals — the asymmetric treatment of the two conversion directions.

2. **Data-domain zone annotations** — Add an optional `domain:` field to `ZoneDef` (`unstructured`, `structured`, `hybrid`). Feed into `origami lint` to warn about extraction nodes outside unstructured-to-structured boundaries and schema-validated nodes in unstructured zones. Low-cost DSL addition with design-time value.

3. **Context filter on zone boundaries** — Add `context_filter:` to `ZoneDef` with `pass` and `block` lists. When a walker crosses a zone boundary, strip blocked keys from `WalkerState.Context`. This is the decoupling capacitor pattern — prevents context noise from propagating across domain boundaries.

4. **Impedance mismatch scoring** — Add `Mismatch(walkerElement, nodeElement) float64` to the affinity calculation. Expose via `WalkObserver` so Kami can visualize mismatched assignments. Feed into calibration metrics to correlate mismatch with outcome quality.

5. **Convergence trajectory tracking** — Track confidence values across loop iterations. Classify as underdamped (oscillating), overdamped (stagnant), critically damped (optimal), or unstable (diverging). Log classification via `WalkObserver`. Use as a calibration signal for tuning `Element.ConvergenceThreshold`.

6. **Evidence SNR metric** — Track evidence item counts at node input and output boundaries. Compute per-node and per-walk SNR. Surface in calibration reports alongside confidence scores. This makes evidence preservation measurable rather than assumed.

7. **Signal conditioning vocabulary in docs** — Adopt circuit conditioning vocabulary (anti-aliasing, amplification, level-shifting, impedance matching) in mask and pipeline design documentation. No code change needed — purely a conceptual framework that helps pipeline designers make principled mask placement decisions.

8. **Gap-closure convergence check** — Change dialectic convergence from "did we exhaust MaxTurns?" to "did the thesis-antithesis gap close?" (Op-amp Golden Rule 1). Measure remaining disagreement between the final thesis and antithesis. A dialectic that runs MaxTurns rounds but leaves major contradictions is like an op-amp that hasn't settled — still in transient.

9. **Evidence immutability guarantee** — Enforce read-only access to walker context and prior artifacts during D0-D4 (Op-amp Golden Rule 2: zero input current). The debate process should observe evidence, never alter it. Violations bias the output by process, not evidence.

10. **Shared-assumption detection (CMRR check)** — Add a dedicated challenge step that surfaces premises shared by both thesis and antithesis. Shared agreement in a dialectic should raise suspicion (potential shared bias), not confidence. This is the one place where consensus is a warning signal.

11. **Quality-speed product measurement** — Use Ouroboros to empirically measure each model's GBWP equivalent: run the same dialectic at varying MaxTurns and plot confidence accuracy vs. rounds. The resulting curve lets pipeline designers choose the optimal operating point for their latency/quality tradeoff.

12. **Persona offset compensation** — When Ouroboros measures a persona's behavioral bias (offset voltage), inject a corrective preamble instruction to trim it. This is the dialectic equivalent of adjusting an op-amp's offset null potentiometer.

---

## References

- Electronic circuit fundamentals: `en.wikipedia.org/wiki/Electronic_circuit`
- Operational amplifier: `en.wikipedia.org/wiki/Operational_amplifier` (golden rules, CMRR, GBWP, compensation, slew rate)
- Horowitz & Hill, *The Art of Electronics* (golden rules of ideal op-amps)
- ADC principles: `en.wikipedia.org/wiki/Analog-to-digital_converter`
- DAC principles: `en.wikipedia.org/wiki/Digital-to-analog_converter`
- Kirchhoff's circuit laws: `en.wikipedia.org/wiki/Kirchhoff%27s_circuit_laws`
- Impedance matching: `en.wikipedia.org/wiki/Impedance_matching`
- Negative feedback: `en.wikipedia.org/wiki/Negative_feedback`
- Signal conditioning: `en.wikipedia.org/wiki/Signal_conditioning`
- Mixed-signal IC design: `en.wikipedia.org/wiki/Mixed-signal_integrated_circuit`
- Origami Extractor: `extractor.go` (Extractor interface, ExtractorRegistry, built-in extractors)
- Origami Prompt Rendering: `render.go` (RenderPrompt utility)
- Origami Elements: `element.go` (6 elements, quantified traits, ConvergenceThreshold)
- Origami Masks: `mask.go` (composable behavioral middleware, pre/post hooks)
- Origami Zones: `dsl.go` (ZoneDef, stickiness, element affinity)
- Origami Artifact: `node.go` (Artifact interface, Confidence, Raw)
- Origami AffinityScheduler: `scheduler.go` (walker-node matching)
- Origami WalkObserver: `observer.go` (observability events)
- Origami Adversarial Dialectic: `dialectic.go` (D0-D4, thesis-antithesis-synthesis)
- Related case studies: `langgraph-graph-duality.md` (graph philosophy), `cloud-native-pipeline-tools.md` (infrastructure patterns)
