import { useState } from "react";

type NotifyChannel = "webhook" | "slack" | "pagerduty" | "email";
type NotifyTrigger = "run_completed" | "run_failed" | "latency_exceeded" | "cost_exceeded";

interface NotificationRule {
  id: string;
  channel: NotifyChannel;
  trigger: NotifyTrigger;
  target: string;
  enabled: boolean;
}

interface NotificationSettingsProps {
  rules: NotificationRule[];
  onCreateRule: (rule: Omit<NotificationRule, "id">) => void;
  onToggleRule: (id: string, enabled: boolean) => void;
  onDeleteRule: (id: string) => void;
}

const CHANNELS: { id: NotifyChannel; label: string }[] = [
  { id: "webhook", label: "Webhook" },
  { id: "slack", label: "Slack" },
  { id: "pagerduty", label: "PagerDuty" },
  { id: "email", label: "Email" },
];

const TRIGGERS: { id: NotifyTrigger; label: string }[] = [
  { id: "run_completed", label: "Run completed" },
  { id: "run_failed", label: "Run failed" },
  { id: "latency_exceeded", label: "Latency threshold exceeded" },
  { id: "cost_exceeded", label: "Cost budget exceeded" },
];

export function NotificationSettings({
  rules,
  onCreateRule,
  onToggleRule,
  onDeleteRule,
}: NotificationSettingsProps) {
  const [showCreate, setShowCreate] = useState(false);
  const [channel, setChannel] = useState<NotifyChannel>("webhook");
  const [trigger, setTrigger] = useState<NotifyTrigger>("run_failed");
  const [target, setTarget] = useState("");

  const handleCreate = () => {
    onCreateRule({ channel, trigger, target, enabled: true });
    setShowCreate(false);
    setTarget("");
  };

  return (
    <div className="p-4">
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-sm font-semibold text-gray-300 uppercase tracking-wide">
          Notifications
        </h2>
        <div className="flex items-center gap-2">
          <span className="text-[10px] bg-purple-900 text-purple-300 px-2 py-0.5 rounded-full">
            Enterprise
          </span>
          <button
            onClick={() => setShowCreate(true)}
            className="text-xs text-blue-400 hover:text-blue-300"
          >
            + Add rule
          </button>
        </div>
      </div>

      {showCreate && (
        <div className="mb-4 p-3 bg-gray-800 rounded-lg border border-gray-700 space-y-2">
          <div className="grid grid-cols-2 gap-2">
            <select
              value={channel}
              onChange={(e) => setChannel(e.target.value as NotifyChannel)}
              className="bg-gray-900 border border-gray-600 rounded px-2 py-1 text-xs"
            >
              {CHANNELS.map((c) => (
                <option key={c.id} value={c.id}>{c.label}</option>
              ))}
            </select>
            <select
              value={trigger}
              onChange={(e) => setTrigger(e.target.value as NotifyTrigger)}
              className="bg-gray-900 border border-gray-600 rounded px-2 py-1 text-xs"
            >
              {TRIGGERS.map((t) => (
                <option key={t.id} value={t.id}>{t.label}</option>
              ))}
            </select>
          </div>
          <input
            value={target}
            onChange={(e) => setTarget(e.target.value)}
            placeholder={channel === "webhook" ? "https://..." : channel === "email" ? "user@example.com" : "channel/service key"}
            className="w-full bg-gray-900 border border-gray-600 rounded px-2 py-1 text-xs"
          />
          <div className="flex gap-2">
            <button onClick={handleCreate} className="text-xs bg-blue-600 hover:bg-blue-500 px-3 py-1 rounded">
              Create
            </button>
            <button onClick={() => setShowCreate(false)} className="text-xs text-gray-400 hover:text-white px-3 py-1">
              Cancel
            </button>
          </div>
        </div>
      )}

      {rules.length === 0 ? (
        <div className="text-gray-500 text-xs">No notification rules configured</div>
      ) : (
        <div className="space-y-2">
          {rules.map((r) => (
            <div
              key={r.id}
              className="flex items-center justify-between p-3 bg-gray-800 rounded-lg border border-gray-700"
            >
              <div>
                <div className="text-xs">
                  <span className="text-gray-300">{TRIGGERS.find((t) => t.id === r.trigger)?.label}</span>
                  <span className="text-gray-600 mx-1">→</span>
                  <span className="text-gray-400">{CHANNELS.find((c) => c.id === r.channel)?.label}</span>
                </div>
                <div className="text-[10px] text-gray-600 font-mono mt-0.5 truncate max-w-[200px]">
                  {r.target}
                </div>
              </div>
              <div className="flex items-center gap-2">
                <button
                  onClick={() => onToggleRule(r.id, !r.enabled)}
                  className={`text-xs px-2 py-0.5 rounded ${
                    r.enabled ? "bg-green-900 text-green-300" : "bg-gray-700 text-gray-500"
                  }`}
                >
                  {r.enabled ? "on" : "off"}
                </button>
                <button
                  onClick={() => onDeleteRule(r.id)}
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
