import { useState } from "react";

interface NodeTemplate {
  id: string;
  name: string;
  type: string;
  element?: string;
  zone?: string;
  config: Record<string, unknown>;
  scope: "personal" | "team";
  author: string;
}

interface NodeTemplatesProps {
  templates: NodeTemplate[];
  onApply: (template: NodeTemplate) => void;
  onSave: (name: string, config: Record<string, unknown>, scope: "personal" | "team") => void;
  onDelete: (id: string) => void;
}

export function NodeTemplates({
  templates,
  onApply,
  onSave,
  onDelete,
}: NodeTemplatesProps) {
  const [showSave, setShowSave] = useState(false);
  const [newName, setNewName] = useState("");
  const [newScope, setNewScope] = useState<"personal" | "team">("personal");

  const personal = templates.filter((t) => t.scope === "personal");
  const team = templates.filter((t) => t.scope === "team");

  const handleSave = () => {
    onSave(newName, {}, newScope);
    setShowSave(false);
    setNewName("");
  };

  return (
    <div className="p-3">
      <div className="flex items-center justify-between mb-3">
        <h3 className="text-xs font-semibold text-gray-400 uppercase tracking-wide">
          Templates
        </h3>
        <button
          onClick={() => setShowSave(true)}
          className="text-[10px] text-blue-400 hover:text-blue-300"
        >
          + Save current
        </button>
      </div>

      {showSave && (
        <div className="mb-3 p-2 bg-gray-800 rounded border border-gray-700 space-y-2">
          <input
            value={newName}
            onChange={(e) => setNewName(e.target.value)}
            placeholder="Template name"
            className="w-full bg-gray-900 border border-gray-600 rounded px-2 py-1 text-xs"
          />
          <div className="flex gap-2">
            <button
              onClick={() => setNewScope("personal")}
              className={`text-[10px] px-2 py-0.5 rounded ${
                newScope === "personal" ? "bg-blue-600" : "bg-gray-700 text-gray-400"
              }`}
            >
              Personal
            </button>
            <button
              onClick={() => setNewScope("team")}
              className={`text-[10px] px-2 py-0.5 rounded ${
                newScope === "team" ? "bg-blue-600" : "bg-gray-700 text-gray-400"
              }`}
            >
              Team
            </button>
          </div>
          <button onClick={handleSave} className="text-xs bg-blue-600 px-3 py-1 rounded w-full">
            Save
          </button>
        </div>
      )}

      {personal.length > 0 && (
        <TemplateGroup title="Personal" templates={personal} onApply={onApply} onDelete={onDelete} />
      )}
      {team.length > 0 && (
        <TemplateGroup title="Team" templates={team} onApply={onApply} onDelete={onDelete} />
      )}
      {templates.length === 0 && (
        <div className="text-xs text-gray-600 text-center py-4">
          No templates saved yet
        </div>
      )}
    </div>
  );
}

function TemplateGroup({
  title,
  templates,
  onApply,
  onDelete,
}: {
  title: string;
  templates: NodeTemplate[];
  onApply: (t: NodeTemplate) => void;
  onDelete: (id: string) => void;
}) {
  return (
    <div className="mb-3">
      <div className="text-[10px] text-gray-500 mb-1">{title}</div>
      {templates.map((t) => (
        <div
          key={t.id}
          className="flex items-center justify-between p-2 bg-gray-800 rounded mb-1 hover:bg-gray-700 cursor-pointer"
          onClick={() => onApply(t)}
        >
          <div>
            <div className="text-xs">{t.name}</div>
            <div className="text-[10px] text-gray-500">
              {t.type}{t.element ? ` · ${t.element}` : ""}
            </div>
          </div>
          <button
            onClick={(e) => {
              e.stopPropagation();
              onDelete(t.id);
            }}
            className="text-gray-600 hover:text-red-400 text-xs"
          >
            ✕
          </button>
        </div>
      ))}
    </div>
  );
}
