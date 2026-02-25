import { useEffect } from 'react'
import { useSSE } from './hooks/useSSE'
import { useKamiWS } from './hooks/useKamiWS'
import { PipelineGraph } from './components/PipelineGraph'
import { MonologuePanel } from './components/MonologuePanel'
import { EvidencePanel } from './components/EvidencePanel'
import { KamiOverlay } from './components/KamiOverlay'
import { initBridge, updateBridge } from './bridge'

const SSE_URL = `${window.location.protocol}//${window.location.hostname}:${window.location.port}/events/stream`
const WS_URL = `ws://${window.location.hostname}:${parseInt(window.location.port || '3000') + 1}`

function App() {
  const { events, connected } = useSSE({ url: SSE_URL })
  const { commands, connected: wsConnected } = useKamiWS({ url: WS_URL })

  useEffect(() => {
    initBridge()
  }, [])

  useEffect(() => {
    updateBridge(events, connected)
  }, [events, connected])

  return (
    <div className="h-screen flex flex-col bg-white">
      {/* Header */}
      <header className="flex items-center justify-between px-4 py-2 border-b border-rh-gray-20 bg-rh-gray-80 text-white">
        <div className="flex items-center gap-3">
          <h1 className="text-lg font-bold tracking-tight">
            <span className="text-rh-red-50">Kami</span> Debugger
          </h1>
        </div>
        <div className="flex items-center gap-3 text-xs">
          <span
            className={`flex items-center gap-1 ${connected ? 'text-rh-teal-30' : 'text-rh-red-30'}`}
          >
            <span
              className={`w-2 h-2 rounded-full ${connected ? 'bg-rh-teal-50' : 'bg-rh-red-50'}`}
            />
            SSE {connected ? 'connected' : 'disconnected'}
          </span>
          <span
            className={`flex items-center gap-1 ${wsConnected ? 'text-rh-teal-30' : 'text-rh-red-30'}`}
          >
            <span
              className={`w-2 h-2 rounded-full ${wsConnected ? 'bg-rh-teal-50' : 'bg-rh-red-50'}`}
            />
            WS {wsConnected ? 'connected' : 'disconnected'}
          </span>
          <span className="text-rh-gray-40">
            {events.length} events
          </span>
        </div>
      </header>

      {/* Main content */}
      <div className="flex flex-1 overflow-hidden">
        {/* Graph area */}
        <div className="flex-1 relative">
          <PipelineGraph events={events} />
          <KamiOverlay commands={commands} />
        </div>

        {/* Side panel */}
        <div className="w-96 flex flex-col border-l border-rh-gray-20 overflow-hidden">
          <div className="border-b border-rh-gray-20">
            <h2 className="px-3 py-2 text-sm font-semibold text-rh-gray-80 bg-rh-gray-20/30">
              Event Log
            </h2>
            <MonologuePanel events={events} />
          </div>
          <div className="flex-1 overflow-auto">
            <h2 className="px-3 py-2 text-sm font-semibold text-rh-gray-80 bg-rh-gray-20/30 border-b border-rh-gray-20">
              Evidence
            </h2>
            <EvidencePanel events={events} />
          </div>
        </div>
      </div>
    </div>
  )
}

export default App
