interface Dependency {
  name: string;
  version: string;
  type: "adapter" | "marble";
  certified: boolean;
  updateAvailable?: string;
}

interface DependencyViewerProps {
  pipelineName: string;
  dependencies: Dependency[];
  onUpdate?: (name: string, targetVersion: string) => void;
}

export function DependencyViewer({
  pipelineName,
  dependencies,
  onUpdate,
}: DependencyViewerProps) {
  const adapters = dependencies.filter((d) => d.type === "adapter");
  const marbles = dependencies.filter((d) => d.type === "marble");

  return (
    <div className="p-4">
      <h2 className="text-sm font-semibold text-gray-300 uppercase tracking-wide mb-3">
        Dependencies: {pipelineName}
      </h2>

      {dependencies.length === 0 ? (
        <div className="text-xs text-gray-500">
          No external dependencies — all nodes use built-in types.
        </div>
      ) : (
        <div className="space-y-4">
          {adapters.length > 0 && (
            <DependencyGroup
              title="Adapters"
              items={adapters}
              onUpdate={onUpdate}
            />
          )}
          {marbles.length > 0 && (
            <DependencyGroup
              title="Marbles"
              items={marbles}
              onUpdate={onUpdate}
            />
          )}
        </div>
      )}

      <div className="mt-4 text-[10px] text-gray-600">
        {dependencies.length} total · {adapters.length} adapters · {marbles.length} marbles
      </div>
    </div>
  );
}

function DependencyGroup({
  title,
  items,
  onUpdate,
}: {
  title: string;
  items: Dependency[];
  onUpdate?: (name: string, targetVersion: string) => void;
}) {
  return (
    <div>
      <div className="text-xs text-gray-400 font-medium mb-2">{title}</div>
      <div className="space-y-1">
        {items.map((d) => (
          <div
            key={d.name}
            className="flex items-center justify-between p-2 bg-gray-800 rounded border border-gray-700"
          >
            <div className="flex items-center gap-2">
              <span className="text-xs font-mono">{d.name}</span>
              <span className="text-[10px] text-gray-500">v{d.version}</span>
              {d.certified && (
                <span className="text-[8px] bg-green-900 text-green-300 px-1 rounded">
                  certified
                </span>
              )}
            </div>
            {d.updateAvailable && onUpdate && (
              <button
                onClick={() => onUpdate(d.name, d.updateAvailable!)}
                className="text-[10px] text-blue-400 hover:text-blue-300"
              >
                Update to v{d.updateAvailable}
              </button>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}
