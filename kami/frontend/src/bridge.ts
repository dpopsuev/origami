import type { KamiEvent } from './hooks/useSSE'

interface KamiSelection {
  elements: { type: string; id: string; kamiKey: string }[]
  timestamp: string
}

interface OrigamiBridge {
  events: KamiEvent[]
  connected: boolean
  version: string
  selection: KamiSelection | null
  getEvents: () => KamiEvent[]
  getLastEvent: () => KamiEvent | null
  getEventsByType: (type: string) => KamiEvent[]
  getNodeEvents: (node: string) => KamiEvent[]
}

declare global {
  interface Window {
    __origami?: OrigamiBridge
  }
}

let stateRef: { events: KamiEvent[]; connected: boolean; selection: KamiSelection | null } = {
  events: [],
  connected: false,
  selection: null,
}

export function initBridge() {
  window.__origami = {
    get events() {
      return stateRef.events
    },
    get connected() {
      return stateRef.connected
    },
    get selection() {
      return stateRef.selection
    },
    set selection(val: KamiSelection | null) {
      stateRef.selection = val
    },
    version: '1.0.0',
    getEvents: () => [...stateRef.events],
    getLastEvent: () =>
      stateRef.events.length > 0
        ? stateRef.events[stateRef.events.length - 1]
        : null,
    getEventsByType: (type: string) =>
      stateRef.events.filter((e) => e.type === type),
    getNodeEvents: (node: string) =>
      stateRef.events.filter((e) => e.node === node),
  }
}

export function updateBridge(events: KamiEvent[], connected: boolean) {
  stateRef = { events, connected, selection: stateRef.selection }
}
