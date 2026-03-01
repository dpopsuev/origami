import type { CircuitDef } from "./yaml-sync";
import { toYAML } from "./yaml-sync";

export function exportAsYAML(def: CircuitDef): string {
  return toYAML(def);
}

export function exportAsJSON(def: CircuitDef): string {
  return JSON.stringify(def, null, 2);
}

export function downloadFile(content: string, filename: string, mimeType: string) {
  const blob = new Blob([content], { type: mimeType });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  URL.revokeObjectURL(url);
}

export function exportAndDownload(
  def: CircuitDef,
  format: "yaml" | "json" = "yaml"
) {
  const name = def.circuit || "circuit";
  if (format === "json") {
    downloadFile(exportAsJSON(def), `${name}.json`, "application/json");
  } else {
    downloadFile(exportAsYAML(def), `${name}.yaml`, "text/yaml");
  }
}
