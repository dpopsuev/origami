import { useState } from "react";

interface AdapterComponent {
  fqcn: string;
  name: string;
  type: "transformer" | "extractor" | "hook" | "node" | "marble";
  adapter: string;
  certified: boolean;
  version: string;
  description?: string;
}

interface ComponentPaletteProps {
  components: AdapterComponent[];
  onDragStart: (fqcn: string) => void;
  onInstall?: (adapter: string) => void;
}

export function ComponentPalette({
  components,
  onDragStart,
  onInstall,
}: ComponentPaletteProps) {
  const [search, setSearch] = useState("");
  const [typeFilter, setTypeFilter] = useState<string>("all");

  const filtered = components.filter((c) => {
    const matchesSearch =
      c.fqcn.toLowerCase().includes(search.toLowerCase()) ||
      c.name.toLowerCase().includes(search.toLowerCase());
    const matchesType = typeFilter === "all" || c.type === typeFilter;
    return matchesSearch && matchesType;
  });

  const types = ["all", ...new Set(components.map((c) => c.type))];

  const groupedByAdapter = new Map<string, AdapterComponent[]>();
  for (const c of filtered) {
    const list = groupedByAdapter.get(c.adapter) || [];
    list.push(c);
    groupedByAdapter.set(c.adapter, list);
  }

  return (
    <div className="p-3 flex flex-col h-full">
      <div className="mb-3">
        <input
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder="Search components..."
          className="w-full bg-gray-800 border border-gray-600 rounded px-3 py-1.5 text-xs focus:border-blue-500 focus:outline-none"
        />
      </div>

      <div className="flex gap-1 mb-3 flex-wrap">
        {types.map((t) => (
          <button
            key={t}
            onClick={() => setTypeFilter(t)}
            className={`text-[10px] px-2 py-0.5 rounded ${
              typeFilter === t
                ? "bg-blue-600 text-white"
                : "bg-gray-800 text-gray-400 hover:bg-gray-700"
            }`}
          >
            {t}
          </button>
        ))}
      </div>

      <div className="flex-1 overflow-y-auto space-y-3">
        {Array.from(groupedByAdapter.entries()).map(([adapter, items]) => (
          <div key={adapter}>
            <div className="flex items-center justify-between mb-1">
              <span className="text-xs text-gray-400 font-medium">{adapter}</span>
              {onInstall && (
                <button
                  onClick={() => onInstall(adapter)}
                  className="text-[10px] text-blue-400 hover:text-blue-300"
                >
                  Install
                </button>
              )}
            </div>
            {items.map((c) => (
              <div
                key={c.fqcn}
                draggable
                onDragStart={() => onDragStart(c.fqcn)}
                className="flex items-center gap-2 p-2 bg-gray-800 rounded mb-1 cursor-grab hover:bg-gray-700 active:cursor-grabbing"
              >
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-1">
                    <span className="text-xs truncate">{c.name}</span>
                    {c.certified && (
                      <span className="text-[8px] bg-green-900 text-green-300 px-1 rounded">
                        certified
                      </span>
                    )}
                  </div>
                  <div className="text-[10px] text-gray-500 font-mono truncate">
                    {c.fqcn}
                  </div>
                </div>
                <span className="text-[10px] text-gray-600 shrink-0">{c.type}</span>
              </div>
            ))}
          </div>
        ))}

        {filtered.length === 0 && (
          <div className="text-xs text-gray-500 text-center py-4">
            No components match your search
          </div>
        )}
      </div>
    </div>
  );
}
