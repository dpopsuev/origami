import { useEffect, useRef, useState, useCallback } from 'react'

export interface KamiEvent {
  type: string
  ts: string
  agent?: string
  node?: string
  edge?: string
  zone?: string
  case_id?: string
  elapsed_ms?: number
  error?: string
  data?: Record<string, unknown>
}

interface UseSSEOptions {
  url: string
  maxEvents?: number
}

export function useSSE({ url, maxEvents = 200 }: UseSSEOptions) {
  const [events, setEvents] = useState<KamiEvent[]>([])
  const [connected, setConnected] = useState(false)
  const esRef = useRef<EventSource | null>(null)

  const connect = useCallback(() => {
    if (esRef.current) esRef.current.close()

    const es = new EventSource(url)
    esRef.current = es

    es.onopen = () => setConnected(true)
    es.onerror = () => setConnected(false)
    es.onmessage = (msg) => {
      try {
        const evt: KamiEvent = JSON.parse(msg.data)
        setEvents((prev) => {
          const next = [...prev, evt]
          return next.length > maxEvents ? next.slice(-maxEvents) : next
        })
      } catch {
        // skip malformed events
      }
    }
  }, [url, maxEvents])

  useEffect(() => {
    connect()
    return () => {
      esRef.current?.close()
    }
  }, [connect])

  const clear = useCallback(() => setEvents([]), [])

  return { events, connected, clear }
}
