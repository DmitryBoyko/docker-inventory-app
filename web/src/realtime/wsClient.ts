import { wsURL } from '../api/client'
import {
  applyConnection,
  applyDockerEvent,
  applyEventsStatus,
  applySnapshot,
  applyStats,
  setWsConnected,
} from './store'
import type { WsEnvelope } from './wsEnvelope'

export type RealtimeHandle = {
  stop: () => void
}

export function startRealtime(): RealtimeHandle {
  let ws: WebSocket | null = null
  let stopped = false
  let retry = 1000
  let pingTimer: number | undefined

  const connect = () => {
    if (stopped) return
    ws = new WebSocket(wsURL()) // includes access_token when set

    ws.onopen = () => {
      setWsConnected(true)
      retry = 1000
      ws?.send(JSON.stringify({ action: 'subscribe', channel: 'stats', filters: {} }))
      pingTimer = window.setInterval(() => {
        if (ws?.readyState === WebSocket.OPEN) {
          ws.send(JSON.stringify({ action: 'ping' }))
        }
      }, 15000)
    }

    ws.onmessage = (ev) => {
      let msg: WsEnvelope
      try {
        msg = JSON.parse(String(ev.data)) as WsEnvelope
      } catch {
        return
      }
      switch (msg.type) {
        case 'container.stats':
          applyStats(msg.data ?? [])
          break
        case 'snapshot.updated':
          applySnapshot(msg.data)
          break
        case 'docker.event':
          applyDockerEvent(msg.data)
          break
        case 'connection.status':
          applyConnection(msg.data)
          break
        case 'events.status':
          applyEventsStatus(msg.data)
          break
        case 'ping':
          ws?.send(JSON.stringify({ action: 'ping' }))
          break
        default:
          break
      }
    }

    ws.onclose = () => {
      setWsConnected(false)
      if (pingTimer) window.clearInterval(pingTimer)
      if (stopped) return
      window.setTimeout(connect, retry)
      retry = Math.min(retry * 2, 15000)
    }

    ws.onerror = () => {
      ws?.close()
    }
  }

  connect()

  return {
    stop: () => {
      stopped = true
      if (pingTimer) window.clearInterval(pingTimer)
      ws?.close()
    },
  }
}
