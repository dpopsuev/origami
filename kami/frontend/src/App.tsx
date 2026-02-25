import { useEffect, useMemo, useState } from 'react'
import { useSSE } from './hooks/useSSE'
import { useKamiWS } from './hooks/useKamiWS'
import { useKamiSelector } from './hooks/useKamiSelector'
import { usePresentation } from './hooks/usePresentation'
import { PipelineGraph } from './components/PipelineGraph'
import { MonologuePanel } from './components/MonologuePanel'
import { EvidencePanel } from './components/EvidencePanel'
import { KamiOverlay } from './components/KamiOverlay'
import { HeroSection } from './sections/HeroSection'
import { AgendaSection } from './sections/AgendaSection'
import { ProblemSection } from './sections/ProblemSection'
import { SolutionSection } from './sections/SolutionSection'
import { AgentIntrosSection } from './sections/AgentIntrosSection'
import { TransitionSection } from './sections/TransitionSection'
import { LiveDemoSection } from './sections/LiveDemoSection'
import { ResultsSection } from './sections/ResultsSection'
import { CompetitiveSection } from './sections/CompetitiveSection'
import { ArchitectureSection } from './sections/ArchitectureSection'
import { RoadmapSection } from './sections/RoadmapSection'
import { ClosingSection } from './sections/ClosingSection'
import { initBridge, updateBridge } from './bridge'
import './selector.css'

const SSE_URL = `${window.location.protocol}//${window.location.hostname}:${window.location.port}/events/stream`
const WS_URL = `ws://${window.location.hostname}:${parseInt(window.location.port || '3000') + 1}`

function App() {
  const { events, connected } = useSSE({ url: SSE_URL })
  const { commands, connected: wsConnected } = useKamiWS({ url: WS_URL })
  const { theme, pipeline, presentation, loading, mode } = usePresentation()
  const [activeSection, setActiveSection] = useState('hero')
  useKamiSelector(true)

  useEffect(() => {
    initBridge()
  }, [])

  useEffect(() => {
    updateBridge(events, connected)
  }, [events, connected])

  const sectionList = useMemo(() => {
    if (mode !== 'presentation' || !presentation) return []
    const list: { id: string; label: string }[] = []
    if (presentation.hero) list.push({ id: 'hero', label: 'Introduction' })
    list.push({ id: 'agenda', label: 'Agenda' })
    if (presentation.problem) list.push({ id: 'problem', label: 'The Problem' })
    if (pipeline?.nodes && Object.keys(pipeline.nodes).length > 0) {
      list.push({ id: 'solution', label: 'The Solution' })
    }
    if (theme?.agent_intros && theme.agent_intros.length > 0) {
      list.push({ id: 'agents', label: 'Meet the Agents' })
    }
    if (presentation.transition_line) list.push({ id: 'transition', label: 'Transition' })
    list.push({ id: 'demo', label: 'Live Demo' })
    if (presentation.results) list.push({ id: 'results', label: 'Results' })
    if (presentation.competitive && presentation.competitive.length > 0) {
      list.push({ id: 'competitive', label: 'Competitive Landscape' })
    }
    if (presentation.architecture) list.push({ id: 'architecture', label: 'Architecture' })
    if (presentation.roadmap && presentation.roadmap.length > 0) {
      list.push({ id: 'roadmap', label: 'Roadmap' })
    }
    if (presentation.closing) list.push({ id: 'closing', label: 'Closing' })
    return list
  }, [mode, presentation, theme, pipeline])

  const sectionIds = useMemo(() => sectionList.map((s) => s.id), [sectionList])

  // IntersectionObserver for active section tracking
  useEffect(() => {
    if (mode !== 'presentation' || sectionIds.length === 0) return
    const observer = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (entry.isIntersecting) {
            setActiveSection(entry.target.id)
          }
        }
      },
      { threshold: 0.5 }
    )
    for (const id of sectionIds) {
      const el = document.getElementById(id)
      if (el) observer.observe(el)
    }
    return () => observer.disconnect()
  }, [mode, sectionIds])

  // Keyboard navigation
  useEffect(() => {
    if (mode !== 'presentation' || sectionIds.length === 0) return
    const handleKey = (e: KeyboardEvent) => {
      if (e.ctrlKey || e.metaKey) return
      const idx = sectionIds.indexOf(activeSection)
      if (e.key === 'ArrowDown' || e.key === 'PageDown') {
        e.preventDefault()
        const next = Math.min(idx + 1, sectionIds.length - 1)
        document.getElementById(sectionIds[next])?.scrollIntoView({ behavior: 'smooth' })
      } else if (e.key === 'ArrowUp' || e.key === 'PageUp') {
        e.preventDefault()
        const prev = Math.max(idx - 1, 0)
        document.getElementById(sectionIds[prev])?.scrollIntoView({ behavior: 'smooth' })
      }
    }
    window.addEventListener('keydown', handleKey)
    return () => window.removeEventListener('keydown', handleKey)
  }, [mode, activeSection, sectionIds])

  if (loading) {
    return (
      <div className="h-screen flex items-center justify-center bg-rh-gray-80 text-white">
        <p className="text-xl animate-pulse">Loading...</p>
      </div>
    )
  }

  // Presentation mode: scroll-snap SPA
  if (mode === 'presentation' && presentation) {
    return (
      <main>
        {presentation.hero && <HeroSection data={presentation.hero} />}
        {sectionList.length > 0 && (
          <AgendaSection sections={sectionList} activeSection={activeSection} />
        )}
        {presentation.problem && <ProblemSection data={presentation.problem} />}
        {pipeline?.nodes && Object.keys(pipeline.nodes).length > 0 && (
          <SolutionSection nodes={pipeline.nodes} />
        )}
        {theme?.agent_intros && theme.agent_intros.length > 0 && (
          <AgentIntrosSection agents={theme.agent_intros} />
        )}
        {presentation.transition_line && (
          <TransitionSection line={presentation.transition_line} />
        )}
        <LiveDemoSection
          events={events}
          commands={commands}
          connected={connected}
          wsConnected={wsConnected}
        />
        {presentation.results && <ResultsSection data={presentation.results} />}
        {presentation.competitive && presentation.competitive.length > 0 && (
          <CompetitiveSection competitors={presentation.competitive} />
        )}
        {presentation.architecture && (
          <ArchitectureSection data={presentation.architecture} />
        )}
        {presentation.roadmap && presentation.roadmap.length > 0 && (
          <RoadmapSection milestones={presentation.roadmap} />
        )}
        {presentation.closing && <ClosingSection data={presentation.closing} />}
      </main>
    )
  }

  // Debugger-only mode (backward compatible)
  return (
    <div className="h-screen flex flex-col bg-white">
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

      <div className="flex flex-1 overflow-hidden">
        <div className="flex-1 relative">
          <PipelineGraph events={events} />
          <KamiOverlay commands={commands} />
        </div>
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
