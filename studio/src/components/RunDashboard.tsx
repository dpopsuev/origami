import { useEffect, useState } from "react";

interface RunInfo {
  id: string;
  pipeline: string;
  started_at: string;
  ended_at?: string;
  status: string;
  node_count: number;
  edge_count: number;
}

interface RunDashboardProps {
  onSelectRun: (runId: string) => void;
}

export function RunDashboard({ onSelectRun }: RunDashboardProps) {
  const [runs, setRuns] = useState<RunInfo[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchRuns = async () => {
      try {
        const resp = await fetch("/api/runs");
        if (resp.ok) {
          const data = await resp.json();
          setRuns(data || []);
        }
      } catch {
        // API not available
      } finally {
        setLoading(false);
      }
    };
    fetchRuns();
    const interval = setInterval(fetchRuns, 5000);
    return () => clearInterval(interval);
  }, []);

  const statusColor = (status: string) => {
    switch (status) {
      case "running":
        return "text-green-400";
      case "completed":
        return "text-blue-400";
      case "error":
        return "text-red-400";
      default:
        return "text-gray-400";
    }
  };

  if (loading) {
    return (
      <div className="p-4 text-gray-400">Loading runs...</div>
    );
  }

  if (runs.length === 0) {
    return (
      <div className="p-4 text-gray-500">
        No pipeline runs yet. Start a pipeline to see runs here.
      </div>
    );
  }

  return (
    <div className="p-4 space-y-2">
      <h2 className="text-sm font-semibold text-gray-300 uppercase tracking-wide mb-3">
        Pipeline Runs
      </h2>
      {runs.map((run) => (
        <button
          key={run.id}
          onClick={() => onSelectRun(run.id)}
          className="w-full text-left p-3 rounded-lg bg-gray-800 hover:bg-gray-700 transition-colors border border-gray-700"
        >
          <div className="flex justify-between items-center">
            <span className="font-mono text-sm">{run.pipeline}</span>
            <span className={`text-xs font-medium ${statusColor(run.status)}`}>
              {run.status}
            </span>
          </div>
          <div className="text-xs text-gray-500 mt-1">
            {run.node_count} nodes · {run.edge_count} edges ·{" "}
            {new Date(run.started_at).toLocaleTimeString()}
          </div>
        </button>
      ))}
    </div>
  );
}
