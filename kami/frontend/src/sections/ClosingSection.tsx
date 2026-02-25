import type { ClosingData } from '../hooks/useKabuki'

interface Props {
  data: ClosingData
}

export function ClosingSection({ data }: Props) {
  return (
    <section
      id="closing"
      data-kami="section:closing"
      className="section flex items-center justify-center bg-rh-gray-80 text-white px-8"
      aria-label="Closing"
    >
      <div className="text-center max-w-2xl">
        <div className="text-5xl font-black mb-6">
          <span className="text-rh-red-50">{data.headline}</span>
        </div>
        {data.tagline && (
          <p className="text-rh-gray-20 text-lg mb-8">{data.tagline}</p>
        )}
        {data.lines && data.lines.length > 0 && (
          <div className="flex flex-col gap-2 text-rh-gray-40 text-sm">
            {data.lines.map((line, i) => (
              <p key={i}>{line}</p>
            ))}
          </div>
        )}
      </div>
    </section>
  )
}
