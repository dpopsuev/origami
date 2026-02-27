import type { PipelineDef } from "./yaml-sync";

export interface ValidationError {
  severity: "error" | "warning" | "info";
  message: string;
  field?: string;
}

export function validatePipeline(def: PipelineDef): ValidationError[] {
  const errors: ValidationError[] = [];

  if (!def.pipeline) {
    errors.push({ severity: "error", message: "Pipeline name is required", field: "pipeline" });
  }

  if (def.nodes.length === 0) {
    errors.push({ severity: "error", message: "At least one node is required", field: "nodes" });
  }

  if (!def.start) {
    errors.push({ severity: "error", message: "Start node is required", field: "start" });
  } else if (!def.nodes.find((n) => n.name === def.start)) {
    errors.push({
      severity: "error",
      message: `Start node "${def.start}" not found in nodes`,
      field: "start",
    });
  }

  const nodeNames = new Set(def.nodes.map((n) => n.name));
  const validElements = new Set([
    "fire", "water", "earth", "air", "diamond", "lightning", "iron",
  ]);

  for (const node of def.nodes) {
    if (!node.name) {
      errors.push({ severity: "error", message: "Node name is required", field: "nodes" });
    }
    if (node.element && !validElements.has(node.element)) {
      errors.push({
        severity: "error",
        message: `Invalid element "${node.element}" on node "${node.name}"`,
        field: `nodes.${node.name}.element`,
      });
    }
  }

  const dupes = def.nodes
    .map((n) => n.name)
    .filter((n, i, arr) => arr.indexOf(n) !== i);
  for (const dupe of new Set(dupes)) {
    errors.push({
      severity: "error",
      message: `Duplicate node name: "${dupe}"`,
      field: "nodes",
    });
  }

  for (const edge of def.edges) {
    if (!edge.id) {
      errors.push({ severity: "error", message: "Edge ID is required", field: "edges" });
    }
    if (!edge.from) {
      errors.push({
        severity: "error",
        message: `Edge ${edge.id}: "from" is required`,
        field: `edges.${edge.id}.from`,
      });
    } else if (!nodeNames.has(edge.from) && edge.from !== "DONE") {
      errors.push({
        severity: "warning",
        message: `Edge ${edge.id}: "from" node "${edge.from}" not found`,
        field: `edges.${edge.id}.from`,
      });
    }
    if (!edge.to) {
      errors.push({
        severity: "error",
        message: `Edge ${edge.id}: "to" is required`,
        field: `edges.${edge.id}.to`,
      });
    } else if (!nodeNames.has(edge.to) && edge.to !== "DONE") {
      errors.push({
        severity: "warning",
        message: `Edge ${edge.id}: "to" node "${edge.to}" not found`,
        field: `edges.${edge.id}.to`,
      });
    }
  }

  // Orphan nodes: no edges lead to or from them (except start)
  const connected = new Set<string>();
  for (const edge of def.edges) {
    connected.add(edge.from);
    connected.add(edge.to);
  }
  if (def.start) connected.add(def.start);
  for (const node of def.nodes) {
    if (!connected.has(node.name)) {
      errors.push({
        severity: "warning",
        message: `Orphan node: "${node.name}" has no edges`,
        field: `nodes.${node.name}`,
      });
    }
  }

  return errors;
}

export function hasErrors(errors: ValidationError[]): boolean {
  return errors.some((e) => e.severity === "error");
}
