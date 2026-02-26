import type { ThemePreference } from '../hooks/useTheme'

interface Props {
  preference: ThemePreference
  onCycle: () => void
}

const ICONS: Record<ThemePreference, string> = {
  system: '◐',
  light: '☀',
  dark: '☾',
}

const LABELS: Record<ThemePreference, string> = {
  system: 'System theme',
  light: 'Light theme',
  dark: 'Dark theme',
}

export function ThemeToggle({ preference, onCycle }: Props) {
  return (
    <button
      onClick={onCycle}
      className="inline-flex items-center justify-center w-8 h-8 rounded-md
                 text-sm transition-colors duration-200
                 hover:bg-[var(--surface-sunken)]
                 text-[color:var(--text-secondary)]"
      aria-label={LABELS[preference]}
      title={LABELS[preference]}
    >
      {ICONS[preference]}
    </button>
  )
}
