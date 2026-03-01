# Laws of UX — Applied to Kabuki

[Laws of UX](https://lawsofux.com/) codifies 30 psychology-backed design principles.
The following 10 directly govern how the Kabuki presentation engine should look and feel.
Each law maps to concrete design decisions in the Kabuki frontend and Asterisk demo content.

## Applicable Laws

### Aesthetic-Usability Effect

> Users perceive aesthetically pleasing design as design that's more usable.

A polished demo builds trust in the product *before* the audience evaluates
functionality. Every section must feel intentional: consistent spacing, harmonious
colors, restrained animation. Color harmony matters more than feature count.

**Kabuki implication:** Use the full Red Hat brand palette with semantic tokens.
Limit secondary colors to 1–2 per act. Restrain red to accents, not fills.

### Peak-End Rule

> People judge an experience by the peak and the ending, not the average.

The War Room (Act 1 peak) and "The Process" grand finale (Act 3 peak) must be the
most visually impressive sections. The Closing section must leave a strong emotional
impression. Invest disproportionate polish in these three.

**Kabuki implication:** War Room gets full-width layout, agent-colored tabs, TX/RX
panels, animated graph. Closing gets a cinematic feel with generous whitespace.

### Von Restorff Effect (Isolation Effect)

> When multiple similar objects are present, the one that differs is remembered.

The active circuit node must stand out through *multiple* channels: color, animation
(pulse), size, and border weight. Don't rely on color alone (accessibility concern).
Red Hat brand concurs: "Use pops of red to highlight key elements."

**Kabuki implication:** Active node gets pulse animation + heavier border + element
color fill. Visited nodes get a subtle teal tint. Unvisited nodes stay neutral.

### Doherty Threshold

> Productivity soars when interactions respond in <400ms.

All CSS transitions must be 200–300ms. Loading states must show progress. Scroll-snap
transitions must feel instant. No perceptible lag on section changes or graph updates.

**Kabuki implication:** CSS `transition-duration: 200ms` as default. Animate-pulse
at 2s period for active node (slow enough to notice, fast enough to feel alive).

### Hick's Law

> Decision time increases with the number and complexity of choices.

26 sections is overwhelming in a flat agenda. Group sections into 3 acts with visual
separators. Progressive disclosure: agenda shows act headers, expands to show
sections within each act.

**Kabuki implication:** AgendaSection groups items by act. Each act is a visual
chunk with a header. Scrolling to an act highlights the act header.

### Miller's Law + Chunking

> Working memory holds 7 ± 2 items.

Act 1 has 8 sections, Act 2 has 9, Act 3 has 6, bookends have 3. Each act is within
cognitive limits. The Agenda must visually chunk by act, not list 26 items flat.

**Kabuki implication:** Agenda renders act groups. Each group header shows the act
name and section count. Max 9 items per group.

### Law of Common Region

> Elements in a shared boundary are perceived as grouped.

Dark background sections mark act transitions and immersive moments (Hero, War Room,
Closing). Light background sections are content and explanation. This creates a
visual rhythm that signals "new context" vs "continuing context."

**Kabuki implication:** Alternating light/dark sections use `--surface-canvas` and
`--surface-accent` semantic tokens. The rhythm is predictable.

### Law of Uniform Connectedness

> Visually connected elements are perceived as more related.

Circuit graph edges with directional arrows are required — without them, nodes feel
isolated. Color-coded edges connecting same-element nodes reinforce relationships.

**Kabuki implication:** CircuitGraph uses dagre layout with SVG arrows. Edge color
matches the source node's element color.

### Fitts's Law

> Target acquisition time is proportional to distance and inversely proportional to size.

Navigation controls (keyboard arrows, agenda links, War Room agent tabs) must be
large touch targets (minimum 44px). War Room tabs at the top of the viewport, not
buried in a sidebar.

**Kabuki implication:** Agenda items have `min-height: 44px`. Agent tabs have
`min-width: 44px`. Control buttons have `padding: 12px 24px`.

### Serial Position Effect

> People best remember the first and last items in a series.

Hero (first) and Closing (last) deserve disproportionate polish. The Hero title,
subtitle, and framework line are the audience's first impression. The Closing
headline is the last thing they see.

**Kabuki implication:** Hero gets the largest type scale (6xl). Closing gets generous
whitespace and a strong tagline. Both use `--surface-accent` (always dark) for
cinematic weight.

## Color Harmony Principles

Derived from [Red Hat Brand Standards](https://www.redhat.com/en/about/brand/standards/color):

- **Red is accent, not fill.** Don't flood with Red Hat red. Use pops of red-50 to
  highlight key elements.
- **1–2 secondary colors per composition.** Act 1 pairs red + teal. Act 2 pairs
  red + purple. Act 3 combines all three.
- **Gray-95 (#151515) is UX black.** Pure black (#000000) is limited to logos and
  graphics. Use gray-95 for all UI surfaces.
- **WCAG AA compliance.** Red-50 on gray-80 fails AA contrast (3.2:1). Use red-40
  (#f56e6e) on dark backgrounds, or white text with red accent elements.
- **Dark mode uses lighter values.** Element colors shift up one step in dark mode
  (fire: red-50 → red-40, earth: purple-50 → purple-40) to maintain contrast.

## References

- [Laws of UX](https://lawsofux.com/) — Jon Yablonski
- [Red Hat Brand Standards: Color](https://www.redhat.com/en/about/brand/standards/color)
- [Red Hat Design System: Color Usage](https://ux.redhat.com/foundations/color/usage)
- [Red Hat Design System: Color Palettes](https://ux.redhat.com/theming/color-palettes)
