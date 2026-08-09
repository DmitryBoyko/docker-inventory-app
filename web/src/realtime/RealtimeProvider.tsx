import { useQueryClient } from '@tanstack/react-query'
import { useEffect, type ReactNode } from 'react'
import { HOST_CHANGE_EVENT } from '../lib/prefs'
import { onSnapshotUpdated, resetLiveHistory } from './store'
import { startRealtime } from './wsClient'

/** Coalesce burst snapshot events into one Async cache invalidation. */
function debounce(fn: () => void, ms: number) {
  let t: ReturnType<typeof setTimeout> | undefined
  return () => {
    if (t) clearTimeout(t)
    t = setTimeout(fn, ms)
  }
}

function invalidateKeys(qc: ReturnType<typeof useQueryClient>, keys: readonly string[]) {
  for (const key of keys) {
    void qc.invalidateQueries({ queryKey: [key] })
  }
}

/** Map backend snapshot.kind → React Query roots (P1 scoped invalidation). */
function keysForKind(kind: string): readonly string[] {
  switch (kind) {
    case 'system':
      return ['system', 'ready']
    case 'inventory':
      return ['containers', 'stacks', 'networks', 'volumes', 'images', 'graph', 'diagnostics', 'container']
    default:
      return [
        'containers',
        'stacks',
        'networks',
        'volumes',
        'images',
        'graph',
        'system',
        'ready',
        'hosts',
        'diagnostics',
        'container',
      ]
  }
}

export function RealtimeProvider({ children }: { children: ReactNode }) {
  const qc = useQueryClient()

  useEffect(() => {
    let handle = startRealtime()
    const pending = new Set<string>()

    const flush = debounce(() => {
      const keys = [...pending]
      pending.clear()
      invalidateKeys(qc, keys)
    }, 400)

    const off = onSnapshotUpdated((n) => {
      // Stats tick ~1 Hz — only refresh host resource card, not full inventory.
      if (n.kind === 'stats') {
        void qc.invalidateQueries({ queryKey: ['system', 'resources'] })
        return
      }
      for (const k of keysForKind(n.kind)) pending.add(k)
      flush()
    })

    const onHostChange = () => {
      resetLiveHistory()
      handle.stop()
      handle = startRealtime()
      void qc.invalidateQueries({ queryKey: ['ready'] })
      void qc.invalidateQueries({ queryKey: ['hosts'] })
      void qc.invalidateQueries({ queryKey: ['metrics'] })
    }
    window.addEventListener(HOST_CHANGE_EVENT, onHostChange)

    return () => {
      window.removeEventListener(HOST_CHANGE_EVENT, onHostChange)
      off()
      handle.stop()
    }
  }, [qc])

  return children
}
