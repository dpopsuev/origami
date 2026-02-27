interface MetricPoint {
  timestamp: string;
  value: number;
}

interface NodeAnalytics {
  nodeId: string;
  p50Ms: number;
  p95Ms: number;
  p99Ms: number;
  totalTokenCost: number;
  errorRate: number;
  walkerEfficiency: number;
}

interface AnalyticsDashboardProps {
  nodeMetrics: NodeAnalytics[];
  costTrend: MetricPoint[];
  errorTrend: MetricPoint[];
  dateRange: { from: string; to: string };
  onDateRangeChange: (from: string, to: string) => void;
}

export function AnalyticsDashboard({
  nodeMetrics,
  costTrend,
  errorTrend,
  dateRange,
  onDateRangeChange,
}: AnalyticsDashboardProps) {
  const totalCost = nodeMetrics.reduce((sum, m) => sum + m.totalTokenCost, 0);
  const avgP95 =
    nodeMetrics.length > 0
      ? nodeMetrics.reduce((sum, m) => sum + m.p95Ms, 0) / nodeMetrics.length
      : 0;
  const avgErrorRate =
    nodeMetrics.length > 0
      ? nodeMetrics.reduce((sum, m) => sum + m.errorRate, 0) / nodeMetrics.length
      : 0;

  return (
    <div className="p-4 space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-sm font-semibold text-gray-300 uppercase tracking-wide">
          Pipeline Analytics
        </h2>
        <div className="flex gap-2 text-xs">
          <input
            type="date"
            value={dateRange.from}
            onChange={(e) => onDateRangeChange(e.target.value, dateRange.to)}
            className="bg-gray-800 border border-gray-600 rounded px-2 py-1 text-[10px]"
          />
          <input
            type="date"
            value={dateRange.to}
            onChange={(e) => onDateRangeChange(dateRange.from, e.target.value)}
            className="bg-gray-800 border border-gray-600 rounded px-2 py-1 text-[10px]"
          />
        </div>
      </div>

      <div className="grid grid-cols-3 gap-3">
        <SummaryCard label="Avg P95 Latency" value={`${avgP95.toFixed(0)}ms`} />
        <SummaryCard label="Total Token Cost" value={`$${totalCost.toFixed(4)}`} />
        <SummaryCard label="Avg Error Rate" value={`${(avgErrorRate * 100).toFixed(1)}%`} />
      </div>

      <div className="grid grid-cols-2 gap-3">
        <MiniChart title="Cost Trend" points={costTrend} color="#3b82f6" />
        <MiniChart title="Error Trend" points={errorTrend} color="#ef4444" />
      </div>

      <div>
        <h3 className="text-xs text-gray-400 font-medium mb-2">Per-Node Metrics</h3>
        <table className="w-full text-xs">
          <thead>
            <tr className="text-gray-500 border-b border-gray-700">
              <th className="text-left py-1.5">Node</th>
              <th className="text-right py-1.5">P50</th>
              <th className="text-right py-1.5">P95</th>
              <th className="text-right py-1.5">P99</th>
              <th className="text-right py-1.5">Cost</th>
              <th className="text-right py-1.5">Errors</th>
            </tr>
          </thead>
          <tbody>
            {nodeMetrics.map((m) => (
              <tr key={m.nodeId} className="border-b border-gray-800">
                <td className="py-1.5 font-mono">{m.nodeId}</td>
                <td className="text-right text-gray-400">{m.p50Ms}ms</td>
                <td className="text-right text-gray-400">{m.p95Ms}ms</td>
                <td className="text-right text-gray-400">{m.p99Ms}ms</td>
                <td className="text-right text-gray-400">${m.totalTokenCost.toFixed(4)}</td>
                <td className={`text-right ${m.errorRate > 0.05 ? "text-red-400" : "text-gray-400"}`}>
                  {(m.errorRate * 100).toFixed(1)}%
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

function SummaryCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="bg-gray-800 rounded-lg p-3 border border-gray-700">
      <div className="text-[10px] text-gray-500 uppercase tracking-wide">{label}</div>
      <div className="text-lg font-semibold mt-1">{value}</div>
    </div>
  );
}

function MiniChart({
  title,
  points,
  color,
}: {
  title: string;
  points: MetricPoint[];
  color: string;
}) {
  if (points.length === 0) {
    return (
      <div className="bg-gray-800 rounded-lg p-3 border border-gray-700">
        <div className="text-[10px] text-gray-500">{title}</div>
        <div className="text-xs text-gray-600 mt-2">No data</div>
      </div>
    );
  }

  const max = Math.max(...points.map((p) => p.value));
  const height = 40;

  return (
    <div className="bg-gray-800 rounded-lg p-3 border border-gray-700">
      <div className="text-[10px] text-gray-500 mb-2">{title}</div>
      <svg width="100%" height={height} viewBox={`0 0 ${points.length} ${height}`} preserveAspectRatio="none">
        <polyline
          fill="none"
          stroke={color}
          strokeWidth="1.5"
          points={points
            .map((p, i) => `${i},${height - (max > 0 ? (p.value / max) * height : 0)}`)
            .join(" ")}
        />
      </svg>
    </div>
  );
}
