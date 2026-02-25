import type { ArchitectureData } from '../hooks/usePresentation'

interface Props {
  data: ArchitectureData
}

const COMPONENT_COLORS = [
  'text-rh-red-50',
  'text-rh-purple-50',
  'text-rh-teal-50',
  'text-rh-gray-60',
]

export function ArchitectureSection({ data }: Props) {
  return (
    <section
      id="architecture"
      data-kami="section:architecture"
      className="section flex items-center justify-center bg-rh-gray-80 text-white px-8"
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
              className="bg-rh-gray-60/30 rounded-2xl p-6"
            >
              <div className={`font-bold text-lg mb-2 ${c.color || COMPONENT_COLORS[i % COMPONENT_COLORS.length]}`}>
                {c.name}
              </div>
              <p className="text-rh-gray-20 text-sm">{c.description}</p>
            </div>
          ))}
        </div>
        {data.footer && (
          <p className="mt-8 text-rh-gray-40 text-sm">{data.footer}</p>
        )}
      </div>
    </section>
  )
}
