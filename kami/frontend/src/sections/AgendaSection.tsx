interface Props {
  sections: { id: string; label: string }[]
  activeSection: string
}

export function AgendaSection({ sections, activeSection }: Props) {
  return (
    <section
      id="agenda"
      data-kami="section:agenda"
      className="section flex items-center justify-center bg-white px-8"
      aria-label="Agenda"
    >
      <div className="max-w-xl w-full">
        <h2 className="text-4xl font-bold text-rh-gray-80 mb-8">Agenda</h2>
        <nav className="flex flex-col gap-3">
          {sections.map((s) => (
            <a
              key={s.id}
              href={`#${s.id}`}
              className={`flex items-center gap-3 text-lg px-4 py-2 rounded-lg transition-all ${
                activeSection === s.id
                  ? 'bg-rh-red-50 text-white font-semibold'
                  : 'text-rh-gray-60 hover:bg-rh-red-10 hover:text-rh-red-60'
              }`}
            >
              <span className="text-rh-red-50">&#9654;</span>
              {s.label}
            </a>
          ))}
        </nav>
      </div>
    </section>
  )
}
