# Red Hat Presentation DNA

**Source:** Red Hat Corporate Slide Template (Google Slides, v00000, Dec 2025, 123 slides)  
**Purpose:** Distilled brand guide for Origami ecosystem UIs, presentations, and demos. All visual output — Kami frontend, Visual Editor, demo presentation web app, calibration reports — must comply with this color system and design language.

---

## 1. Color System

### 1.1 Core Reds

| Token | Hex | Usage |
|-------|-----|-------|
| red-10 | `#fce3e3` | Light tint backgrounds |
| red-20 | `#fbc5c5` | Subtle emphasis |
| red-30 | `#f9a8a8` | Secondary accents |
| red-40 | `#f56e6e` | Interactive hover states |
| **red-50** | **`#ee0000`** | **Red Hat Red — primary brand color** |
| red-60 | `#a60000` | Dark accent, headings on light backgrounds |
| red-70 | `#5f0000` | Deepest red, high-contrast headers |

### 1.2 Neutrals

| Token | Hex | Usage |
|-------|-----|-------|
| white | `#ffffff` | Backgrounds, text on dark |
| gray-20 | `#e0e0e0` | Borders, dividers |
| gray-40 | `#a3a3a3` | Secondary text, disabled |
| gray-60 | `#4d4d4d` | Body text |
| gray-80 | `#292929` | Headings, high emphasis |
| black | `#000000` | Maximum contrast |

### 1.3 Color Collections (for diagrams, charts, and multi-color UIs)

Pick **one** collection per presentation or UI context. Do not mix collections.

**Collection 1 — Red + Purple + Teal** (recommended for Origami)

| Token | Hex |
|-------|-----|
| red-70 | `#5f0000` |
| red-50 | `#ee0000` |
| purple-70 | `#21134d` |
| purple-50 | `#5e40be` |
| teal-50 | `#37a3a3` |
| teal-30 | `#9ad8d8` |
| teal-10 | `#daf2f2` |
| black | `#000000` |

**Collection 2 — Red + Purple + Orange**

| Token | Hex |
|-------|-----|
| red-70 | `#5f0000` |
| red-50 | `#ee0000` |
| purple-70 | `#21134d` |
| purple-50 | `#5e40be` |
| orange-50 | `#f5921b` |
| orange-30 | `#fccb8f` |
| orange-10 | `#ffe8cc` |
| black | `#000000` |

**Collection 3 — Red + Yellow + Teal**

| Token | Hex |
|-------|-----|
| red-70 | `#5f0000` |
| red-60 | `#a60000` |
| red-50 | `#ee0000` |
| yellow-20 | `#ffe072` |
| yellow-10 | `#fff4cc` |
| teal-70 | `#004d4d` |
| teal-50 | `#37a3a3` |
| black | `#000000` |

**Collection 4 — Red + Purple + Yellow**

| Token | Hex |
|-------|-----|
| red-70 | `#5f0000` |
| red-50 | `#ee0000` |
| purple-70 | `#21134d` |
| purple-50 | `#5e40be` |
| yellow-30 | `#ffe072` |
| yellow-20 | `#ffe072` |
| yellow-10 | `#fff4cc` |
| black | `#000000` |

---

## 2. Element-to-Color Mapping

Origami's 6 elements must render using RH-approved colors. The mapping uses **Collection 1** (Red + Purple + Teal) as the primary palette, borrowing from Collections 2-3 for full coverage.

| Element | Personality | RH Color | Hex | Tint (backgrounds) | Tint Hex |
|---------|-------------|----------|-----|---------------------|----------|
| **Fire** | Decisive, fast, impatient | red-50 | `#ee0000` | red-10 | `#fce3e3` |
| **Water** | Thorough, deep, patient | teal-50 | `#37a3a3` | teal-10 | `#daf2f2` |
| **Earth** | Methodical, steady, reliable | gray-60 | `#4d4d4d` | gray-20 | `#e0e0e0` |
| **Air** | Creative, holistic, lateral | purple-50 | `#5e40be` | purple-10 | `#f0ebff` |
| **Diamond** | Precise, exacting, rigorous | yellow-20 | `#ffe072` | yellow-10 | `#fff4cc` |
| **Lightning** | Fastest, volatile, direct | orange-50 | `#f5921b` | orange-10 | `#ffe8cc` |

**Iron** (derived element) inherits Earth's colors: gray-60 / gray-20.

### Zone backgrounds

Pipeline zones use the **tint column** as background fill. This provides sufficient contrast for text and node shapes while maintaining visual separation between zones.

### Node shapes

Node borders use the **RH Color** column. Active nodes use a filled background at 20% opacity of the element color. Completed nodes use the full tint.

---

## 3. Slide Type Catalog

### 3.1 Types and when to use them

| Slide Type | RH Template Slides | Use For |
|------------|-------------------|---------|
| **Title** | 9, 11, 13, 15, 17, 19, 73, 75, 77, 79 | Opening slide. Product logo optional. Max 2 presenters. |
| **Thank You / Closing** | 10, 12, 14, 16, 18, 20, 74, 76, 78, 80 | Final slide. RH boilerplate + social links. Always use. |
| **Divider** | 23-28, 82-84 | Section breaks. Title max 3 lines. Optional supporting copy. |
| **Agenda** | 30-33, 86-87 | Session outline. Use `▸` bullet markers. |
| **Content** | 34-55, 88-107 | Core content. Title max 1 line. 6 layout variants (text-only, image+text, split columns, icons). |
| **Quote** | 56-58, 108-110 | Customer/stakeholder quotes. Attribution: Name + Title + Company. |
| **Data / Chart** | 60-66, 112-118 | Pie, bar, percentage callouts. Source citation required. Accessibility: never color-alone. |
| **Table** | 67-68, 119-120 | Structured comparisons. Column/row headers max 2 lines. Body cells max 2 lines. |
| **Timeline** | 69, 121 | Chronological progression. Year markers (20XX). Max 5 milestones per slide. |
| **Full-bleed Photo** | 70-71, 122-123 | Impact/transition moment. One heading + one paragraph overlay. |

### 3.2 Layout variants (light vs dark)

The template provides two complete sets: slides 9-71 (light theme) and slides 73-123 (dark theme). Both share identical structure. Choose one theme per presentation; do not mix.

---

## 4. Design Constraints

### Typography and content

- Presentation title: **max 2 lines**
- Slide title: **max 1 line** (divider slides: max 3 lines)
- Subheading: **max 3 lines**
- Table body cells: **max 2 lines**
- Bullet marker: **`▸`** (right-pointing triangle, not `-`, `*`, or `•`)
- Source citation: **required** on every data/chart/table slide (`Source: <reference>`)

### Color usage

- Start with core reds + neutrals for most slides
- Use one color collection (1-4) for diagrams, charts, and multi-color UIs
- **Never use color alone** to distinguish data — combine with text sizing, spacing, contrast, or patterns
- High contrast between text and background at all times

### Product branding

- Product logo: optional. Use right-click "Replace Image" to insert, never manually place the RH logo
- Version number: in theme footer area, not in slide content
- Confidential designator: in theme header, updated via Slide > Edit Theme

### Accessibility

- Text sizing and spacing must complement color differentiation
- All charts must have text labels, not just colored segments
- Refer to Red Hat Visual Accessibility guidelines for full requirements

---

## 5. Mapping to Origami Outputs

### 5.1 Demo presentation web app (Asterisk `demo-presentation` contract)

The demo is a **single interactive web app** (React + Vite + Tailwind) that IS the presentation. RH slide types become web section patterns. The app is served by `asterisk demo --replay` and wraps the Kami graph visualization inside an RH-branded story flow.

| Section | Web Pattern | Content |
|---------|-------------|---------|
| **Hero** | Hero | Full-viewport, animated Origami logo, "Asterisk: AI-Driven Root-Cause Analysis", presenter info |
| **Agenda** | Navigator | Interactive section navigator with `▸` markers, click-to-jump between sections |
| **Problem** | SplitPane | CI failure stats left, animated counter right |
| **Solution** | IconGrid | Pipeline graph preview (static Mermaid render), 7 nodes, 3 zones |
| **Agent Intros** | CardCarousel | 3D CSS polyhedra per agent, name, element, personality tags, model identity |
| **Transition** | Divider | Full-screen animated text: "Time to investigate some crimes against CI" |
| **Live Demo** | EmbeddedKami | Kami graph visualization embedded directly, SSE-driven animation (the centerpiece) |
| **Results** | MetricCard | Animated M19 bar comparison, metric cards, live-updating if `--live` |
| **Competitive** | InteractiveTable | Origami vs CrewAI vs OmO with hover highlights |
| **Architecture** | ImagePane | Mermaid diagram rendered client-side |
| **Roadmap** | HorizontalTimeline | Animated milestone dots for Sprint 1-6 |
| **Closing** | Closing | RH boilerplate, social links, CTA |

#### 5.1.1 Web section patterns

Each RH slide type maps to a reusable React component pattern. These patterns form the section library for the presentation SPA.

| RH Slide Type | React Component | Behavior |
|---------------|-----------------|----------|
| Title | `<Hero>` | Full-viewport, centered title, animated entrance (fade-up + scale). Background: dark theme with red-50 accent line. |
| Divider | `<Transition>` | Full-screen, CSS text animation (typewriter or slide-in). Background: solid red-50 or purple-70. |
| Agenda | `<Navigator>` | Sticky sidebar or horizontal nav strip. `▸` markers highlight current section. Click-to-jump scrolls smoothly. |
| Content (image+text) | `<SplitPane>` | Two-column layout: text/stats left, image/animation right. Responsive stacking on mobile. |
| Content (icons) | `<IconGrid>` | Grid of icon+label cards. Each card has element-colored border. Used for pipeline node overview. |
| Content (split columns) | `<CardCarousel>` | Horizontal card strip with CSS 3D transforms. Each card shows an agent with element-colored polyhedron. |
| Data / Chart | `<MetricCard>` / `<BarChart>` | Animated on scroll-into-view (IntersectionObserver). Numbers count up, bars grow. Source citation footer. |
| Table | `<InteractiveTable>` | Row/column hover highlights. Responsive: horizontal scroll on mobile. Header max 2 lines, cells max 2 lines. |
| Timeline | `<HorizontalTimeline>` | Animated milestone progression (dots connect left-to-right on scroll). Year markers in 20XX format. Max 5 milestones visible. |
| Quote | `<MonologuePanel>` | Agent personality panel with element-colored left border. Persona avatar + quoted text. Used in Act 3 monologues. |
| Thank You | `<Closing>` | RH boilerplate layout: logo, social links row, CTA button (red-50). Centered, minimal. |
| Full-bleed Photo | `<EmbeddedKami>` | The live pipeline visualization. Full-width, no padding. Receives SSE events and renders the Kami graph in real-time. |

### 5.2 Kami frontend

The React frontend uses RH Color Collection 1 as the base theme:

- **Graph background**: white
- **Zone backgrounds**: Element tints (see Section 2)
- **Node borders**: Element colors (see Section 2)
- **Active node glow**: red-50 `#ee0000`
- **Edge lines**: gray-40 `#a3a3a3`
- **Edge labels**: gray-60 `#4d4d4d`
- **Panel backgrounds**: gray-20 `#e0e0e0` or teal-10 `#daf2f2`
- **Primary action buttons**: red-50 `#ee0000`
- **Secondary buttons**: purple-50 `#5e40be`
- **Text primary**: gray-80 `#292929`
- **Text secondary**: gray-60 `#4d4d4d`

### 5.3 Visual Editor

The Visual Editor follows the same color system as Kami, plus:

- **Enterprise Edition**: Use [PatternFly](https://www.patternfly.org/) (Red Hat's open-source design system) for all UI components — buttons, forms, tables, navigation, alerts. PatternFly is already RH-brand-compliant.
- **Community Edition**: PatternFly recommended but not required. Must still use RH color collections.

### 5.4 Calibration reports

CLI-generated calibration reports (ASCII tables, markdown summaries) should use the `▸` bullet marker and follow the table constraint (header max 2 lines, body cells max 2 lines) for any output that may be pasted into slides.

---

## References

- Red Hat brand standards: `brand.redhat.com`
- Red Hat icon repository: Google Slides (internal link in template)
- PatternFly design system: `patternfly.org`
- Origami elements: `element.go` (6 core + Iron derived)
- Origami personas: `persona.go` (8 personas, 4 Light + 4 Shadow)
- Visual Accessibility at Red Hat: internal resource (referenced in template)
