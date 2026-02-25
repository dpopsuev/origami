import type { KamiEvent } from './hooks/useSSE'

interface OrigamiBridge {
  events: KamiEvent[]
  connected: boolean
  version: string
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

let stateRef: { events: KamiEvent[]; connected: boolean } = {
  events: [],
  connected: false,
}

export function initBridge() {
  window.__origami = {
    get events() {
      return stateRef.events
    },
    get connected() {
      return stateRef.connected
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
  stateRef = { events, connected }
}
