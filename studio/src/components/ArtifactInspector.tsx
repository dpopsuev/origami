import { useEffect, useState } from "react";

interface StudioEvent {
  id: number;
  run_id: string;
  type: string;
  ts: string;
  node?: string;
  edge?: string;
  walker?: string;
  elapsed_ms?: number;
  error?: string;
  metadata?: Record<string, unknown>;
}

interface ArtifactInspectorProps {
  runId: string;
  selectedNode?: string | null;
}

export function ArtifactInspector({ runId, selectedNode }: ArtifactInspectorProps) {
  const [events, setEvents] = useState<StudioEvent[]>([]);

  useEffect(() => {
    if (!runId) return;

    const fetchEvents = async () => {
      try {
        const resp = await fetch(`/api/runs/${runId}/events`);
        if (resp.ok) {
          const data = await resp.json();
          setEvents(data || []);
        }
      } catch {
        // API not available
      }
    };
    fetchEvents();

    const eventSource = new EventSource(`/api/runs/${runId}/events/stream`);
    eventSource.onmessage = (msg) => {
      try {
        const evt: StudioEvent = JSON.parse(msg.data);
        setEvents((prev) => [...prev, evt]);
      } catch {
        // skip malformed
      }
    };
    return () => eventSource.close();
  }, [runId]);

  const filtered = selectedNode
    ? events.filter((e) => e.node === selectedNode)
    : events;

  const typeColor = (type: string) => {
    if (type.includes("error")) return "text-red-400";
    if (type.includes("enter")) return "text-green-400";
    if (type.includes("exit")) return "text-blue-400";
    if (type.includes("transition")) return "text-yellow-400";
    return "text-gray-400";
  };

  return (
    <div className="p-4 overflow-auto h-full">
      <h2 className="text-sm font-semibold text-gray-300 uppercase tracking-wide mb-3">
        {selectedNode ? `Events: ${selectedNode}` : "All Events"}
      </h2>
      {filtered.length === 0 ? (
        <div className="text-gray-500 text-sm">No events</div>
      ) : (
        <div className="space-y-1">
          {filtered.map((evt) => (
            <div
              key={evt.id}
              className="text-xs font-mono p-2 rounded bg-gray-800 border border-gray-700"
            >
              <span className={typeColor(evt.type)}>{evt.type}</span>
              {evt.node && (
                <span className="text-gray-400 ml-2">node={evt.node}</span>
              )}
              {evt.walker && (
                <span className="text-gray-400 ml-2">walker={evt.walker}</span>
              )}
              {evt.elapsed_ms != null && evt.elapsed_ms > 0 && (
                <span className="text-gray-500 ml-2">
                  {evt.elapsed_ms}ms
                </span>
              )}
              {evt.error && (
                <span className="text-red-400 ml-2">{evt.error}</span>
              )}
              <span className="text-gray-600 ml-2">
                {new Date(evt.ts).toLocaleTimeString()}
              </span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
