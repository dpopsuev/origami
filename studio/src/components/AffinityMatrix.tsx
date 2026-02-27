const ELEMENT_COLORS: Record<string, string> = {
  fire: "#DC143C",
  water: "#007BA7",
  earth: "#0047AB",
  air: "#FFBF00",
  diamond: "#0F52BA",
  lightning: "#DC143C",
  iron: "#48494B",
};

interface AffinityScore {
  walkerId: string;
  walkerElement: string;
  nodeId: string;
  score: number;
}

interface AffinityMatrixProps {
  walkers: Array<{ id: string; element: string; persona: string }>;
  nodes: string[];
  scores: AffinityScore[];
}

export function AffinityMatrix({ walkers, nodes, scores }: AffinityMatrixProps) {
  const scoreMap = new Map<string, number>();
  for (const s of scores) {
    scoreMap.set(`${s.walkerId}:${s.nodeId}`, s.score);
  }

  const cellColor = (score: number) => {
    if (score >= 0.8) return "bg-green-900/60";
    if (score >= 0.5) return "bg-yellow-900/40";
    if (score >= 0.3) return "bg-orange-900/30";
    return "bg-red-900/20";
  };

  return (
    <div className="p-4 overflow-auto">
      <h2 className="text-sm font-semibold text-gray-300 uppercase tracking-wide mb-3">
        Walker-Node Affinity Matrix
      </h2>

      <div className="overflow-x-auto">
        <table className="text-xs">
          <thead>
            <tr>
              <th className="p-2 text-left text-gray-500 sticky left-0 bg-gray-900 z-10">
                Walker
              </th>
              {nodes.map((n) => (
                <th key={n} className="p-2 text-center text-gray-500 font-mono">
                  {n}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {walkers.map((w) => (
              <tr key={w.id}>
                <td className="p-2 sticky left-0 bg-gray-900 z-10">
                  <div className="flex items-center gap-1.5">
                    <div
                      className="w-2 h-2 rounded-full shrink-0"
                      style={{ backgroundColor: ELEMENT_COLORS[w.element] || "#555" }}
                    />
                    <span className="font-mono">{w.id}</span>
                    <span className="text-gray-600 text-[10px]">({w.persona})</span>
                  </div>
                </td>
                {nodes.map((n) => {
                  const score = scoreMap.get(`${w.id}:${n}`) ?? 0;
                  return (
                    <td
                      key={n}
                      className={`p-2 text-center font-mono ${cellColor(score)}`}
                      title={`${w.id} → ${n}: ${(score * 100).toFixed(0)}%`}
                    >
                      {(score * 100).toFixed(0)}
                    </td>
                  );
                })}
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <div className="flex gap-3 mt-3 text-[10px] text-gray-500">
        <span className="flex items-center gap-1">
          <span className="w-3 h-3 bg-green-900/60 rounded" /> {">="}80%
        </span>
        <span className="flex items-center gap-1">
          <span className="w-3 h-3 bg-yellow-900/40 rounded" /> 50-79%
        </span>
        <span className="flex items-center gap-1">
          <span className="w-3 h-3 bg-orange-900/30 rounded" /> 30-49%
        </span>
        <span className="flex items-center gap-1">
          <span className="w-3 h-3 bg-red-900/20 rounded" /> {"<"}30%
        </span>
      </div>
    </div>
  );
}
