import { getAuthToken } from './prefs'

export type LogStreamHandle = { stop: () => void }

export function logsWsURL(id: string, opts: { tail: number; timestamps: boolean }) {
  const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const u = new URL(`${proto}//${window.location.host}/api/v1/containers/${encodeURIComponent(id)}/logs/ws`)
  u.searchParams.set('tail', String(opts.tail))
  if (opts.timestamps) u.searchParams.set('timestamps', 'true')
  const token = getAuthToken()
  if (token) u.searchParams.set('access_token', token)
  return u.toString()
}

/** Follow container logs over a dedicated WebSocket. */
export function startLogStream(
  id: string,
  opts: { tail: number; timestamps: boolean },
  onChunk: (text: string) => void,
  onStatus?: (msg: string) => void,
): LogStreamHandle {
  let ws: WebSocket | null = null
  let stopped = false

  const connect = () => {
    if (stopped) return
    ws = new WebSocket(logsWsURL(id, opts))
    ws.onopen = () => onStatus?.('live')
    ws.onmessage = (ev) => {
      try {
        const msg = JSON.parse(String(ev.data)) as {
          type?: string
          data?: { text?: string; message?: string; connected?: boolean }
        }
        if (msg.type === 'container.logs' && msg.data?.text) {
          onChunk(msg.data.text)
          return
        }
        if (msg.type === 'error') {
          onStatus?.(msg.data?.message ?? 'stream error')
        }
      } catch {
        /* ignore */
      }
    }
    ws.onclose = () => {
      if (!stopped) onStatus?.('disconnected')
    }
    ws.onerror = () => {
      ws?.close()
    }
  }

  connect()

  return {
    stop: () => {
      stopped = true
      ws?.close()
    },
  }
}
