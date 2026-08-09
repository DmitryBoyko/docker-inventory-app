import type { AggregateBytes, ByteMetric } from '../api/types'

const units = ['B', 'KiB', 'MiB', 'GiB', 'TiB'] as const

export function formatBytes(n: number | null | undefined): string {
  if (n == null || Number.isNaN(n)) return '—'
  let v = Math.abs(n)
  let i = 0
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024
    i += 1
  }
  const sign = n < 0 ? '-' : ''
  const digits = i === 0 ? 0 : v < 10 ? 2 : v < 100 ? 1 : 0
  return `${sign}${v.toFixed(digits)} ${units[i]}`
}

export function formatByteMetric(m: ByteMetric | AggregateBytes | undefined): string {
  if (!m) return '—'
  if (!m.available || m.bytes == null) {
    return m.reason ? `n/a (${m.reason})` : 'n/a'
  }
  const base = formatBytes(m.bytes)
  if ('partial' in m && m.partial) return `${base} (partial)`
  return base
}

export function formatCpu(pct: number | undefined): string {
  if (pct == null || Number.isNaN(pct)) return '—'
  return `${pct.toFixed(1)}%`
}

export function formatUptime(seconds: number | null | undefined): string {
  if (seconds == null || seconds < 0) return '—'
  const s = Math.floor(seconds)
  const d = Math.floor(s / 86400)
  const h = Math.floor((s % 86400) / 3600)
  const m = Math.floor((s % 3600) / 60)
  if (d > 0) return `${d}d ${h}h`
  if (h > 0) return `${h}h ${m}m`
  return `${m}m`
}

export function formatAgeMs(ms: number | undefined): string {
  if (ms == null) return '—'
  if (ms < 1000) return `${ms} ms`
  return `${(ms / 1000).toFixed(1)} s`
}
