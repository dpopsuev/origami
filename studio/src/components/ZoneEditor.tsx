import { useState } from "react";
import type { PipelineZone, PipelineNode } from "../sync/yaml-sync";

interface ZoneEditorProps {
  zones: PipelineZone[];
  availableNodes: PipelineNode[];
  onUpdate: (zones: PipelineZone[]) => void;
}

const ELEMENTS = ["fire", "water", "earth", "air", "diamond", "lightning", "iron"];

export function ZoneEditor({ zones, availableNodes, onUpdate }: ZoneEditorProps) {
  const [editingZone, setEditingZone] = useState<string | null>(null);

  const addZone = () => {
    const name = `zone-${zones.length + 1}`;
    onUpdate([...zones, { name, nodes: [], element: "fire" }]);
    setEditingZone(name);
  };

  const removeZone = (name: string) => {
    onUpdate(zones.filter((z) => z.name !== name));
    if (editingZone === name) setEditingZone(null);
  };

  const updateZone = (name: string, updates: Partial<PipelineZone>) => {
    onUpdate(zones.map((z) => (z.name === name ? { ...z, ...updates } : z)));
  };

  const toggleNodeInZone = (zoneName: string, nodeName: string) => {
    const zone = zones.find((z) => z.name === zoneName);
    if (!zone) return;
    const nodes = zone.nodes.includes(nodeName)
      ? zone.nodes.filter((n) => n !== nodeName)
      : [...zone.nodes, nodeName];
    updateZone(zoneName, { nodes });
  };

  return (
    <div className="p-3 space-y-3">
      <div className="flex justify-between items-center">
        <h3 className="text-xs font-semibold text-gray-400 uppercase tracking-wide">
          Zones
        </h3>
        <button
          onClick={addZone}
          className="text-xs bg-gray-700 hover:bg-gray-600 px-2 py-0.5 rounded transition-colors"
        >
          + Add
        </button>
      </div>

      {zones.map((zone) => (
        <div key={zone.name} className="bg-gray-800 rounded p-2 border border-gray-700">
          <div className="flex justify-between items-center mb-2">
            <input
              value={zone.name}
              onChange={(e) => updateZone(zone.name, { name: e.target.value })}
              className="bg-transparent border-b border-gray-600 text-xs font-medium focus:outline-none"
            />
            <div className="flex gap-1">
              <button
                onClick={() => setEditingZone(editingZone === zone.name ? null : zone.name)}
                className="text-[10px] text-gray-400 hover:text-white"
              >
                {editingZone === zone.name ? "▲" : "▼"}
              </button>
              <button
                onClick={() => removeZone(zone.name)}
                className="text-[10px] text-red-400 hover:text-red-300"
              >
                ✕
              </button>
            </div>
          </div>

          {editingZone === zone.name && (
            <div className="space-y-2">
              <div>
                <label className="text-[10px] text-gray-500">Element</label>
                <select
                  value={zone.element || ""}
                  onChange={(e) => updateZone(zone.name, { element: e.target.value })}
                  className="w-full bg-gray-900 border border-gray-600 rounded px-1 py-0.5 text-xs"
                >
                  <option value="">None</option>
                  {ELEMENTS.map((el) => (
                    <option key={el} value={el}>{el}</option>
                  ))}
                </select>
              </div>

              <div>
                <label className="text-[10px] text-gray-500">Stickiness</label>
                <input
                  type="number"
                  min={0}
                  max={3}
                  value={zone.stickiness || 0}
                  onChange={(e) => updateZone(zone.name, { stickiness: parseInt(e.target.value) })}
                  className="w-full bg-gray-900 border border-gray-600 rounded px-1 py-0.5 text-xs"
                />
              </div>

              <div>
                <label className="text-[10px] text-gray-500">Nodes</label>
                <div className="flex flex-wrap gap-1 mt-1">
                  {availableNodes.map((node) => (
                    <button
                      key={node.name}
                      onClick={() => toggleNodeInZone(zone.name, node.name)}
                      className={`text-[10px] px-1.5 py-0.5 rounded ${
                        zone.nodes.includes(node.name)
                          ? "bg-blue-800 text-blue-200"
                          : "bg-gray-700 text-gray-400"
                      }`}
                    >
                      {node.name}
                    </button>
                  ))}
                </div>
              </div>
            </div>
          )}

          <div className="text-[10px] text-gray-500 mt-1">
            {zone.nodes.length} nodes · {zone.element || "no element"}
          </div>
        </div>
      ))}
    </div>
  );
}
