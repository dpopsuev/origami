import type { AgentIntro } from '../hooks/usePresentation'

interface Props {
  agents: AgentIntro[]
}

const ELEMENT_STYLES: Record<string, string> = {
  Fire: 'border-rh-red-50 bg-rh-red-10',
  Water: 'border-rh-teal-50 bg-rh-teal-10',
  Earth: 'border-rh-purple-50 bg-rh-purple-50/10',
  Air: 'border-rh-teal-30 bg-rh-teal-10',
  Diamond: 'border-rh-gray-60 bg-rh-gray-20/50',
  Lightning: 'border-rh-red-40 bg-rh-red-10',
}

const ELEMENT_BADGES: Record<string, string> = {
  Fire: '\uD83D\uDD25',
  Water: '\uD83C\uDF0A',
  Earth: '\uD83E\uDEA8',
  Air: '\uD83D\uDCA8',
  Diamond: '\uD83D\uDC8E',
  Lightning: '\u26A1',
}

export function AgentIntrosSection({ agents }: Props) {
  return (
    <section
      id="agents"
      data-kami="section:agents"
      className="section flex items-center justify-center bg-rh-gray-80 text-white px-8"
      aria-label="Meet the Agents"
    >
      <div className="max-w-5xl">
        <h2 className="text-4xl font-bold mb-8 text-center">Meet the Agents</h2>
        <p className="text-center text-rh-gray-40 mb-10">
          Each AI agent has a distinct personality, element affinity, and investigative style.
        </p>
        <div className="grid md:grid-cols-3 gap-5">
          {agents.map((a) => (
            <div
              key={a.persona_name}
              data-kami={`agent:${a.persona_name}`}
              className={`rounded-2xl border-2 p-5 text-rh-gray-80 ${ELEMENT_STYLES[a.element] || 'border-rh-gray-20 bg-white'}`}
            >
              <div className="flex items-center gap-2 mb-2">
                <span className="text-2xl">{ELEMENT_BADGES[a.element] || ''}</span>
                <div>
                  <div className="font-bold text-lg">{a.persona_name}</div>
                  <div className="text-xs text-rh-gray-60">
                    {a.role} &middot; {a.element}
                  </div>
                </div>
              </div>
              <p className="text-sm italic text-rh-gray-60">
                &ldquo;{a.catchphrase}&rdquo;
              </p>
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}
