import { useMemo, useState } from 'react'
import type { KamiEvent } from '../hooks/useSSE'
import type { WSCommand } from '../hooks/useKamiWS'
import { PipelineGraph } from '../components/PipelineGraph'
import { KamiOverlay } from '../components/KamiOverlay'

interface Props {
  events: KamiEvent[]
  commands: WSCommand[]
  connected: boolean
  wsConnected: boolean
  nodeDescriptions?: Record<string, string>
  onPause?: () => void
  onResume?: () => void
}

interface AgentInfo {
  name: string
  element: string
  lastNode?: string
  eventCount: number
}

interface CaseInfo {
  id: string
  summary?: string
  confidence?: number
  defectType?: string
}

const EL_BG: Record<string, string> = {
  fire: 'bg-[var(--el-fire)]',
  water: 'bg-[var(--el-water)]',
  earth: 'bg-[var(--el-earth)]',
  air: 'bg-[var(--el-air)]',
  void: 'bg-[var(--el-void)]',
  lightning: 'bg-[var(--brand-accent)]',
  diamond: 'bg-[var(--el-void)]',
}

export function LiveDemoSection({
  events, commands, connected, wsConnected,
  nodeDescriptions, onPause, onResume,
}: Props) {
  const [selectedAgent, setSelectedAgent] = useState<string | null>(null)
  const [selectedCase, setSelectedCase] = useState<string | null>(null)
  const [paused, setPaused] = useState(false)

  // Derive active agents from events
  const agents = useMemo<AgentInfo[]>(() => {
    const map = new Map<string, AgentInfo>()
    for (const e of events) {
      if (!e.agent) continue
      const existing = map.get(e.agent)
      if (existing) {
        existing.eventCount++
        if (e.node) existing.lastNode = e.node
      } else {
        map.set(e.agent, {
          name: e.agent,
          element: (e.data?.element as string || 'fire').toLowerCase(),
          lastNode: e.node,
          eventCount: 1,
        })
      }
    }
    return Array.from(map.values())
  }, [events])

  // TX: last outgoing prompt for selected agent
  const txContent = useMemo(() => {
    if (!selectedAgent) return null
    for (let i = events.length - 1; i >= 0; i--) {
      const e = events[i]
      if (e.agent === selectedAgent && e.type === 'signal' && e.data?.prompt) {
        return String(e.data.prompt)
      }
      if (e.agent === selectedAgent && e.type === 'node_enter' && e.data?.prompt) {
        return String(e.data.prompt)
      }
    }
    return null
  }, [events, selectedAgent])

  // RX: last response for selected agent
  const rxContent = useMemo(() => {
    if (!selectedAgent) return null
    for (let i = events.length - 1; i >= 0; i--) {
      const e = events[i]
      if (e.agent === selectedAgent && e.type === 'node_exit' && e.data) {
        const response = e.data.response || e.data.result || e.data.artifact
        if (response) {
          return typeof response === 'string' ? response : JSON.stringify(response, null, 2)
        }
      }
    }
    return null
  }, [events, selectedAgent])

  // Derive RCA cases from report node exits
  const cases = useMemo<CaseInfo[]>(() => {
    const map = new Map<string, CaseInfo>()
    for (const e of events) {
      if (!e.case_id) continue
      if (!map.has(e.case_id)) {
        map.set(e.case_id, { id: e.case_id })
      }
      if (e.type === 'node_exit' && e.node === 'report' && e.data) {
        const info = map.get(e.case_id)!
        if (e.data.summary) info.summary = String(e.data.summary)
        if (e.data.confidence != null) info.confidence = Number(e.data.confidence)
        if (e.data.defect_type) info.defectType = String(e.data.defect_type)
      }
    }
    return Array.from(map.values())
  }, [events])

  const selectedCaseInfo = cases.find((c) => c.id === selectedCase)

  const handlePauseToggle = () => {
    if (paused) {
      onResume?.()
    } else {
      onPause?.()
    }
    setPaused(!paused)
  }

  const resolvedAgent = selectedAgent || (agents.length > 0 ? agents[0].name : null)

  return (
    <section
      id="demo"
      data-kami="section:demo"
      className="section flex flex-col bg-canvas"
      aria-label="Live Demo / War Room"
    >
      {/* Top bar: Agent tabs + status + controls */}
      <div className="flex items-center justify-between px-3 py-1.5 border-b border-edge bg-accent-surface text-fg-on-accent">
        <div className="flex items-center gap-2">
          <h2 className="text-sm font-bold tracking-tight mr-3">
            <span className="text-brand">War</span> Room
          </h2>
          {agents.map((agent) => {
            const isSelected = resolvedAgent === agent.name
            const elClass = EL_BG[agent.element] || 'bg-[var(--brand-accent)]'
            return (
              <button
                key={agent.name}
                onClick={() => setSelectedAgent(agent.name)}
                className={`px-2 py-1 rounded text-xs font-medium transition-all ${
                  isSelected
                    ? `${elClass} text-white shadow-sm`
                    : 'text-fg-faint hover:text-fg-on-accent'
                }`}
              >
                <span className={`inline-block w-2 h-2 rounded-full mr-1 ${elClass}`} />
                {agent.name}
              </button>
            )
          })}
          {agents.length === 0 && (
            <span className="text-xs text-fg-faint italic">Waiting for agents...</span>
          )}
        </div>
        <div className="flex items-center gap-3 text-xs">
          <button
            onClick={handlePauseToggle}
            className="px-2 py-1 rounded bg-raised text-fg border border-edge hover:bg-sunken transition-colors"
          >
            {paused ? '\u25B6 Resume' : '\u23F8 Pause'}
          </button>
          <span className={`flex items-center gap-1 ${connected ? 'text-success' : 'text-danger'}`}>
            <span className={`w-1.5 h-1.5 rounded-full ${connected ? 'bg-success' : 'bg-danger'}`} />
            SSE
          </span>
          <span className={`flex items-center gap-1 ${wsConnected ? 'text-success' : 'text-danger'}`}>
            <span className={`w-1.5 h-1.5 rounded-full ${wsConnected ? 'bg-success' : 'bg-danger'}`} />
            WS
          </span>
          <span className="text-fg-faint">{events.length}</span>
        </div>
      </div>

      {/* Middle: TX | Graph | RX */}
      <div className="flex flex-1 overflow-hidden min-h-0">
        {/* TX panel */}
        <div className="w-64 flex flex-col border-r border-edge overflow-hidden">
          <div className="px-3 py-1.5 text-xs font-semibold text-fg bg-sunken/30 border-b border-edge flex items-center gap-1">
            <span className="text-el-fire">TX</span>
            <span className="text-fg-faint">Outgoing</span>
          </div>
          <div className="flex-1 overflow-auto p-3">
            {txContent ? (
              <pre className="text-xs text-fg-muted whitespace-pre-wrap font-mono leading-relaxed">{txContent}</pre>
            ) : (
              <p className="text-xs text-fg-faint italic">
                {resolvedAgent ? `No prompt from ${resolvedAgent} yet` : 'Select an agent'}
              </p>
            )}
          </div>
        </div>

        {/* Center: Pipeline graph */}
        <div className="flex-1 relative min-w-0">
          <PipelineGraph events={events} nodeDescriptions={nodeDescriptions} />
          <KamiOverlay commands={commands} />
        </div>

        {/* RX panel */}
        <div className="w-64 flex flex-col border-l border-edge overflow-hidden">
          <div className="px-3 py-1.5 text-xs font-semibold text-fg bg-sunken/30 border-b border-edge flex items-center gap-1">
            <span className="text-el-water">RX</span>
            <span className="text-fg-faint">Incoming</span>
          </div>
          <div className="flex-1 overflow-auto p-3">
            {rxContent ? (
              <pre className="text-xs text-fg-muted whitespace-pre-wrap font-mono leading-relaxed">{rxContent}</pre>
            ) : (
              <p className="text-xs text-fg-faint italic">
                {resolvedAgent ? `No response from ${resolvedAgent} yet` : 'Select an agent'}
              </p>
            )}
          </div>
        </div>
      </div>

      {/* Bottom: RCA case tabs */}
      {cases.length > 0 && (
        <div className="border-t border-edge bg-sunken/20">
          <div className="flex items-center gap-1 px-3 py-1 overflow-x-auto">
            <span className="text-xs font-semibold text-fg-muted mr-2">Cases</span>
            {cases.map((c) => (
              <button
                key={c.id}
                onClick={() => setSelectedCase(c.id === selectedCase ? null : c.id)}
                className={`px-2 py-1 rounded text-xs transition-colors ${
                  c.id === selectedCase
                    ? 'bg-brand text-white'
                    : 'text-fg-muted hover:text-fg hover:bg-raised'
                }`}
              >
                {c.id}
                {c.confidence != null && (
                  <span className="ml-1 opacity-70">{(c.confidence * 100).toFixed(0)}%</span>
                )}
              </button>
            ))}
          </div>
          {selectedCaseInfo && (
            <div className="px-4 py-2 border-t border-edge text-sm">
              <div className="flex items-center gap-4">
                {selectedCaseInfo.defectType && (
                  <span className="text-fg font-medium">{selectedCaseInfo.defectType}</span>
                )}
                {selectedCaseInfo.confidence != null && (
                  <span className="text-fg-muted">
                    Confidence: {(selectedCaseInfo.confidence * 100).toFixed(0)}%
                  </span>
                )}
              </div>
              {selectedCaseInfo.summary && (
                <p className="text-fg-muted mt-1 text-xs">{selectedCaseInfo.summary}</p>
              )}
            </div>
          )}
        </div>
      )}
    </section>
  )
}
