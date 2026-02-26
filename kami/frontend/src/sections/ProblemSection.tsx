import type { ProblemData } from '../hooks/useKabuki'

interface Props {
  data: ProblemData
}

export function ProblemSection({ data }: Props) {
  return (
    <section
      id="problem"
      data-kami="section:problem"
      className="section flex items-center justify-center bg-accent-surface text-fg-on-accent px-8"
      aria-label="The Problem"
    >
      <div className="max-w-4xl grid md:grid-cols-2 gap-12 items-center">
        <div>
          <h2 className="text-4xl font-bold mb-6">{data.title}</h2>
          <p className="text-fg-on-accent/80 text-lg leading-relaxed mb-4">{data.narrative}</p>
          {data.bullet_points.length > 0 && (
            <ul className="text-fg-on-accent/80 space-y-3">
              {data.bullet_points.map((bp, i) => (
                <li key={i} className="flex items-start gap-2">
                  <span className="text-brand mt-1">&#9679;</span>
                  {bp}
                </li>
              ))}
            </ul>
          )}
        </div>
        {data.stat && (
          <div className="flex flex-col items-center gap-4">
            <div className="text-7xl font-black text-brand">{data.stat}</div>
            {data.stat_label && (
              <p className="text-fg-faint text-center">{data.stat_label}</p>
            )}
          </div>
        )}
      </div>
    </section>
  )
}
