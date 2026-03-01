import { useState } from "react";

interface ScheduleEntry {
  id: string;
  circuit: string;
  cron: string;
  enabled: boolean;
  lastRun?: string;
  nextRun?: string;
}

interface RunSchedulerProps {
  schedules: ScheduleEntry[];
  circuits: string[];
  onCreateSchedule: (circuit: string, cron: string) => void;
  onToggleSchedule: (id: string, enabled: boolean) => void;
  onDeleteSchedule: (id: string) => void;
}

export function RunScheduler({
  schedules,
  circuits,
  onCreateSchedule,
  onToggleSchedule,
  onDeleteSchedule,
}: RunSchedulerProps) {
  const [showCreate, setShowCreate] = useState(false);
  const [newCircuit, setNewCircuit] = useState(circuits[0] || "");
  const [newCron, setNewCron] = useState("0 */6 * * *");

  const handleCreate = () => {
    onCreateSchedule(newCircuit, newCron);
    setShowCreate(false);
  };

  return (
    <div className="p-4">
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-sm font-semibold text-gray-300 uppercase tracking-wide">
          Scheduled Runs
        </h2>
        <div className="flex items-center gap-2">
          <span className="text-[10px] bg-purple-900 text-purple-300 px-2 py-0.5 rounded-full">
            Enterprise
          </span>
          <button
            onClick={() => setShowCreate(true)}
            className="text-xs text-blue-400 hover:text-blue-300"
          >
            + New schedule
          </button>
        </div>
      </div>

      {showCreate && (
        <div className="mb-4 p-3 bg-gray-800 rounded-lg border border-gray-700 space-y-2">
          <select
            value={newCircuit}
            onChange={(e) => setNewCircuit(e.target.value)}
            className="w-full bg-gray-900 border border-gray-600 rounded px-2 py-1 text-xs"
          >
            {circuits.map((p) => (
              <option key={p} value={p}>{p}</option>
            ))}
          </select>
          <input
            value={newCron}
            onChange={(e) => setNewCron(e.target.value)}
            placeholder="Cron expression"
            className="w-full bg-gray-900 border border-gray-600 rounded px-2 py-1 text-xs font-mono"
          />
          <div className="flex gap-2">
            <button
              onClick={handleCreate}
              className="text-xs bg-blue-600 hover:bg-blue-500 px-3 py-1 rounded"
            >
              Create
            </button>
            <button
              onClick={() => setShowCreate(false)}
              className="text-xs text-gray-400 hover:text-white px-3 py-1"
            >
              Cancel
            </button>
          </div>
        </div>
      )}

      {schedules.length === 0 ? (
        <div className="text-gray-500 text-xs">No schedules configured</div>
      ) : (
        <div className="space-y-2">
          {schedules.map((s) => (
            <div
              key={s.id}
              className="flex items-center justify-between p-3 bg-gray-800 rounded-lg border border-gray-700"
            >
              <div>
                <div className="text-sm font-mono">{s.circuit}</div>
                <div className="text-xs text-gray-500 font-mono mt-0.5">
                  {s.cron}
                </div>
                {s.nextRun && (
                  <div className="text-[10px] text-gray-600 mt-0.5">
                    Next: {s.nextRun}
                  </div>
                )}
              </div>
              <div className="flex items-center gap-2">
                <button
                  onClick={() => onToggleSchedule(s.id, !s.enabled)}
                  className={`text-xs px-2 py-0.5 rounded ${
                    s.enabled
                      ? "bg-green-900 text-green-300"
                      : "bg-gray-700 text-gray-500"
                  }`}
                >
                  {s.enabled ? "enabled" : "disabled"}
                </button>
                <button
                  onClick={() => onDeleteSchedule(s.id)}
                  className="text-xs text-gray-500 hover:text-red-400"
                >
                  ✕
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
