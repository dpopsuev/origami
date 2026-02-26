import { useEffect, useState } from 'react'
import type { WSCommand } from '../hooks/useKamiWS'

interface Props {
  commands: WSCommand[]
}

interface Highlight {
  nodes: string[]
  color: string
}

interface Marker {
  node: string
  label: string
  color: string
}

export function KamiOverlay({ commands }: Props) {
  const [highlights, setHighlights] = useState<Highlight[]>([])
  const [markers, setMarkers] = useState<Marker[]>([])
  const [speed, setSpeed] = useState(1.0)

  const styles = getComputedStyle(document.documentElement)
  const defaultHighlight = styles.getPropertyValue('--brand-accent').trim() || '#ee0000'
  const defaultMarker = styles.getPropertyValue('--el-earth').trim() || '#5e40be'

  useEffect(() => {
    const last = commands[commands.length - 1]
    if (!last) return

    switch (last.action) {
      case 'highlight_nodes':
        setHighlights((prev) => [
          ...prev,
          {
            nodes: last.nodes as string[],
            color: (last.color as string) || defaultHighlight,
          },
        ])
        break
      case 'highlight_zone':
        break
      case 'place_marker':
        setMarkers((prev) => [
          ...prev,
          {
            node: last.node as string,
            label: last.label as string,
            color: (last.color as string) || defaultMarker,
          },
        ])
        break
      case 'clear_all':
        setHighlights([])
        setMarkers([])
        break
      case 'set_speed':
        setSpeed(last.speed as number)
        break
    }
  }, [commands, defaultHighlight, defaultMarker])

  if (highlights.length === 0 && markers.length === 0) return null

  return (
    <div className="absolute top-2 right-2 flex flex-col gap-1 z-50">
      {markers.map((m, i) => (
        <div
          key={i}
          className="text-xs px-2 py-1 rounded"
          style={{ backgroundColor: m.color + '20', color: m.color }}
        >
          {'\uD83D\uDCCC'} {m.node}: {m.label}
        </div>
      ))}
      {speed !== 1.0 && (
        <div className="text-xs px-2 py-1 rounded bg-rh-teal-10 text-el-water">
          Speed: {speed}x
        </div>
      )}
    </div>
  )
}
