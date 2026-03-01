/**
 * Bidirectional sync engine: graph changes → YAML diffs, YAML changes → graph update.
 *
 * The graph is the source of truth for visual edits (drag, connect, delete).
 * The YAML is the source of truth for text edits (Monaco).
 * Both directions produce diffs that are applied to the other side.
 */

export interface CircuitNode {
  name: string;
  element?: string;
  family?: string;
  extractor?: string;
  transformer?: string;
  provider?: string;
  prompt?: string;
  x?: number;
  y?: number;
}

export interface CircuitEdge {
  id: string;
  name?: string;
  from: string;
  to: string;
  when?: string;
  shortcut?: boolean;
  loop?: boolean;
  condition?: string;
}

export interface CircuitWalker {
  name: string;
  element?: string;
  persona?: string;
  preamble?: string;
}

export interface CircuitZone {
  name: string;
  nodes: string[];
  element?: string;
  stickiness?: number;
}

export interface CircuitDef {
  circuit: string;
  description?: string;
  nodes: CircuitNode[];
  edges: CircuitEdge[];
  walkers?: CircuitWalker[];
  zones?: CircuitZone[];
  start?: string;
  done?: string;
  vars?: Record<string, unknown>;
}

/**
 * Parse YAML text into a CircuitDef.
 * Uses a simple line-based parser for the subset of YAML we generate.
 * Full YAML parsing would use js-yaml but this keeps the bundle small.
 */
export function parseYAML(text: string): CircuitDef | null {
  try {
    // Simple structured parser for Origami circuit YAML
    const lines = text.split("\n");
    const def: CircuitDef = {
      circuit: "",
      nodes: [],
      edges: [],
    };

    let section = "";
    let current: Record<string, unknown> = {};

    for (const line of lines) {
      const trimmed = line.trim();
      if (!trimmed || trimmed.startsWith("#")) continue;

      // Top-level scalar
      if (!line.startsWith(" ") && !line.startsWith("-")) {
        const [key, ...rest] = trimmed.split(":");
        const val = rest.join(":").trim().replace(/^["']|["']$/g, "");

        if (key === "circuit") def.circuit = val;
        else if (key === "description") def.description = val;
        else if (key === "start") def.start = val;
        else if (key === "done") def.done = val;
        else if (["nodes", "edges", "walkers", "zones"].includes(key)) {
          section = key;
        }
        continue;
      }

      // List item start
      if (trimmed.startsWith("- ")) {
        if (Object.keys(current).length > 0) {
          pushItem(def, section, current);
        }
        current = {};
        const inner = trimmed.slice(2);
        const [key, ...rest] = inner.split(":");
        if (rest.length > 0) {
          current[key.trim()] = rest.join(":").trim().replace(/^["']|["']$/g, "");
        }
        continue;
      }

      // Continuation fields
      if (trimmed.includes(":")) {
        const [key, ...rest] = trimmed.split(":");
        const val = rest.join(":").trim().replace(/^["']|["']$/g, "");
        if (val === "true") current[key.trim()] = true;
        else if (val === "false") current[key.trim()] = false;
        else current[key.trim()] = val;
      }
    }

    if (Object.keys(current).length > 0) {
      pushItem(def, section, current);
    }

    return def.circuit ? def : null;
  } catch {
    return null;
  }
}

function pushItem(
  def: CircuitDef,
  section: string,
  item: Record<string, unknown>
) {
  switch (section) {
    case "nodes":
      def.nodes.push(item as unknown as CircuitNode);
      break;
    case "edges":
      def.edges.push(item as unknown as CircuitEdge);
      break;
    case "walkers":
      if (!def.walkers) def.walkers = [];
      def.walkers.push(item as unknown as CircuitWalker);
      break;
  }
}

/**
 * Serialize a CircuitDef back to YAML text.
 */
export function toYAML(def: CircuitDef): string {
  const lines: string[] = [];

  lines.push(`circuit: ${def.circuit}`);
  if (def.description) lines.push(`description: "${def.description}"`);
  if (def.vars && Object.keys(def.vars).length > 0) {
    lines.push("vars:");
    for (const [k, v] of Object.entries(def.vars)) {
      lines.push(`  ${k}: ${JSON.stringify(v)}`);
    }
  }

  if (def.zones && def.zones.length > 0) {
    lines.push("zones:");
    for (const zone of def.zones) {
      lines.push(`  ${zone.name}:`);
      lines.push(`    nodes: [${zone.nodes.join(", ")}]`);
      if (zone.element) lines.push(`    element: ${zone.element}`);
      if (zone.stickiness) lines.push(`    stickiness: ${zone.stickiness}`);
    }
  }

  lines.push("nodes:");
  for (const node of def.nodes) {
    lines.push(`  - name: ${node.name}`);
    if (node.element) lines.push(`    element: ${node.element}`);
    if (node.family) lines.push(`    family: ${node.family}`);
    if (node.extractor) lines.push(`    extractor: ${node.extractor}`);
    if (node.transformer) lines.push(`    transformer: ${node.transformer}`);
    if (node.provider) lines.push(`    provider: ${node.provider}`);
    if (node.prompt) lines.push(`    prompt: "${node.prompt}"`);
  }

  lines.push("edges:");
  for (const edge of def.edges) {
    lines.push(`  - id: ${edge.id}`);
    if (edge.name) lines.push(`    name: ${edge.name}`);
    lines.push(`    from: ${edge.from}`);
    lines.push(`    to: ${edge.to}`);
    if (edge.when) lines.push(`    when: "${edge.when}"`);
    if (edge.shortcut) lines.push("    shortcut: true");
    if (edge.loop) lines.push("    loop: true");
    if (edge.condition) lines.push(`    condition: "${edge.condition}"`);
  }

  if (def.walkers && def.walkers.length > 0) {
    lines.push("walkers:");
    for (const w of def.walkers) {
      lines.push(`  - name: ${w.name}`);
      if (w.element) lines.push(`    element: ${w.element}`);
      if (w.persona) lines.push(`    persona: ${w.persona}`);
      if (w.preamble) lines.push(`    preamble: "${w.preamble}"`);
    }
  }

  if (def.start) lines.push(`start: ${def.start}`);
  if (def.done) lines.push(`done: ${def.done}`);

  return lines.join("\n") + "\n";
}

/**
 * Apply a graph change (node move, edge create, etc.) to a CircuitDef.
 */
export function applyGraphChange(
  def: CircuitDef,
  change: GraphChange
): CircuitDef {
  const next = structuredClone(def);

  switch (change.type) {
    case "node-add":
      next.nodes.push(change.node!);
      break;
    case "node-remove":
      next.nodes = next.nodes.filter((n) => n.name !== change.nodeId);
      next.edges = next.edges.filter(
        (e) => e.from !== change.nodeId && e.to !== change.nodeId
      );
      break;
    case "node-update":
      next.nodes = next.nodes.map((n) =>
        n.name === change.nodeId ? { ...n, ...change.updates } : n
      );
      break;
    case "edge-add":
      next.edges.push(change.edge!);
      break;
    case "edge-remove":
      next.edges = next.edges.filter((e) => e.id !== change.edgeId);
      break;
    case "edge-update":
      next.edges = next.edges.map((e) =>
        e.id === change.edgeId ? { ...e, ...change.updates } : e
      );
      break;
    case "set-start":
      next.start = change.nodeId;
      break;
  }

  return next;
}

export type GraphChangeType =
  | "node-add"
  | "node-remove"
  | "node-update"
  | "edge-add"
  | "edge-remove"
  | "edge-update"
  | "set-start";

export interface GraphChange {
  type: GraphChangeType;
  nodeId?: string;
  edgeId?: string;
  node?: CircuitNode;
  edge?: CircuitEdge;
  updates?: Record<string, unknown>;
}
