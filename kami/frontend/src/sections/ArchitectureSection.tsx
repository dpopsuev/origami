import type { ArchitectureData } from '../hooks/useKabuki'

interface Props {
  data: ArchitectureData
}

const COMPONENT_COLORS = [
  'text-brand',
  'text-el-earth',
  'text-el-water',
  'text-fg-muted',
]

export function ArchitectureSection({ data }: Props) {
  return (
    <section
      id="architecture"
      data-kami="section:architecture"
      className="section flex items-center justify-center bg-accent-surface text-fg-on-accent px-8"
      aria-label="Architecture"
    >
      <div className="max-w-4xl text-center">
        <h2 className="text-4xl font-bold mb-8">{data.title}</h2>
        <div className={`grid gap-6 text-left ${
          data.components.length <= 3
            ? 'md:grid-cols-3'
            : data.components.length <= 4
              ? 'md:grid-cols-2 lg:grid-cols-4'
              : 'md:grid-cols-3'
        }`}>
          {data.components.map((c, i) => (
            <div
              key={c.name}
              data-kami={`component:${c.name.toLowerCase().replace(/\s+/g, '-')}`}
              className="bg-raised/30 rounded-2xl p-6"
            >
              <div className={`font-bold text-lg mb-2 ${c.color || COMPONENT_COLORS[i % COMPONENT_COLORS.length]}`}>
                {c.name}
              </div>
              <p className="text-fg-on-accent/70 text-sm">{c.description}</p>
            </div>
          ))}
        </div>
        {data.footer && (
          <p className="mt-8 text-fg-faint text-sm">{data.footer}</p>
        )}
      </div>
    </section>
  )
}
