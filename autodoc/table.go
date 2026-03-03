package autodoc

import (
	"fmt"
	"strings"

	framework "github.com/dpopsuev/origami"
)

// RenderNodeTable generates a Markdown table with per-node reference information.
func RenderNodeTable(def *framework.CircuitDef, opts *MermaidOptions) string {
	nodeDS := classifyNodes(def, opts)
	zoneOf := buildZoneMap(def)

	var b strings.Builder
	b.WriteString("| Node | Description | Zone | Family | Transformer | Element | Hooks | D/S |\n")
	b.WriteString("|------|-------------|------|--------|-------------|---------|-------|-----|\n")

	for _, nd := range def.Nodes {
		desc := nd.Description
		if desc == "" {
			desc = "-"
		}
		zone := zoneOf[nd.Name]
		if zone == "" {
			zone = "-"
		}
		family := nd.Family
		if family == "" {
			family = "-"
		}
		transformer := nd.Transformer
		if transformer == "" {
			transformer = "-"
		}
		element := nd.Element
		if element == "" {
			element = "-"
		}
		hooks := "-"
		if len(nd.After) > 0 {
			hooks = strings.Join(nd.After, ", ")
		}

		ds := "-"
		switch nodeDS[nd.Name] {
		case dsDeterministic:
			ds = "D"
		case dsStochastic:
			ds = "S"
		}

		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s | %s | %s |\n",
			nd.Name, desc, zone, family, transformer, element, hooks, ds)
	}

	return b.String()
}

// RenderSummary generates a summary section with aggregate statistics.
func RenderSummary(def *framework.CircuitDef, opts *MermaidOptions) string {
	nodeDS := classifyNodes(def, opts)

	detCount, stochCount, unknownCount := 0, 0, 0
	for _, nd := range def.Nodes {
		switch nodeDS[nd.Name] {
		case dsDeterministic:
			detCount++
		case dsStochastic:
			stochCount++
		default:
			unknownCount++
		}
	}

	shortcuts, loops := 0, 0
	for _, e := range def.Edges {
		if e.Shortcut {
			shortcuts++
		}
		if e.Loop {
			loops++
		}
	}

	var b strings.Builder
	b.WriteString("## Summary\n\n")
	fmt.Fprintf(&b, "- **Nodes:** %d", len(def.Nodes))
	if detCount > 0 || stochCount > 0 {
		fmt.Fprintf(&b, " (%d deterministic, %d stochastic", detCount, stochCount)
		if unknownCount > 0 {
			fmt.Fprintf(&b, ", %d unclassified", unknownCount)
		}
		b.WriteString(")")
	}
	b.WriteString("\n")
	fmt.Fprintf(&b, "- **Edges:** %d", len(def.Edges))
	if shortcuts > 0 || loops > 0 {
		parts := []string{}
		if shortcuts > 0 {
			parts = append(parts, fmt.Sprintf("%d shortcut", shortcuts))
		}
		if loops > 0 {
			parts = append(parts, fmt.Sprintf("%d loop", loops))
		}
		fmt.Fprintf(&b, " (%s)", strings.Join(parts, ", "))
	}
	b.WriteString("\n")
	if len(def.Zones) > 0 {
		fmt.Fprintf(&b, "- **Zones:** %d\n", len(def.Zones))
	}

	return b.String()
}
