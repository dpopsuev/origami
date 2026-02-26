import type { KamiEvent } from '../hooks/useSSE'

interface Props {
  events: KamiEvent[]
}

const TYPE_COLORS: Record<string, string> = {
  node_enter: 'bg-rh-teal-10 text-el-water',
  node_exit: 'bg-rh-teal-10 text-fg-muted',
  transition: 'bg-rh-purple-10 text-el-earth',
  signal: 'bg-brand-subtle text-brand',
  walk_error: 'bg-rh-red-20 text-rh-red-70',
  breakpoint_hit: 'bg-rh-red-30 text-rh-red-70',
  paused: 'bg-sunken text-fg',
  resumed: 'bg-rh-teal-30 text-fg',
}

export function MonologuePanel({ events }: Props) {
  return (
    <div className="flex flex-col gap-1 p-3 overflow-y-auto max-h-96 font-mono text-sm" data-kami="component:monologue">
      {events.length === 0 && (
        <div className="text-fg-faint italic">Waiting for events...</div>
      )}
      {events.map((e, i) => (
        <div
          key={i}
          className={`flex items-center gap-2 rounded px-2 py-1 ${TYPE_COLORS[e.type] || 'bg-sunken/50 text-fg-muted'}`}
        >
          <span className="shrink-0 text-xs opacity-60">
            {new Date(e.ts).toLocaleTimeString()}
          </span>
          <span className="font-semibold shrink-0">{e.type}</span>
          {e.node && <span className="text-xs">{'\u2192'} {e.node}</span>}
          {e.agent && <span className="text-xs opacity-70">[{e.agent}]</span>}
          {e.error && (
            <span className="text-xs text-danger ml-auto">{e.error}</span>
          )}
        </div>
      ))}
    </div>
  )
}
