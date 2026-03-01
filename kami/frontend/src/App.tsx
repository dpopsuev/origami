import { type ReactNode, useEffect, useMemo, useState } from 'react'
import { useSSE } from './hooks/useSSE'
import { useKamiWS } from './hooks/useKamiWS'
import { useKamiSelector } from './hooks/useKamiSelector'
import { useKabuki } from './hooks/useKabuki'
import { useTheme } from './hooks/useTheme'
import { ThemeToggle } from './components/ThemeToggle'
import { CircuitGraph } from './components/CircuitGraph'
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
import { CodeShowcaseSection } from './sections/CodeShowcaseSection'
import { ConceptSection } from './sections/ConceptSection'
import { initBridge, updateBridge } from './bridge'
import './selector.css'

const SSE_URL = `${window.location.protocol}//${window.location.hostname}:${window.location.port}/events/stream`
const WS_URL = `ws://${window.location.hostname}:${parseInt(window.location.port || '3000') + 1}`

const SECTION_LABELS: Record<string, string> = {
  hero: 'Introduction',
  agenda: 'Agenda',
  problem: 'The Problem',
  solution: 'The Solution',
  agents: 'Meet the Agents',
  transition: 'Transition',
  demo: 'Live Demo',
  results: 'Results',
  competitive: 'Competitive Landscape',
  architecture: 'Architecture',
  roadmap: 'Roadmap',
  closing: 'Closing',
}

const DEFAULT_ORDER = [
  'hero', 'agenda', 'problem', 'solution', 'agents',
  'transition', 'demo', 'results', 'competitive',
  'architecture', 'roadmap', 'closing',
]

function App() {
  const { events, connected } = useSSE({ url: SSE_URL })
  const { commands, connected: wsConnected, send: wsSend } = useKamiWS({ url: WS_URL })
  const { theme, circuit, kabuki, loading, mode } = useKabuki()
  const { preference, cycle } = useTheme()
  const [activeSection, setActiveSection] = useState('hero')
  useKamiSelector(true)

  useEffect(() => {
    initBridge()
  }, [])

  useEffect(() => {
    updateBridge(events, connected)
  }, [events, connected])

  // Section availability: which sections have data to render
  const available = useMemo<Set<string>>(() => {
    if (mode !== 'kabuki' || !kabuki) return new Set()
    const s = new Set<string>()
    if (kabuki.hero) s.add('hero')
    s.add('agenda')
    if (kabuki.problem) s.add('problem')
    if (circuit?.nodes && Object.keys(circuit.nodes).length > 0) s.add('solution')
    if (theme?.agent_intros && theme.agent_intros.length > 0) s.add('agents')
    if (kabuki.transition_line) s.add('transition')
    s.add('demo')
    if (kabuki.results) s.add('results')
    if (kabuki.competitive && kabuki.competitive.length > 0) s.add('competitive')
    if (kabuki.architecture) s.add('architecture')
    if (kabuki.roadmap && kabuki.roadmap.length > 0) s.add('roadmap')
    if (kabuki.closing) s.add('closing')
    for (const cs of kabuki.code_showcases || []) s.add(cs.id)
    for (const cg of kabuki.concepts || []) s.add(cg.id)
    return s
  }, [mode, kabuki, theme, circuit])

  // Dynamic labels from code showcases and concept groups
  const dynamicLabels = useMemo<Record<string, string>>(() => {
    const labels: Record<string, string> = {}
    for (const cs of kabuki?.code_showcases || []) labels[cs.id] = cs.title
    for (const cg of kabuki?.concepts || []) labels[cg.id] = cg.title
    return labels
  }, [kabuki])

  // Ordered list of renderable sections (server-driven or default)
  const sectionList = useMemo(() => {
    const order = kabuki?.section_order ?? DEFAULT_ORDER
    return order
      .filter((id) => available.has(id))
      .map((id) => ({ id, label: SECTION_LABELS[id] || dynamicLabels[id] || id }))
  }, [kabuki, available, dynamicLabels])

  const sectionIds = useMemo(() => sectionList.map((s) => s.id), [sectionList])

  // Section component registry
  const sectionRenderers = useMemo<Record<string, ReactNode>>(() => {
    if (!kabuki) return {}
    const r: Record<string, ReactNode> = {
      hero: kabuki.hero ? <HeroSection data={kabuki.hero} /> : null,
      agenda: <AgendaSection sections={sectionList} activeSection={activeSection} />,
      problem: kabuki.problem ? <ProblemSection data={kabuki.problem} /> : null,
      solution: circuit?.nodes ? <SolutionSection nodes={circuit.nodes} /> : null,
      agents: theme?.agent_intros ? <AgentIntrosSection agents={theme.agent_intros} /> : null,
      transition: kabuki.transition_line ? <TransitionSection line={kabuki.transition_line} /> : null,
      demo: <LiveDemoSection
        events={events}
        commands={commands}
        connected={connected}
        wsConnected={wsConnected}
        nodeDescriptions={theme?.node_descriptions}
        onPause={() => wsSend({ action: 'pause' })}
        onResume={() => wsSend({ action: 'resume' })}
      />,
      results: kabuki.results ? <ResultsSection data={kabuki.results} /> : null,
      competitive: kabuki.competitive ? <CompetitiveSection competitors={kabuki.competitive} /> : null,
      architecture: kabuki.architecture ? <ArchitectureSection data={kabuki.architecture} /> : null,
      roadmap: kabuki.roadmap ? <RoadmapSection milestones={kabuki.roadmap} /> : null,
      closing: kabuki.closing ? <ClosingSection data={kabuki.closing} /> : null,
    }
    for (const cs of kabuki.code_showcases || []) {
      r[cs.id] = <CodeShowcaseSection id={cs.id} title={cs.title} blocks={cs.blocks} />
    }
    for (const cg of kabuki.concepts || []) {
      r[cg.id] = <ConceptSection id={cg.id} title={cg.title} subtitle={cg.subtitle} cards={cg.cards} />
    }
    return r
  }, [kabuki, circuit, theme, events, commands, connected, wsConnected, sectionList, activeSection])

  useEffect(() => {
    if (mode !== 'kabuki' || sectionIds.length === 0) return
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

  useEffect(() => {
    if (mode !== 'kabuki' || sectionIds.length === 0) return
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
      <div className="h-screen flex items-center justify-center bg-accent-surface text-fg-on-accent">
        <p className="text-xl animate-pulse">Loading...</p>
      </div>
    )
  }

  if (mode === 'kabuki' && kabuki) {
    return (
      <main>
        <div className="fixed top-4 right-4 z-50">
          <ThemeToggle preference={preference} onCycle={cycle} />
        </div>
        {sectionIds.map((id) => {
          const node = sectionRenderers[id]
          return node ? <div key={id}>{node}</div> : null
        })}
      </main>
    )
  }

  // Debugger-only mode
  return (
    <div className="h-screen flex flex-col bg-canvas">
      <header className="flex items-center justify-between px-4 py-2 border-b border-edge bg-accent-surface text-fg-on-accent">
        <div className="flex items-center gap-3">
          <h1 className="text-lg font-bold tracking-tight">
            <span className="text-brand">Kami</span> Debugger
          </h1>
        </div>
        <div className="flex items-center gap-3 text-xs">
          <span className={`flex items-center gap-1 ${connected ? 'text-success' : 'text-danger'}`}>
            <span className={`w-2 h-2 rounded-full ${connected ? 'bg-success' : 'bg-danger'}`} />
            SSE {connected ? 'connected' : 'disconnected'}
          </span>
          <span className={`flex items-center gap-1 ${wsConnected ? 'text-success' : 'text-danger'}`}>
            <span className={`w-2 h-2 rounded-full ${wsConnected ? 'bg-success' : 'bg-danger'}`} />
            WS {wsConnected ? 'connected' : 'disconnected'}
          </span>
          <span className="text-fg-faint">
            {events.length} events
          </span>
          <ThemeToggle preference={preference} onCycle={cycle} />
        </div>
      </header>

      <div className="flex flex-1 overflow-hidden">
        <div className="flex-1 relative">
          <CircuitGraph events={events} nodeDescriptions={theme?.node_descriptions} />
          <KamiOverlay commands={commands} />
        </div>
        <div className="w-96 flex flex-col border-l border-edge overflow-hidden">
          <div className="border-b border-edge">
            <h2 className="px-3 py-2 text-sm font-semibold text-fg bg-sunken/30">
              Event Log
            </h2>
            <MonologuePanel events={events} />
          </div>
          <div className="flex-1 overflow-auto">
            <h2 className="px-3 py-2 text-sm font-semibold text-fg bg-sunken/30 border-b border-edge">
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
