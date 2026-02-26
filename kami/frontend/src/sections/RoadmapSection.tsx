import type { MilestoneData } from '../hooks/useKabuki'

interface Props {
  milestones: MilestoneData[]
}

export function RoadmapSection({ milestones }: Props) {
  return (
    <section
      id="roadmap"
      data-kami="section:roadmap"
      className="section flex items-center justify-center bg-canvas px-8"
      aria-label="Roadmap"
    >
      <div className="max-w-4xl w-full text-center">
        <h2 className="text-4xl font-bold text-fg mb-10">Roadmap</h2>
        <div className="flex items-center justify-center gap-2">
          {milestones.map((m, i) => (
            <div key={m.id} className="flex items-center gap-2">
              <div className="flex flex-col items-center">
                <div
                  data-kami={`milestone:${m.id}`}
                  className={`w-12 h-12 rounded-full flex items-center justify-center text-sm font-bold ${
                    m.status === 'done'
                      ? 'bg-success text-white'
                      : m.status === 'current'
                        ? 'bg-brand text-white animate-pulse'
                        : 'bg-sunken text-fg-muted'
                  }`}
                >
                  {m.id}
                </div>
                <div className="text-xs text-fg-muted mt-2 w-20 text-center">
                  {m.label}
                </div>
              </div>
              {i < milestones.length - 1 && (
                <div className={`w-8 h-0.5 ${
                  m.status === 'done' ? 'bg-success' : 'bg-sunken'
                }`} />
              )}
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}
