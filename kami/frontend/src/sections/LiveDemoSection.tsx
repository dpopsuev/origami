import type { KamiEvent } from '../hooks/useSSE'
import type { WSCommand } from '../hooks/useKamiWS'
import { PipelineGraph } from '../components/PipelineGraph'
import { MonologuePanel } from '../components/MonologuePanel'
import { EvidencePanel } from '../components/EvidencePanel'
import { KamiOverlay } from '../components/KamiOverlay'

interface Props {
  events: KamiEvent[]
  commands: WSCommand[]
  connected: boolean
  wsConnected: boolean
}

export function LiveDemoSection({ events, commands, connected, wsConnected }: Props) {
  return (
    <section
      id="demo"
      data-kami="section:demo"
      className="section flex flex-col bg-white"
      aria-label="Live Demo"
    >
      <div className="flex items-center justify-between px-4 py-2 border-b border-rh-gray-20 bg-rh-gray-80 text-white">
        <div className="flex items-center gap-3">
          <h2 className="text-lg font-bold tracking-tight">
            <span className="text-rh-red-50">Kami</span> Live Demo
          </h2>
        </div>
        <div className="flex items-center gap-3 text-xs">
          <span className={`flex items-center gap-1 ${connected ? 'text-rh-teal-30' : 'text-rh-red-30'}`}>
            <span className={`w-2 h-2 rounded-full ${connected ? 'bg-rh-teal-50' : 'bg-rh-red-50'}`} />
            SSE {connected ? 'connected' : 'disconnected'}
          </span>
          <span className={`flex items-center gap-1 ${wsConnected ? 'text-rh-teal-30' : 'text-rh-red-30'}`}>
            <span className={`w-2 h-2 rounded-full ${wsConnected ? 'bg-rh-teal-50' : 'bg-rh-red-50'}`} />
            WS {wsConnected ? 'connected' : 'disconnected'}
          </span>
          <span className="text-rh-gray-40">{events.length} events</span>
        </div>
      </div>

      <div className="flex flex-1 overflow-hidden">
        <div className="flex-1 relative">
          <PipelineGraph events={events} />
          <KamiOverlay commands={commands} />
        </div>
        <div className="w-96 flex flex-col border-l border-rh-gray-20 overflow-hidden">
          <div className="border-b border-rh-gray-20">
            <h3 className="px-3 py-2 text-sm font-semibold text-rh-gray-80 bg-rh-gray-20/30">
              Event Log
            </h3>
            <MonologuePanel events={events} />
          </div>
          <div className="flex-1 overflow-auto">
            <h3 className="px-3 py-2 text-sm font-semibold text-rh-gray-80 bg-rh-gray-20/30 border-b border-rh-gray-20">
              Evidence
            </h3>
            <EvidencePanel events={events} />
          </div>
        </div>
      </div>
    </section>
  )
}
