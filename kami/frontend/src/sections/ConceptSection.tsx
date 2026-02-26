interface ConceptCard {
  name: string
  icon?: string
  description: string
  color?: string
}

interface Props {
  id: string
  title: string
  subtitle?: string
  cards: ConceptCard[]
}

export function ConceptSection({ id, title, subtitle, cards }: Props) {
  return (
    <section
      id={id}
      data-kami={`section:${id}`}
      className="section flex items-center justify-center bg-canvas px-8"
      aria-label={title}
    >
      <div className="max-w-5xl w-full text-center">
        <h2 className="text-3xl font-bold text-fg mb-2">{title}</h2>
        {subtitle && (
          <p className="text-fg-muted text-lg mb-8 max-w-2xl mx-auto">{subtitle}</p>
        )}
        <div className={`grid gap-4 ${
          cards.length <= 3 ? 'grid-cols-1 md:grid-cols-3'
            : cards.length <= 4 ? 'grid-cols-2 md:grid-cols-4'
            : 'grid-cols-2 md:grid-cols-3 lg:grid-cols-4'
        }`}>
          {cards.map((card) => (
            <div
              key={card.name}
              className="rounded-xl border border-edge bg-raised p-5 text-left transition-shadow hover:shadow-lg"
            >
              <div className="flex items-center gap-2 mb-3">
                {card.icon && <span className="text-2xl">{card.icon}</span>}
                <h3
                  className="font-bold text-fg"
                  style={card.color ? { color: card.color } : undefined}
                >
                  {card.name}
                </h3>
                {card.color && (
                  <span
                    className="w-3 h-3 rounded-full shrink-0"
                    style={{ backgroundColor: card.color }}
                  />
                )}
              </div>
              <p className="text-sm text-fg-muted leading-relaxed">{card.description}</p>
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}
