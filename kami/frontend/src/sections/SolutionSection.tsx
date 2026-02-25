interface Props {
  nodes: Record<string, string>
}

const ZONE_COLORS = [
  'bg-rh-teal-10 border-rh-teal-50',
  'bg-rh-purple-50/10 border-rh-purple-50',
  'bg-rh-red-10 border-rh-red-50',
  'bg-rh-gray-20/50 border-rh-gray-60',
]

export function SolutionSection({ nodes }: Props) {
  const entries = Object.entries(nodes)
  return (
    <section
      id="solution"
      data-kami="section:solution"
      className="section flex items-center justify-center bg-white px-8"
      aria-label="The Solution"
    >
      <div className="max-w-5xl text-center">
        <h2 className="text-4xl font-bold text-rh-gray-80 mb-4">The Solution</h2>
        <p className="text-rh-gray-60 text-lg mb-10 max-w-2xl mx-auto">
          A {entries.length}-node AI pipeline that processes failures through
          specialized stages, each with its own purpose and expertise.
        </p>
        <div className="flex flex-wrap justify-center gap-4">
          {entries.map(([name, desc], i) => (
            <div key={name} className="flex items-center gap-3">
              <div
                data-kami={`node:${name}`}
                className={`rounded-xl border-2 px-5 py-3 ${ZONE_COLORS[i % ZONE_COLORS.length]}`}
              >
                <div className="font-bold text-rh-gray-80 capitalize">{name}</div>
                <div className="text-xs text-rh-gray-60">{desc}</div>
              </div>
              {i < entries.length - 1 && (
                <span className="text-rh-gray-40 text-xl">{'\u2192'}</span>
              )}
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}
