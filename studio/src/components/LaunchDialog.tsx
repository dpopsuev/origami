import { useState } from "react";

interface PipelineVar {
  key: string;
  value: string;
}

interface LaunchDialogProps {
  pipelines: string[];
  onLaunch: (pipeline: string, vars: Record<string, string>) => void;
  onClose: () => void;
}

export function LaunchDialog({ pipelines, onLaunch, onClose }: LaunchDialogProps) {
  const [selectedPipeline, setSelectedPipeline] = useState(pipelines[0] || "");
  const [vars, setVars] = useState<PipelineVar[]>([]);
  const [launching, setLaunching] = useState(false);

  const addVar = () => setVars([...vars, { key: "", value: "" }]);

  const updateVar = (idx: number, field: "key" | "value", val: string) => {
    const next = [...vars];
    next[idx] = { ...next[idx], [field]: val };
    setVars(next);
  };

  const removeVar = (idx: number) => setVars(vars.filter((_, i) => i !== idx));

  const handleLaunch = async () => {
    setLaunching(true);
    const varsObj: Record<string, string> = {};
    for (const v of vars) {
      if (v.key.trim()) varsObj[v.key.trim()] = v.value;
    }
    onLaunch(selectedPipeline, varsObj);
  };

  return (
    <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50">
      <div className="bg-gray-900 border border-gray-700 rounded-lg shadow-xl w-[480px] max-h-[80vh] overflow-y-auto">
        <div className="flex items-center justify-between px-4 py-3 border-b border-gray-700">
          <h2 className="text-sm font-semibold">Launch Pipeline Run</h2>
          <button onClick={onClose} className="text-gray-500 hover:text-white">
            ✕
          </button>
        </div>

        <div className="p-4 space-y-4">
          <div>
            <label className="text-xs text-gray-400 block mb-1">Pipeline</label>
            <select
              value={selectedPipeline}
              onChange={(e) => setSelectedPipeline(e.target.value)}
              className="w-full bg-gray-800 border border-gray-600 rounded px-3 py-2 text-sm focus:border-blue-500 focus:outline-none"
            >
              {pipelines.map((p) => (
                <option key={p} value={p}>{p}</option>
              ))}
            </select>
          </div>

          <div>
            <div className="flex items-center justify-between mb-2">
              <label className="text-xs text-gray-400">Variables</label>
              <button
                onClick={addVar}
                className="text-xs text-blue-400 hover:text-blue-300"
              >
                + Add variable
              </button>
            </div>
            {vars.length === 0 && (
              <div className="text-xs text-gray-600">No variables configured</div>
            )}
            {vars.map((v, i) => (
              <div key={i} className="flex gap-2 mb-2">
                <input
                  value={v.key}
                  onChange={(e) => updateVar(i, "key", e.target.value)}
                  placeholder="key"
                  className="flex-1 bg-gray-800 border border-gray-600 rounded px-2 py-1 text-xs focus:border-blue-500 focus:outline-none"
                />
                <input
                  value={v.value}
                  onChange={(e) => updateVar(i, "value", e.target.value)}
                  placeholder="value"
                  className="flex-1 bg-gray-800 border border-gray-600 rounded px-2 py-1 text-xs focus:border-blue-500 focus:outline-none"
                />
                <button
                  onClick={() => removeVar(i)}
                  className="text-gray-500 hover:text-red-400 text-xs px-1"
                >
                  ✕
                </button>
              </div>
            ))}
          </div>
        </div>

        <div className="flex justify-end gap-2 px-4 py-3 border-t border-gray-700">
          <button
            onClick={onClose}
            className="px-4 py-2 text-xs text-gray-400 hover:text-white rounded border border-gray-600 hover:border-gray-500"
          >
            Cancel
          </button>
          <button
            onClick={handleLaunch}
            disabled={!selectedPipeline || launching}
            className="px-4 py-2 text-xs bg-blue-600 hover:bg-blue-500 text-white rounded disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {launching ? "Launching..." : "Launch"}
          </button>
        </div>
      </div>
    </div>
  );
}
