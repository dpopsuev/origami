import { useState } from "react";
import type { CircuitWalker } from "../sync/yaml-sync";

interface WalkerEditorProps {
  walkers: CircuitWalker[];
  onUpdate: (walkers: CircuitWalker[]) => void;
}

const ELEMENTS = ["fire", "water", "earth", "air", "diamond", "lightning"];
const PERSONAS = [
  "herald", "seeker", "sentinel", "weaver",
  "arbiter", "catalyst", "oracle", "phantom",
];

export function WalkerEditor({ walkers, onUpdate }: WalkerEditorProps) {
  const [editing, setEditing] = useState<string | null>(null);

  const addWalker = () => {
    const name = `walker-${walkers.length + 1}`;
    onUpdate([...walkers, { name, element: "fire", persona: "herald" }]);
    setEditing(name);
  };

  const removeWalker = (name: string) => {
    onUpdate(walkers.filter((w) => w.name !== name));
    if (editing === name) setEditing(null);
  };

  const updateWalker = (name: string, updates: Partial<CircuitWalker>) => {
    onUpdate(walkers.map((w) => (w.name === name ? { ...w, ...updates } : w)));
  };

  return (
    <div className="p-3 space-y-3">
      <div className="flex justify-between items-center">
        <h3 className="text-xs font-semibold text-gray-400 uppercase tracking-wide">
          Walkers
        </h3>
        <button
          onClick={addWalker}
          className="text-xs bg-gray-700 hover:bg-gray-600 px-2 py-0.5 rounded transition-colors"
        >
          + Add
        </button>
      </div>

      {walkers.map((walker) => (
        <div
          key={walker.name}
          className="bg-gray-800 rounded p-2 border border-gray-700"
        >
          <div className="flex justify-between items-center mb-1">
            <span className="text-xs font-medium">{walker.name}</span>
            <div className="flex gap-1">
              <button
                onClick={() => setEditing(editing === walker.name ? null : walker.name)}
                className="text-[10px] text-gray-400 hover:text-white"
              >
                {editing === walker.name ? "▲" : "▼"}
              </button>
              <button
                onClick={() => removeWalker(walker.name)}
                className="text-[10px] text-red-400 hover:text-red-300"
              >
                ✕
              </button>
            </div>
          </div>

          {editing === walker.name && (
            <div className="space-y-2 mt-2">
              <div>
                <label className="text-[10px] text-gray-500">Name</label>
                <input
                  value={walker.name}
                  onChange={(e) => updateWalker(walker.name, { name: e.target.value })}
                  className="w-full bg-gray-900 border border-gray-600 rounded px-1 py-0.5 text-xs"
                />
              </div>
              <div>
                <label className="text-[10px] text-gray-500">Element</label>
                <select
                  value={walker.element || ""}
                  onChange={(e) => updateWalker(walker.name, { element: e.target.value })}
                  className="w-full bg-gray-900 border border-gray-600 rounded px-1 py-0.5 text-xs"
                >
                  {ELEMENTS.map((el) => (
                    <option key={el} value={el}>{el}</option>
                  ))}
                </select>
              </div>
              <div>
                <label className="text-[10px] text-gray-500">Persona</label>
                <select
                  value={walker.persona || ""}
                  onChange={(e) => updateWalker(walker.name, { persona: e.target.value })}
                  className="w-full bg-gray-900 border border-gray-600 rounded px-1 py-0.5 text-xs"
                >
                  {PERSONAS.map((p) => (
                    <option key={p} value={p}>{p}</option>
                  ))}
                </select>
              </div>
              <div>
                <label className="text-[10px] text-gray-500">Preamble</label>
                <textarea
                  value={walker.preamble || ""}
                  onChange={(e) => updateWalker(walker.name, { preamble: e.target.value })}
                  className="w-full bg-gray-900 border border-gray-600 rounded px-1 py-0.5 text-xs h-16 resize-none"
                  placeholder="Walker preamble..."
                />
              </div>
            </div>
          )}

          <div className="text-[10px] text-gray-500">
            {walker.element || "?"} · {walker.persona || "?"}
          </div>
        </div>
      ))}
    </div>
  );
}
