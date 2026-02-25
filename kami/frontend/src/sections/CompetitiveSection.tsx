import type { CompetitorData } from '../hooks/useKabuki'

interface Props {
  competitors: CompetitorData[]
}

export function CompetitiveSection({ competitors }: Props) {
  if (competitors.length === 0) return null

  const fieldKeys = Object.keys(competitors[0].fields)

  return (
    <section
      id="competitive"
      data-kami="section:competitive"
      className="section flex items-center justify-center bg-white px-8"
      aria-label="Competitive Landscape"
    >
      <div className="max-w-5xl w-full">
        <h2 className="text-4xl font-bold text-rh-gray-80 mb-8 text-center">
          Competitive Landscape
        </h2>
        <div className="overflow-x-auto">
          <table className="w-full text-left">
            <thead>
              <tr className="border-b-2 border-rh-gray-20">
                <th className="py-3 px-4 text-rh-gray-60 font-semibold">Framework</th>
                {fieldKeys.map((key) => (
                  <th key={key} className="py-3 px-4 text-rh-gray-60 font-semibold capitalize">
                    {key.replace(/_/g, ' ')}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {competitors.map((c) => (
                <tr
                  key={c.name}
                  data-kami={`framework:${c.name}`}
                  className={`border-b border-rh-gray-20 ${c.highlight ? 'bg-rh-red-10 font-semibold' : ''}`}
                >
                  <td className="py-3 px-4">
                    {c.highlight ? <span className="text-rh-red-50">{c.name}</span> : c.name}
                  </td>
                  {fieldKeys.map((key) => (
                    <td key={key} className="py-3 px-4">{c.fields[key]}</td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </section>
  )
}
