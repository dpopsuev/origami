import type { PipelineDef } from "./yaml-sync";
import { toYAML } from "./yaml-sync";

export function exportAsYAML(def: PipelineDef): string {
  return toYAML(def);
}

export function exportAsJSON(def: PipelineDef): string {
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
  def: PipelineDef,
  format: "yaml" | "json" = "yaml"
) {
  const name = def.pipeline || "pipeline";
  if (format === "json") {
    downloadFile(exportAsJSON(def), `${name}.json`, "application/json");
  } else {
    downloadFile(exportAsYAML(def), `${name}.yaml`, "text/yaml");
  }
}
