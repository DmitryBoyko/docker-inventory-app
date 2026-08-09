import { useQueryClient } from '@tanstack/react-query'
import { useEffect, type ReactNode } from 'react'
import { HOST_CHANGE_EVENT } from '../lib/prefs'
import { onSnapshotUpdated } from './store'
import { startRealtime } from './wsClient'

export function RealtimeProvider({ children }: { children: ReactNode }) {
  const qc = useQueryClient()

  useEffect(() => {
    let handle = startRealtime()
    const off = onSnapshotUpdated((n) => {
      if (n.kind === 'stats') {
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
      void qc.invalidateQueries({ queryKey: ['ready'] })
      void qc.invalidateQueries({ queryKey: ['hosts'] })
    })

    const onHostChange = () => {
      handle.stop()
      handle = startRealtime()
      void qc.invalidateQueries({ queryKey: ['ready'] })
      void qc.invalidateQueries({ queryKey: ['hosts'] })
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
