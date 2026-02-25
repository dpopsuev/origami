import type { KamiEvent } from '../hooks/useSSE'

interface Props {
  events: KamiEvent[]
}

const TYPE_COLORS: Record<string, string> = {
  node_enter: 'bg-rh-teal-10 text-rh-teal-50',
  node_exit: 'bg-rh-teal-10 text-rh-gray-60',
  transition: 'bg-rh-purple-50/10 text-rh-purple-50',
  signal: 'bg-rh-red-10 text-rh-red-60',
  walk_error: 'bg-rh-red-20 text-rh-red-70',
  breakpoint_hit: 'bg-rh-red-30 text-rh-red-70',
  paused: 'bg-rh-gray-20 text-rh-gray-80',
  resumed: 'bg-rh-teal-30 text-rh-gray-80',
}

export function MonologuePanel({ events }: Props) {
  return (
    <div className="flex flex-col gap-1 p-3 overflow-y-auto max-h-96 font-mono text-sm" data-kami="component:monologue">
      {events.length === 0 && (
        <div className="text-rh-gray-40 italic">Waiting for events...</div>
      )}
      {events.map((e, i) => (
        <div
          key={i}
          className={`flex items-center gap-2 rounded px-2 py-1 ${TYPE_COLORS[e.type] || 'bg-rh-gray-20/50 text-rh-gray-60'}`}
        >
          <span className="shrink-0 text-xs opacity-60">
            {new Date(e.ts).toLocaleTimeString()}
          </span>
          <span className="font-semibold shrink-0">{e.type}</span>
          {e.node && <span className="text-xs">→ {e.node}</span>}
          {e.agent && <span className="text-xs opacity-70">[{e.agent}]</span>}
          {e.error && (
            <span className="text-xs text-rh-red-50 ml-auto">{e.error}</span>
          )}
        </div>
      ))}
    </div>
  )
}
