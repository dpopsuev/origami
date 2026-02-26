interface CodeBlock {
  language: string
  code: string
  annotation?: string
}

interface Props {
  id: string
  title: string
  blocks: CodeBlock[]
}

const LANG_LABELS: Record<string, string> = {
  yaml: 'YAML',
  go: 'Go',
  json: 'JSON',
  typescript: 'TypeScript',
  javascript: 'JavaScript',
  bash: 'Bash',
  shell: 'Shell',
}

export function CodeShowcaseSection({ id, title, blocks }: Props) {
  return (
    <section
      id={id}
      data-kami={`section:${id}`}
      className="section flex items-center justify-center bg-canvas px-8"
      aria-label={title}
    >
      <div className="max-w-5xl w-full">
        <h2 className="text-3xl font-bold text-fg mb-8 text-center">{title}</h2>
        <div className="space-y-6">
          {blocks.map((block, i) => (
            <div key={i} className="rounded-xl border border-edge overflow-hidden bg-accent-surface">
              <div className="flex items-center justify-between px-4 py-2 border-b border-edge bg-sunken/30">
                <span className="text-xs font-mono font-semibold text-fg-muted">
                  {LANG_LABELS[block.language] || block.language}
                </span>
                {block.annotation && (
                  <span className="text-xs text-fg-faint">{block.annotation}</span>
                )}
              </div>
              <pre className="p-4 overflow-x-auto text-sm font-mono leading-relaxed text-fg-on-accent">
                <code>{block.code}</code>
              </pre>
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}
