interface PersonaCardProps {
  name: string;
  element: string;
  persona: string;
  quote?: string;
  traits?: string;
  active?: boolean;
}

const ELEMENT_COLORS: Record<string, string> = {
  fire: "#DC143C",
  water: "#007BA7",
  earth: "#0047AB",
  air: "#FFBF00",
  diamond: "#0F52BA",
  lightning: "#DC143C",
};

export function PersonaCard({
  name,
  element,
  persona,
  quote,
  traits,
  active,
}: PersonaCardProps) {
  const color = ELEMENT_COLORS[element] || "#555";

  return (
    <div
      className={`p-3 rounded-lg border ${
        active ? "border-white/30 bg-gray-700" : "border-gray-700 bg-gray-800"
      } transition-all`}
    >
      <div className="flex items-center gap-2 mb-2">
        <div
          className="w-3 h-3 rounded-full"
          style={{ backgroundColor: color }}
        />
        <span className="text-sm font-medium">{name}</span>
        {active && (
          <span className="text-[10px] bg-green-900 text-green-300 px-1.5 py-0.5 rounded-full">
            active
          </span>
        )}
      </div>
      <div className="text-xs text-gray-400">
        {element} · {persona}
      </div>
      {quote && (
        <div className="text-xs text-gray-500 italic mt-1">"{quote}"</div>
      )}
      {traits && (
        <div className="text-[10px] text-gray-600 mt-1 font-mono">{traits}</div>
      )}
    </div>
  );
}
