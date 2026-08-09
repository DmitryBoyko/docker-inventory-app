import type { AggregateBytes, ByteMetric } from '../api/types'
import { en } from '../i18n/locales/en'
import { ru } from '../i18n/locales/ru'
import { getLocale } from './prefs'

const units = ['B', 'KiB', 'MiB', 'GiB', 'TiB'] as const

function tr(key: string, params?: Record<string, string | number>): string {
  let locale: 'en' | 'ru' = 'en'
  try {
    locale = getLocale()
  } catch {
    locale = 'en'
  }
  const catalog = locale === 'ru' ? ru : en
  let text = catalog[key] ?? en[key] ?? key
  if (params) {
    for (const [k, v] of Object.entries(params)) {
      text = text.replaceAll(`{${k}}`, String(v))
    }
  }
  return text
}

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
    return m.reason ? tr('format.naReason', { reason: m.reason }) : tr('common.na')
  }
  const base = formatBytes(m.bytes)
  if ('partial' in m && m.partial) return tr('format.partialBytes', { value: base })
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
  if (ms == null || Number.isNaN(ms) || ms < 0) return '—'
  if (ms < 1500) return tr('format.ageJustNow')
  if (ms < 60_000) {
    const n = Math.max(1, Math.round(ms / 1000))
    return tr('format.ageSeconds', { n })
  }
  if (ms < 3600_000) {
    const n = Math.max(1, Math.round(ms / 60_000))
    return tr('format.ageMinutes', { n })
  }
  if (ms < 86400_000) {
    const n = Math.max(1, Math.round(ms / 3600_000))
    return tr('format.ageHours', { n })
  }
  const n = Math.max(1, Math.round(ms / 86400_000))
  return tr('format.ageDays', { n })
}

