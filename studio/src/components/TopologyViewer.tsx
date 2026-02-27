interface WorkerNode {
  id: string;
  zone: string;
  provider: string;
  status: "idle" | "busy" | "error" | "offline";
  currentCase?: string;
  stepsCompleted: number;
}

interface TopologyViewerProps {
  workers: WorkerNode[];
  zones: string[];
}

const STATUS_COLORS: Record<string, string> = {
  idle: "#22c55e",
  busy: "#3b82f6",
  error: "#ef4444",
  offline: "#6b7280",
};

export function TopologyViewer({ workers, zones }: TopologyViewerProps) {
  const workersByZone = new Map<string, WorkerNode[]>();
  for (const w of workers) {
    const list = workersByZone.get(w.zone) || [];
    list.push(w);
    workersByZone.set(w.zone, list);
  }

  return (
    <div className="p-4">
      <div className="flex items-center gap-2 mb-4">
        <h2 className="text-sm font-semibold text-gray-300 uppercase tracking-wide">
          Execution Topology
        </h2>
        <span className="text-[10px] bg-purple-900 text-purple-300 px-2 py-0.5 rounded-full">
          Enterprise
        </span>
      </div>

      <div className="grid gap-4">
        {zones.map((zone) => (
          <div
            key={zone}
            className="bg-gray-800 rounded-lg border border-gray-700 p-3"
          >
            <div className="text-xs text-gray-400 font-medium mb-2">
              Zone: {zone}
            </div>
            <div className="grid grid-cols-3 gap-2">
              {(workersByZone.get(zone) || []).map((w) => (
                <div
                  key={w.id}
                  className="flex items-center gap-2 p-2 bg-gray-900 rounded text-xs"
                >
                  <div
                    className="w-2 h-2 rounded-full shrink-0"
                    style={{ backgroundColor: STATUS_COLORS[w.status] }}
                  />
                  <div className="min-w-0">
                    <div className="font-mono truncate">{w.id}</div>
                    <div className="text-[10px] text-gray-500">
                      {w.status}
                      {w.currentCase ? ` · ${w.currentCase}` : ""}
                      {w.stepsCompleted > 0 ? ` · ${w.stepsCompleted} steps` : ""}
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </div>
        ))}
      </div>

      <div className="flex gap-4 mt-4 text-[10px] text-gray-500">
        {Object.entries(STATUS_COLORS).map(([status, color]) => (
          <div key={status} className="flex items-center gap-1">
            <div className="w-2 h-2 rounded-full" style={{ backgroundColor: color }} />
            {status}
          </div>
        ))}
      </div>
    </div>
  );
}
