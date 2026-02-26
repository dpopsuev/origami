import type { ResultsData } from '../hooks/useKabuki'

interface Props {
  data: ResultsData
}

export function ResultsSection({ data }: Props) {
  return (
    <section
      id="results"
      data-kami="section:results"
      className="section flex items-center justify-center bg-accent-surface text-fg-on-accent px-8"
      aria-label="Results"
    >
      <div className="max-w-4xl w-full">
        <h2 className="text-4xl font-bold mb-8 text-center">{data.title}</h2>
        {data.description && (
          <p className="text-center text-fg-faint mb-10">{data.description}</p>
        )}
        <div className="grid md:grid-cols-2 gap-8">
          {data.metrics.map((m) => (
            <div
              key={m.label}
              data-kami={`metric:${m.label.replace(/\s+/g, '-').toLowerCase()}`}
              className="bg-raised/30 rounded-2xl p-6"
            >
              <div className="text-fg-faint text-sm mb-2">{m.label}</div>
              <div className="text-5xl font-black mb-4">{m.value.toFixed(2)}</div>
              <div className="w-full bg-raised rounded-full h-3">
                <div
                  className={`${m.color || 'bg-success'} h-3 rounded-full transition-all duration-1000`}
                  style={{ width: `${m.value * 100}%` }}
                />
              </div>
            </div>
          ))}
        </div>
        {data.summary && data.summary.length > 0 && (
          <div className="mt-8 grid grid-cols-3 gap-4 text-center">
            {data.summary.map((s) => (
              <div key={s.label} className="bg-raised/30 rounded-xl p-4">
                <div className={`text-2xl font-bold ${s.color || 'text-el-air'}`}>
                  {s.value}
                </div>
                <div className="text-xs text-fg-faint">{s.label}</div>
              </div>
            ))}
          </div>
        )}
      </div>
    </section>
  )
}
