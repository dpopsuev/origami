import type { AgentIntro } from '../hooks/useKabuki'

interface Props {
  agents: AgentIntro[]
}

const ELEMENT_STYLES: Record<string, string> = {
  Fire: 'border-el-fire bg-brand-subtle',
  Water: 'border-el-water bg-rh-teal-10',
  Earth: 'border-el-earth bg-rh-purple-10',
  Air: 'border-el-air bg-rh-teal-10',
  Diamond: 'border-edge bg-sunken/50',
  Lightning: 'border-rh-red-40 bg-brand-subtle',
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
      className="section flex items-center justify-center bg-accent-surface text-fg-on-accent px-8"
      aria-label="Meet the Agents"
    >
      <div className="max-w-5xl">
        <h2 className="text-4xl font-bold mb-8 text-center">Meet the Agents</h2>
        <p className="text-center text-fg-faint mb-10">
          Each AI agent has a distinct personality, element affinity, and investigative style.
        </p>
        <div className="grid md:grid-cols-3 gap-5">
          {agents.map((a) => (
            <div
              key={a.persona_name}
              data-kami={`agent:${a.persona_name}`}
              className={`rounded-2xl border-2 p-5 text-fg ${ELEMENT_STYLES[a.element] || 'border-edge bg-canvas'}`}
            >
              <div className="flex items-center gap-2 mb-2">
                <span className="text-2xl">{ELEMENT_BADGES[a.element] || ''}</span>
                <div>
                  <div className="font-bold text-lg">{a.persona_name}</div>
                  <div className="text-xs text-fg-muted">
                    {a.role} &middot; {a.element}
                  </div>
                </div>
              </div>
              <p className="text-sm italic text-fg-muted">
                &ldquo;{a.catchphrase}&rdquo;
              </p>
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}
