import { useState } from "react";
import type { NodeMetrics } from "../overlays/types";

type InspectorTab = "config" | "runs" | "artifacts" | "metrics";

interface NodeInspectorProps {
  nodeId: string;
  nodeData: Record<string, unknown>;
  metrics?: NodeMetrics;
  yamlSource?: string;
  artifacts?: { input?: string; output?: string };
  onClose: () => void;
}

export function NodeInspector({
  nodeId,
  nodeData,
  metrics,
  yamlSource,
  artifacts,
  onClose,
}: NodeInspectorProps) {
  const [activeTab, setActiveTab] = useState<InspectorTab>("config");

  const tabs: { id: InspectorTab; label: string }[] = [
    { id: "config", label: "Config" },
    { id: "runs", label: "Runs" },
    { id: "artifacts", label: "Artifacts" },
    { id: "metrics", label: "Metrics" },
  ];

  return (
    <div className="w-80 bg-gray-900 border-l border-gray-700 flex flex-col h-full overflow-hidden">
      <div className="flex items-center justify-between px-3 py-2 border-b border-gray-700">
        <div className="flex items-center gap-2">
          <div
            className="w-2.5 h-2.5 rounded-full"
            style={{
              backgroundColor:
                (nodeData.element as string)
                  ? ELEMENT_COLORS[(nodeData.element as string)] || "#555"
                  : "#555",
            }}
          />
          <span className="text-sm font-medium">{nodeId}</span>
        </div>
        <button
          onClick={onClose}
          className="text-gray-500 hover:text-white text-sm px-1"
        >
          ✕
        </button>
      </div>

      <div className="flex border-b border-gray-700">
        {tabs.map((tab) => (
          <button
            key={tab.id}
            onClick={() => setActiveTab(tab.id)}
            className={`flex-1 text-xs py-2 transition-colors ${
              activeTab === tab.id
                ? "text-white border-b-2 border-blue-500"
                : "text-gray-500 hover:text-gray-300"
            }`}
          >
            {tab.label}
          </button>
        ))}
      </div>

      <div className="flex-1 overflow-y-auto p-3 text-xs">
        {activeTab === "config" && (
          <ConfigTab nodeData={nodeData} yamlSource={yamlSource} />
        )}
        {activeTab === "runs" && <RunsTab metrics={metrics} />}
        {activeTab === "artifacts" && <ArtifactsTab artifacts={artifacts} />}
        {activeTab === "metrics" && <MetricsTab metrics={metrics} />}
      </div>
    </div>
  );
}

const ELEMENT_COLORS: Record<string, string> = {
  fire: "#DC143C",
  water: "#007BA7",
  earth: "#0047AB",
  air: "#FFBF00",
  diamond: "#0F52BA",
  lightning: "#DC143C",
  iron: "#48494B",
};

function ConfigTab({
  nodeData,
  yamlSource,
}: {
  nodeData: Record<string, unknown>;
  yamlSource?: string;
}) {
  return (
    <div className="space-y-3">
      <div>
        <div className="text-gray-400 font-medium mb-1">Properties</div>
        <div className="space-y-1">
          {Object.entries(nodeData).map(([key, value]) => (
            <div key={key} className="flex justify-between">
              <span className="text-gray-500">{key}</span>
              <span className="text-gray-300 font-mono">
                {String(value)}
              </span>
            </div>
          ))}
        </div>
      </div>
      {yamlSource && (
        <div>
          <div className="text-gray-400 font-medium mb-1">YAML Source</div>
          <pre className="bg-gray-800 p-2 rounded text-gray-300 overflow-x-auto whitespace-pre text-[11px]">
            {yamlSource}
          </pre>
        </div>
      )}
    </div>
  );
}

function RunsTab({ metrics }: { metrics?: NodeMetrics }) {
  if (!metrics) {
    return <div className="text-gray-500">No run data available</div>;
  }

  return (
    <div className="space-y-2">
      <div className="flex justify-between">
        <span className="text-gray-500">Total visits</span>
        <span>{metrics.visitCount}</span>
      </div>
      <div className="flex justify-between">
        <span className="text-gray-500">Errors</span>
        <span className={metrics.errorCount > 0 ? "text-red-400" : ""}>
          {metrics.errorCount}
        </span>
      </div>
      {metrics.lastVisitedAt && (
        <div className="flex justify-between">
          <span className="text-gray-500">Last visited</span>
          <span>{metrics.lastVisitedAt}</span>
        </div>
      )}
      {metrics.walkers.length > 0 && (
        <div>
          <div className="text-gray-500 mb-1">Walkers</div>
          <div className="flex flex-wrap gap-1">
            {metrics.walkers.map((w) => (
              <span
                key={w}
                className="bg-gray-800 px-2 py-0.5 rounded text-gray-300"
              >
                {w}
              </span>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

function ArtifactsTab({
  artifacts,
}: {
  artifacts?: { input?: string; output?: string };
}) {
  if (!artifacts) {
    return <div className="text-gray-500">No artifacts available</div>;
  }

  return (
    <div className="space-y-3">
      {artifacts.input && (
        <div>
          <div className="text-gray-400 font-medium mb-1">Input</div>
          <pre className="bg-gray-800 p-2 rounded text-gray-300 overflow-x-auto whitespace-pre text-[11px]">
            {artifacts.input}
          </pre>
        </div>
      )}
      {artifacts.output && (
        <div>
          <div className="text-gray-400 font-medium mb-1">Output</div>
          <pre className="bg-gray-800 p-2 rounded text-gray-300 overflow-x-auto whitespace-pre text-[11px]">
            {artifacts.output}
          </pre>
        </div>
      )}
    </div>
  );
}

function MetricsTab({ metrics }: { metrics?: NodeMetrics }) {
  if (!metrics) {
    return <div className="text-gray-500">No metrics available</div>;
  }

  const successRate =
    metrics.visitCount > 0
      ? ((1 - metrics.errorCount / metrics.visitCount) * 100).toFixed(1)
      : "N/A";

  return (
    <div className="space-y-3">
      <div>
        <div className="text-gray-400 font-medium mb-1">Performance</div>
        <div className="space-y-1">
          <div className="flex justify-between">
            <span className="text-gray-500">Avg duration</span>
            <span>{metrics.avgDurationMs}ms</span>
          </div>
          <div className="flex justify-between">
            <span className="text-gray-500">Success rate</span>
            <span>{successRate}%</span>
          </div>
          <div className="flex justify-between">
            <span className="text-gray-500">Visit count</span>
            <span>{metrics.visitCount}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-gray-500">Error count</span>
            <span className={metrics.errorCount > 0 ? "text-red-400" : ""}>
              {metrics.errorCount}
            </span>
          </div>
        </div>
      </div>

      {metrics.walkers.length > 0 && (
        <div>
          <div className="text-gray-400 font-medium mb-1">Walker activity</div>
          <div className="text-gray-300">
            {metrics.walkers.length} unique walker(s)
          </div>
        </div>
      )}
    </div>
  );
}
