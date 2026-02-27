import { useEffect, useState } from "react";

interface ComparisonEvent {
  id: number;
  type: string;
  node?: string;
  elapsed_ms?: number;
}

interface RunComparisonProps {
  leftRunId: string;
  rightRunId: string;
  onClose: () => void;
}

interface NodeCompare {
  nodeId: string;
  leftDurationMs?: number;
  rightDurationMs?: number;
  leftVisits: number;
  rightVisits: number;
  delta: number;
}

function aggregateByNode(events: ComparisonEvent[]): Map<string, { totalMs: number; visits: number }> {
  const map = new Map<string, { totalMs: number; visits: number }>();
  for (const e of events) {
    if (e.node && e.type === "node_exit") {
      const existing = map.get(e.node) || { totalMs: 0, visits: 0 };
      existing.totalMs += e.elapsed_ms || 0;
      existing.visits++;
      map.set(e.node, existing);
    }
  }
  return map;
}

export function RunComparison({ leftRunId, rightRunId, onClose }: RunComparisonProps) {
  const [comparisons, setComparisons] = useState<NodeCompare[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchBoth = async () => {
      try {
        const [leftResp, rightResp] = await Promise.all([
          fetch(`/api/runs/${leftRunId}/events`),
          fetch(`/api/runs/${rightRunId}/events`),
        ]);

        const leftEvents: ComparisonEvent[] = leftResp.ok ? await leftResp.json() : [];
        const rightEvents: ComparisonEvent[] = rightResp.ok ? await rightResp.json() : [];

        const leftAgg = aggregateByNode(leftEvents);
        const rightAgg = aggregateByNode(rightEvents);

        const allNodes = new Set([...leftAgg.keys(), ...rightAgg.keys()]);
        const results: NodeCompare[] = [];

        for (const nodeId of allNodes) {
          const left = leftAgg.get(nodeId);
          const right = rightAgg.get(nodeId);
          const leftAvg = left ? left.totalMs / left.visits : 0;
          const rightAvg = right ? right.totalMs / right.visits : 0;

          results.push({
            nodeId,
            leftDurationMs: left ? leftAvg : undefined,
            rightDurationMs: right ? rightAvg : undefined,
            leftVisits: left?.visits || 0,
            rightVisits: right?.visits || 0,
            delta: rightAvg - leftAvg,
          });
        }

        results.sort((a, b) => Math.abs(b.delta) - Math.abs(a.delta));
        setComparisons(results);
      } catch {
        // API not available
      } finally {
        setLoading(false);
      }
    };

    fetchBoth();
  }, [leftRunId, rightRunId]);

  return (
    <div className="p-4 overflow-auto h-full">
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-sm font-semibold text-gray-300 uppercase tracking-wide">
          Run Comparison
        </h2>
        <button onClick={onClose} className="text-gray-500 hover:text-white text-sm">
          ✕
        </button>
      </div>

      <div className="text-xs text-gray-500 mb-3">
        {leftRunId} vs {rightRunId}
      </div>

      {loading ? (
        <div className="text-gray-400">Loading...</div>
      ) : comparisons.length === 0 ? (
        <div className="text-gray-500">No comparable data</div>
      ) : (
        <table className="w-full text-xs">
          <thead>
            <tr className="text-gray-500 border-b border-gray-700">
              <th className="text-left py-2">Node</th>
              <th className="text-right py-2">Left (ms)</th>
              <th className="text-right py-2">Right (ms)</th>
              <th className="text-right py-2">Delta</th>
            </tr>
          </thead>
          <tbody>
            {comparisons.map((c) => (
              <tr key={c.nodeId} className="border-b border-gray-800">
                <td className="py-1.5 font-mono">{c.nodeId}</td>
                <td className="text-right text-gray-400">
                  {c.leftDurationMs != null ? `${c.leftDurationMs.toFixed(0)}` : "—"}
                </td>
                <td className="text-right text-gray-400">
                  {c.rightDurationMs != null ? `${c.rightDurationMs.toFixed(0)}` : "—"}
                </td>
                <td
                  className={`text-right ${
                    c.delta > 0
                      ? "text-red-400"
                      : c.delta < 0
                      ? "text-green-400"
                      : "text-gray-500"
                  }`}
                >
                  {c.delta > 0 ? "+" : ""}
                  {c.delta.toFixed(0)}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}
