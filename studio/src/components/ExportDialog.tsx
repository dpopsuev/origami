import { useState } from "react";

type ExportFormat = "svg" | "png" | "mermaid" | "yaml";

interface ExportDialogProps {
  pipelineName: string;
  onExport: (format: ExportFormat) => void;
  onClose: () => void;
}

const FORMATS: { id: ExportFormat; label: string; description: string }[] = [
  { id: "svg", label: "SVG", description: "Editable vector — design tools, docs" },
  { id: "png", label: "PNG", description: "Raster image — includes current overlay state" },
  { id: "mermaid", label: "Mermaid", description: "Pasteable diagram code for markdown" },
  { id: "yaml", label: "YAML", description: "Pipeline definition file" },
];

export function ExportDialog({ pipelineName, onExport, onClose }: ExportDialogProps) {
  const [selectedFormat, setSelectedFormat] = useState<ExportFormat>("svg");

  return (
    <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50">
      <div className="bg-gray-900 border border-gray-700 rounded-lg shadow-xl w-[360px]">
        <div className="flex items-center justify-between px-4 py-3 border-b border-gray-700">
          <h2 className="text-sm font-semibold">Export: {pipelineName}</h2>
          <button onClick={onClose} className="text-gray-500 hover:text-white">
            ✕
          </button>
        </div>

        <div className="p-4 space-y-2">
          {FORMATS.map((f) => (
            <button
              key={f.id}
              onClick={() => setSelectedFormat(f.id)}
              className={`w-full text-left p-3 rounded-lg border transition-colors ${
                selectedFormat === f.id
                  ? "border-blue-500/30 bg-blue-900/20"
                  : "border-gray-700 bg-gray-800 hover:border-gray-600"
              }`}
            >
              <div className="text-sm font-medium">{f.label}</div>
              <div className="text-[10px] text-gray-500">{f.description}</div>
            </button>
          ))}
        </div>

        <div className="flex justify-end gap-2 px-4 py-3 border-t border-gray-700">
          <button
            onClick={onClose}
            className="px-4 py-2 text-xs text-gray-400 hover:text-white rounded border border-gray-600"
          >
            Cancel
          </button>
          <button
            onClick={() => onExport(selectedFormat)}
            className="px-4 py-2 text-xs bg-blue-600 hover:bg-blue-500 text-white rounded"
          >
            Export
          </button>
        </div>
      </div>
    </div>
  );
}
