import { useEffect, useState } from 'react'

/** Format an ISO timestamp as `YYYY-MM-DD HH:mm:ss UTC`. */
export function formatUtcClock(iso: string | undefined): string {
  if (!iso) return '—'
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return '—'
  const y = d.getUTCFullYear()
  const mo = String(d.getUTCMonth() + 1).padStart(2, '0')
  const da = String(d.getUTCDate()).padStart(2, '0')
  const h = String(d.getUTCHours()).padStart(2, '0')
  const mi = String(d.getUTCMinutes()).padStart(2, '0')
  const s = String(d.getUTCSeconds()).padStart(2, '0')
  return `${y}-${mo}-${da} ${h}:${mi}:${s} UTC`
}

/**
 * Advance a server clock between API refreshes so the Engine panel
 * shows a live UTC time based on the last /info sample.
 */
export function useServerUtcClock(
  systemTimeUtc: string | undefined,
  dataUpdatedAt: number | undefined,
): string {
  const [now, setNow] = useState(() => Date.now())

  useEffect(() => {
    const id = window.setInterval(() => setNow(Date.now()), 1000)
    return () => window.clearInterval(id)
  }, [])

  if (!systemTimeUtc) return '—'
  const base = Date.parse(systemTimeUtc)
  if (Number.isNaN(base)) return '—'
  const elapsed = dataUpdatedAt ? Math.max(0, now - dataUpdatedAt) : 0
  return formatUtcClock(new Date(base + elapsed).toISOString())
}
