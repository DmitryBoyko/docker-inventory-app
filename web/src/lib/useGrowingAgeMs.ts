import { useEffect, useState } from 'react'

/**
 * Grow a server-reported snapshot age between React Query refetches
 * so "updated N ago" stays truthful on screen.
 */
export function useGrowingAgeMs(
  snapshotAgeMs: number | undefined,
  dataUpdatedAt: number | undefined,
): number | undefined {
  const [now, setNow] = useState(() => Date.now())

  useEffect(() => {
    const id = window.setInterval(() => setNow(Date.now()), 1000)
    return () => window.clearInterval(id)
  }, [])

  if (snapshotAgeMs == null) return undefined
  if (!dataUpdatedAt) return snapshotAgeMs
  return snapshotAgeMs + Math.max(0, now - dataUpdatedAt)
}
