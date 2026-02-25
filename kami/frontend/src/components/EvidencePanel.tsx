import type { KamiEvent } from '../hooks/useSSE'

interface Props {
  events: KamiEvent[]
}

export function EvidencePanel({ events }: Props) {
  const artifacts = events.filter(
    (e) => e.type === 'node_exit' && e.data
  )

  if (artifacts.length === 0) {
    return (
      <div className="p-3 text-rh-gray-40 italic text-sm">
        No artifacts yet.
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-2 p-3 overflow-y-auto max-h-64">
      {artifacts.map((e, i) => (
        <div
          key={i}
          className="rounded border border-rh-gray-20 bg-white p-3"
        >
          <div className="flex items-center gap-2 mb-1">
            <span className="font-semibold text-rh-purple-50">{e.node}</span>
            {e.agent && (
              <span className="text-xs text-rh-gray-40">[{e.agent}]</span>
            )}
          </div>
          {e.data && (
            <pre className="text-xs text-rh-gray-60 whitespace-pre-wrap">
              {JSON.stringify(e.data, null, 2)}
            </pre>
          )}
        </div>
      ))}
    </div>
  )
}
