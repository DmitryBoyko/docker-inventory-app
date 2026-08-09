import { authHeaders, getAuthToken, getSelectedHost } from '../lib/prefs'
import type {
  ApiEnvelope,
  ApiErrorBody,
  Container,
  Graph,
  HostInfo,
  Image,
  Network,
  ReadyResponse,
  Stack,
  MetricsHistory,
  SystemInfo,
  SystemResources,
  SystemSettings,
  Volume,
} from './types'

const API = '/api/v1'

export class ApiError extends Error {
  status: number
  code: string

  constructor(status: number, code: string, message: string) {
    super(message)
    this.status = status
    this.code = code
  }
}

/** Append ?host= from local preference (ADR-014). */
function withHost(path: string): string {
  const host = getSelectedHost()
  if (!host) return path
  const join = path.includes('?') ? '&' : '?'
  return `${path}${join}host=${encodeURIComponent(host)}`
}

async function getJSON<T>(path: string): Promise<T> {
  const res = await fetch(withHost(path), { headers: authHeaders() })
  const text = await res.text()
  let body: unknown = null
  if (text) {
    try {
      body = JSON.parse(text) as unknown
    } catch {
      throw new ApiError(res.status, 'invalid_json', text.slice(0, 200))
    }
  }
  if (!res.ok) {
    const err = body as ApiErrorBody | null
    throw new ApiError(
      res.status,
      err?.error?.code ?? 'http_error',
      err?.error?.message ?? `HTTP ${res.status}`,
    )
  }
  return body as T
}

export async function fetchReady() {
  const res = await fetch(withHost(`${API}/ready`), { headers: authHeaders() })
  const body = (await res.json()) as ReadyResponse
  if (res.status === 401) {
    throw new ApiError(401, body.error?.code ?? 'unauthorized', body.error?.message ?? 'unauthorized')
  }
  // 503 still returns a usable ReadyResponse envelope.
  if (!res.ok && res.status !== 503) {
    throw new ApiError(res.status, body.error?.code ?? 'http_error', body.error?.message ?? `HTTP ${res.status}`)
  }
  return body
}

export function fetchHosts() {
  return getJSON<{
    timestamp: string
    defaultHost: string
    data: HostInfo[]
  }>(`${API}/hosts`)
}

export function fetchContainers(params?: { q?: string; stack?: string; state?: string }) {
  const qs = new URLSearchParams()
  if (params?.q) qs.set('q', params.q)
  if (params?.stack) qs.set('stack', params.stack)
  if (params?.state) qs.set('state', params.state)
  const suffix = qs.size ? `?${qs}` : ''
  return getJSON<ApiEnvelope<Container[]>>(`${API}/containers${suffix}`)
}

export function fetchContainer(id: string) {
  return getJSON<ApiEnvelope<Container>>(`${API}/containers/${encodeURIComponent(id)}`)
}

export function fetchContainerInspect(id: string, redact = true) {
  const qs = new URLSearchParams()
  qs.set('redact', redact ? 'true' : 'false')
  return getJSON<ApiEnvelope<{
    id: string
    name: string
    redacted: boolean
    redactedFields?: string[]
    inspect: unknown
  }>>(`${API}/containers/${encodeURIComponent(id)}/inspect?${qs}`)
}

export function fetchContainerLogs(
  id: string,
  params?: { tail?: number; since?: string; timestamps?: boolean },
) {
  const qs = new URLSearchParams()
  if (params?.tail != null) qs.set('tail', String(params.tail))
  if (params?.since) qs.set('since', params.since)
  if (params?.timestamps) qs.set('timestamps', 'true')
  const suffix = qs.size ? `?${qs}` : ''
  return getJSON<ApiEnvelope<{
    id: string
    name: string
    tail: number
    since?: string | null
    timestamps: boolean
    truncated: boolean
    text: string
    warning: string
  }>>(`${API}/containers/${encodeURIComponent(id)}/logs${suffix}`)
}

export function fetchStacks() {
  return getJSON<ApiEnvelope<Stack[]>>(`${API}/stacks`)
}

export function fetchNetworks(params?: { q?: string; driver?: string }) {
  const qs = new URLSearchParams()
  if (params?.q) qs.set('q', params.q)
  if (params?.driver) qs.set('driver', params.driver)
  const suffix = qs.size ? `?${qs}` : ''
  return getJSON<ApiEnvelope<Network[]>>(`${API}/networks${suffix}`)
}

export function fetchVolumes(params?: { q?: string; stack?: string }) {
  const qs = new URLSearchParams()
  if (params?.q) qs.set('q', params.q)
  if (params?.stack) qs.set('stack', params.stack)
  const suffix = qs.size ? `?${qs}` : ''
  return getJSON<ApiEnvelope<Volume[]>>(`${API}/volumes${suffix}`)
}

export function fetchImages(params?: { q?: string; dangling?: string }) {
  const qs = new URLSearchParams()
  if (params?.q) qs.set('q', params.q)
  if (params?.dangling) qs.set('dangling', params.dangling)
  const suffix = qs.size ? `?${qs}` : ''
  return getJSON<ApiEnvelope<Image[]>>(`${API}/images${suffix}`)
}

export function fetchGraph(params?: { scope?: string; stack?: string }) {
  const qs = new URLSearchParams()
  if (params?.scope) qs.set('scope', params.scope)
  if (params?.stack) qs.set('stack', params.stack)
  const suffix = qs.size ? `?${qs}` : ''
  return getJSON<ApiEnvelope<Graph>>(`${API}/graph${suffix}`)
}

export function fetchSystemResources() {
  return getJSON<ApiEnvelope<SystemResources>>(`${API}/system/resources`)
}

export function fetchSystemInfo() {
  return getJSON<ApiEnvelope<SystemInfo>>(`${API}/system/info`)
}

export function fetchSystemSettings() {
  return getJSON<ApiEnvelope<SystemSettings>>(`${API}/system/settings`)
}

export function fetchEntityCommands(kind: string, ref = '', shell?: string) {
  const qs = new URLSearchParams()
  if (ref) qs.set('ref', ref)
  if (shell) qs.set('shell', shell)
  const base =
    kind === 'system'
      ? `${API}/entities/system/commands`
      : `${API}/entities/${encodeURIComponent(kind)}/commands`
  const suffix = qs.size ? `?${qs}` : ''
  return getJSON<{
    timestamp: string
    host: string
    data: import('./types').RenderedCommand[]
  }>(`${base}${suffix}`)
}

export function fetchDiagnostics() {
  return getJSON<{
    timestamp: string
    host: string
    count: number
    data: import('./types').Finding[]
  }>(`${API}/diagnostics`)
}

export function fetchProvenance(id?: string) {
  if (id) {
    return getJSON<{ timestamp: string; data: import('./types').ProvenanceSpec }>(
      `${API}/provenance/${encodeURIComponent(id)}`,
    )
  }
  return getJSON<{ timestamp: string; data: import('./types').ProvenanceSpec[] }>(`${API}/provenance`)
}

export function fetchSnapshots() {
  return getJSON<{ timestamp: string; host: string; data: import('./types').SnapshotMeta[] }>(
    `${API}/snapshots`,
  )
}

export async function createSnapshot(label?: string) {
  const res = await fetch(withHost(`${API}/snapshots`), {
    method: 'POST',
    headers: { ...authHeaders(), 'Content-Type': 'application/json' },
    body: JSON.stringify({ label: label ?? '' }),
  })
  const body = await res.json()
  if (!res.ok) {
    throw new ApiError(res.status, body?.error?.code ?? 'http_error', body?.error?.message ?? `HTTP ${res.status}`)
  }
  return body as { timestamp: string; host: string; data: import('./types').SnapshotMeta }
}

export function fetchSnapshotDiff(id: string, against = 'current') {
  const qs = new URLSearchParams({ against })
  return getJSON<{ timestamp: string; host: string; data: import('./types').SnapshotDiff }>(
    `${API}/snapshots/${encodeURIComponent(id)}/diff?${qs}`,
  )
}

export async function deleteSnapshot(id: string) {
  const res = await fetch(withHost(`${API}/snapshots/${encodeURIComponent(id)}`), {
    method: 'DELETE',
    headers: authHeaders(),
  })
  if (!res.ok) {
    let message = `HTTP ${res.status}`
    try {
      const body = (await res.json()) as ApiErrorBody
      message = body.error?.message ?? message
    } catch {
      /* ignore */
    }
    throw new ApiError(res.status, 'delete_failed', message)
  }
}

export type MetricsHistoryScope = 'host' | 'container'

export function fetchMetricsHistory(params?: {
  scope?: MetricsHistoryScope
  id?: string
  from?: string
  to?: string
  step?: string
}) {
  const qs = new URLSearchParams()
  if (params?.scope) qs.set('scope', params.scope)
  if (params?.id) qs.set('id', params.id)
  if (params?.from) qs.set('from', params.from)
  if (params?.to) qs.set('to', params.to)
  if (params?.step) qs.set('step', params.step)
  const suffix = qs.size ? `?${qs}` : ''
  return getJSON<{
    timestamp: string
    host: string
    data: MetricsHistory
  }>(`${API}/metrics/history${suffix}`)
}

export type ExportFormat = 'json' | 'csv'
export type ExportScope = 'all' | 'containers' | 'stacks'

/** Download inventory export (structured PS replacement). */
export async function downloadExport(format: ExportFormat, scope: ExportScope = 'all') {
  const qs = new URLSearchParams({ format, scope })
  const res = await fetch(withHost(`${API}/export?${qs}`), { headers: authHeaders() })
  if (!res.ok) {
    let message = `HTTP ${res.status}`
    try {
      const body = (await res.json()) as ApiErrorBody
      message = body.error?.message ?? message
    } catch {
      /* ignore */
    }
    throw new ApiError(res.status, 'export_failed', message)
  }
  const blob = await res.blob()
  const cd = res.headers.get('Content-Disposition') ?? ''
  const m = /filename="([^"]+)"/i.exec(cd)
  const filename = m?.[1] ?? `docker-visualizer.${format}`
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.click()
  URL.revokeObjectURL(url)
}

/** Build WS URL including access_token and selected host when set. */
export function wsURL(): string {
  const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const u = new URL(`${proto}//${window.location.host}/api/v1/ws`)
  const token = getAuthToken()
  if (token) u.searchParams.set('access_token', token)
  const host = getSelectedHost()
  if (host) u.searchParams.set('host', host)
  return u.toString()
}
