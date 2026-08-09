import { useQueryClient } from '@tanstack/react-query'
import { useEffect, type ReactNode } from 'react'
import { onSnapshotUpdated } from './store'
import { startRealtime } from './wsClient'

export function RealtimeProvider({ children }: { children: ReactNode }) {
  const qc = useQueryClient()

  useEffect(() => {
    const handle = startRealtime()
    const off = onSnapshotUpdated((n) => {
      if (n.kind === 'stats') {
        // Stats overlay comes from WS; only lightly refresh rollups.
        void qc.invalidateQueries({ queryKey: ['system', 'resources'] })
        return
      }
      void qc.invalidateQueries({ queryKey: ['containers'] })
      void qc.invalidateQueries({ queryKey: ['stacks'] })
      void qc.invalidateQueries({ queryKey: ['networks'] })
      void qc.invalidateQueries({ queryKey: ['volumes'] })
      void qc.invalidateQueries({ queryKey: ['images'] })
      void qc.invalidateQueries({ queryKey: ['graph'] })
      void qc.invalidateQueries({ queryKey: ['system'] })
    })
    return () => {
      off()
      handle.stop()
    }
  }, [qc])

  return children
}
