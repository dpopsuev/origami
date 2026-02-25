import type { MilestoneData } from '../hooks/usePresentation'

interface Props {
  milestones: MilestoneData[]
}

export function RoadmapSection({ milestones }: Props) {
  return (
    <section
      id="roadmap"
      data-kami="section:roadmap"
      className="section flex items-center justify-center bg-white px-8"
      aria-label="Roadmap"
    >
      <div className="max-w-4xl w-full text-center">
        <h2 className="text-4xl font-bold text-rh-gray-80 mb-10">Roadmap</h2>
        <div className="flex items-center justify-center gap-2">
          {milestones.map((m, i) => (
            <div key={m.id} className="flex items-center gap-2">
              <div className="flex flex-col items-center">
                <div
                  data-kami={`milestone:${m.id}`}
                  className={`w-12 h-12 rounded-full flex items-center justify-center text-sm font-bold ${
                    m.status === 'done'
                      ? 'bg-rh-teal-50 text-white'
                      : m.status === 'current'
                        ? 'bg-rh-red-50 text-white animate-pulse'
                        : 'bg-rh-gray-20 text-rh-gray-60'
                  }`}
                >
                  {m.id}
                </div>
                <div className="text-xs text-rh-gray-60 mt-2 w-20 text-center">
                  {m.label}
                </div>
              </div>
              {i < milestones.length - 1 && (
                <div className={`w-8 h-0.5 ${
                  m.status === 'done' ? 'bg-rh-teal-50' : 'bg-rh-gray-20'
                }`} />
              )}
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}
