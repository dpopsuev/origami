import { useEffect, useRef, useState, useCallback } from 'react'

export interface WSCommand {
  action: string
  [key: string]: unknown
}

interface UseKamiWSOptions {
  url: string
}

export function useKamiWS({ url }: UseKamiWSOptions) {
  const [commands, setCommands] = useState<WSCommand[]>([])
  const [connected, setConnected] = useState(false)
  const wsRef = useRef<WebSocket | null>(null)

  const connect = useCallback(() => {
    if (wsRef.current) wsRef.current.close()

    const ws = new WebSocket(url)
    wsRef.current = ws

    ws.onopen = () => setConnected(true)
    ws.onclose = () => setConnected(false)
    ws.onerror = () => setConnected(false)

    ws.onmessage = (msg) => {
      try {
        const cmd: WSCommand = JSON.parse(msg.data)
        setCommands((prev) => [...prev, cmd])
      } catch {
        // skip malformed
      }
    }
  }, [url])

  useEffect(() => {
    connect()
    return () => {
      wsRef.current?.close()
    }
  }, [connect])

  const send = useCallback((data: unknown) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(data))
    }
  }, [])

  return { commands, connected, send }
}
