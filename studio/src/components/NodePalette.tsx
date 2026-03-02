import { useCallback } from "react";

const ELEMENT_ITEMS = [
  { element: "fire", label: "Fire", color: "#DC143C", desc: "Bold, fast" },
  { element: "water", label: "Water", color: "#007BA7", desc: "Methodical" },
  { element: "earth", label: "Earth", color: "#0047AB", desc: "Pragmatic" },
  { element: "air", label: "Air", color: "#FFBF00", desc: "Creative" },
  { element: "diamond", label: "Diamond", color: "#0F52BA", desc: "Skeptical" },
  { element: "lightning", label: "Lightning", color: "#DC143C", desc: "Dispatcher" },
];

const FAMILY_ITEMS = [
  { family: "ingest", label: "Ingest" },
  { family: "classify", label: "Classify" },
  { family: "analyze", label: "Analyze" },
  { family: "review", label: "Review" },
  { family: "report", label: "Report" },
  { family: "custom", label: "Custom" },
];

interface NodePaletteProps {
  onDragStart: (data: { element?: string; family?: string }) => void;
}

export function NodePalette({ onDragStart }: NodePaletteProps) {
  const handleDragStart = useCallback(
    (e: React.DragEvent, data: { element?: string; family?: string }) => {
      e.dataTransfer.setData("application/origami-node", JSON.stringify(data));
      e.dataTransfer.effectAllowed = "move";
      onDragStart(data);
    },
    [onDragStart]
  );

  return (
    <div className="p-3 space-y-4">
      <div>
        <h3 className="text-xs font-semibold text-gray-400 uppercase tracking-wide mb-2">
          Elements
        </h3>
        <div className="grid grid-cols-2 gap-1">
          {ELEMENT_ITEMS.map((item) => (
            <div
              key={item.element}
              draggable
              onDragStart={(e) => handleDragStart(e, { element: item.element })}
              className="flex items-center gap-2 p-2 rounded cursor-grab hover:bg-gray-700 transition-colors"
            >
              <div
                className="w-3 h-3 rounded-full shrink-0"
                style={{ backgroundColor: item.color }}
              />
              <div>
                <div className="text-xs font-medium">{item.label}</div>
                <div className="text-[10px] text-gray-500">{item.desc}</div>
              </div>
            </div>
          ))}
        </div>
      </div>

      <div>
        <h3 className="text-xs font-semibold text-gray-400 uppercase tracking-wide mb-2">
          Families
        </h3>
        <div className="space-y-1">
          {FAMILY_ITEMS.map((item) => (
            <div
              key={item.family}
              draggable
              onDragStart={(e) => handleDragStart(e, { family: item.family })}
              className="p-2 rounded cursor-grab hover:bg-gray-700 transition-colors text-xs"
            >
              {item.label}
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
