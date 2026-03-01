import { useState } from "react";
import type { CircuitEdge } from "../sync/yaml-sync";

interface EdgeConditionBuilderProps {
  edge: CircuitEdge;
  onUpdate: (updates: Partial<CircuitEdge>) => void;
  onClose: () => void;
}

const PRESETS = [
  { label: "Always", when: "true" },
  { label: "Output confidence > threshold", when: 'output.confidence > 0.8' },
  { label: "Has error", when: "output.error != nil" },
  { label: "Loop count < N", when: "state.loops.{node} < 3" },
  { label: "Custom expression", when: "" },
];

export function EdgeConditionBuilder({
  edge,
  onUpdate,
  onClose,
}: EdgeConditionBuilderProps) {
  const [when, setWhen] = useState(edge.when || "true");
  const [name, setName] = useState(edge.name || "");
  const [isShortcut, setIsShortcut] = useState(edge.shortcut || false);
  const [isLoop, setIsLoop] = useState(edge.loop || false);

  const handleApply = () => {
    onUpdate({
      when,
      name: name || undefined,
      shortcut: isShortcut || undefined,
      loop: isLoop || undefined,
    });
    onClose();
  };

  return (
    <div className="p-4 bg-gray-800 rounded-lg border border-gray-600 w-80 space-y-3">
      <div className="flex justify-between items-center">
        <h3 className="text-sm font-semibold">Edge: {edge.id}</h3>
        <button
          onClick={onClose}
          className="text-gray-400 hover:text-white text-xs"
        >
          ✕
        </button>
      </div>

      <div>
        <label className="block text-xs text-gray-400 mb-1">Name</label>
        <input
          value={name}
          onChange={(e) => setName(e.target.value)}
          className="w-full bg-gray-900 border border-gray-600 rounded px-2 py-1 text-xs"
          placeholder="Edge name"
        />
      </div>

      <div>
        <label className="block text-xs text-gray-400 mb-1">Condition (when)</label>
        <div className="space-y-1 mb-2">
          {PRESETS.map((p) => (
            <button
              key={p.label}
              onClick={() => setWhen(p.when || when)}
              className={`block w-full text-left text-xs px-2 py-1 rounded ${
                when === p.when ? "bg-blue-900 text-blue-300" : "hover:bg-gray-700"
              }`}
            >
              {p.label}
            </button>
          ))}
        </div>
        <input
          value={when}
          onChange={(e) => setWhen(e.target.value)}
          className="w-full bg-gray-900 border border-gray-600 rounded px-2 py-1 text-xs font-mono"
          placeholder='Expression, e.g. output.confidence > 0.8'
        />
      </div>

      <div className="flex gap-4">
        <label className="flex items-center gap-1 text-xs">
          <input
            type="checkbox"
            checked={isShortcut}
            onChange={(e) => setIsShortcut(e.target.checked)}
          />
          Shortcut
        </label>
        <label className="flex items-center gap-1 text-xs">
          <input
            type="checkbox"
            checked={isLoop}
            onChange={(e) => setIsLoop(e.target.checked)}
          />
          Loop
        </label>
      </div>

      <button
        onClick={handleApply}
        className="w-full bg-blue-600 hover:bg-blue-500 text-white text-xs py-1.5 rounded transition-colors"
      >
        Apply
      </button>
    </div>
  );
}
