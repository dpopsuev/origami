import { useEffect, useState } from 'react'

export interface AgentIntro {
  persona_name: string
  element: string
  role: string
  catchphrase: string
}

export interface Dialog {
  from: string
  to: string
  message: string
}

export interface ThemeData {
  name: string
  agent_intros: AgentIntro[]
  node_descriptions: Record<string, string>
  costume_assets: Record<string, string>
  cooperation_dialogs: Dialog[]
}

export interface PipelineData {
  nodes: Record<string, string>
}

export interface HeroData {
  title: string
  subtitle: string
  presenter?: string
  logo?: string
  framework?: string
}

export interface ProblemData {
  title: string
  narrative: string
  bullet_points: string[]
  stat?: string
  stat_label?: string
}

export interface MetricData {
  label: string
  value: number
  color?: string
}

export interface SummaryCardData {
  value: string
  label: string
  color?: string
}

export interface ResultsData {
  title: string
  description?: string
  metrics: MetricData[]
  summary?: SummaryCardData[]
}

export interface CompetitorData {
  name: string
  fields: Record<string, string>
  highlight?: boolean
}

export interface ArchComponentData {
  name: string
  description: string
  color?: string
}

export interface ArchitectureData {
  title: string
  components: ArchComponentData[]
  footer?: string
}

export interface MilestoneData {
  id: string
  label: string
  status: string
}

export interface ClosingData {
  headline: string
  tagline?: string
  lines?: string[]
}

export interface PresentationData {
  hero?: HeroData
  problem?: ProblemData
  results?: ResultsData
  competitive?: CompetitorData[]
  architecture?: ArchitectureData
  roadmap?: MilestoneData[]
  closing?: ClosingData
  transition_line?: string
}

export type AppMode = 'presentation' | 'debugger'

export interface UsePresentationResult {
  theme: ThemeData | null
  pipeline: PipelineData | null
  presentation: PresentationData | null
  loading: boolean
  mode: AppMode
}

export function usePresentation(): UsePresentationResult {
  const [theme, setTheme] = useState<ThemeData | null>(null)
  const [pipeline, setPipeline] = useState<PipelineData | null>(null)
  const [presentation, setPresentation] = useState<PresentationData | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let cancelled = false
    async function load() {
      try {
        const [themeRes, pipelineRes, presentationRes] = await Promise.all([
          fetch('/api/theme'),
          fetch('/api/pipeline'),
          fetch('/api/presentation'),
        ])
        if (cancelled) return
        const themeJSON = await themeRes.json()
        const pipelineJSON = await pipelineRes.json()
        const presentationJSON = await presentationRes.json()
        if (cancelled) return
        setTheme(themeJSON)
        setPipeline(pipelineJSON)
        setPresentation(presentationJSON)
      } catch {
        // API unavailable — fall back to debugger mode
      } finally {
        if (!cancelled) setLoading(false)
      }
    }
    load()
    return () => { cancelled = true }
  }, [])

  const hasPresentation = presentation !== null &&
    (presentation.hero != null || presentation.problem != null ||
     presentation.results != null || presentation.closing != null)

  return {
    theme,
    pipeline,
    presentation,
    loading,
    mode: hasPresentation ? 'presentation' : 'debugger',
  }
}
