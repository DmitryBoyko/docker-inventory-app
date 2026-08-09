import type { DockerEvent, EventsStatus, HistoryPoint, LiveState, SnapshotNotice, StatsItem } from './types'
import type { ConnectionStatus, ContainerStats } from '../api/types'

const MAX_HISTORY = 60

type Listener = () => void

let state: LiveState = {
  connected: false,
  docker: null,
  events: null,
  statsById: {},
  history: [],
  lastSnapshotVersion: 0,
  lastEvent: null,
}

const listeners = new Set<Listener>()
const snapshotListeners = new Set<(n: SnapshotNotice) => void>()

export function getLiveState() {
  return state
}

export function subscribeLive(listener: Listener) {
  listeners.add(listener)
  return () => {
    listeners.delete(listener)
  }
}

export function onSnapshotUpdated(listener: (n: SnapshotNotice) => void) {
  snapshotListeners.add(listener)
  return () => snapshotListeners.delete(listener)
}

function emit() {
  for (const l of listeners) l()
}

function setState(patch: Partial<LiveState>) {
  state = { ...state, ...patch }
  emit()
}

export function setWsConnected(connected: boolean) {
  setState({ connected })
}

/** Clear ring-buffer history (e.g. host switch). */
export function resetLiveHistory() {
  setState({ history: [], statsById: {} })
}

export function applyConnection(docker: ConnectionStatus) {
  setState({ docker })
}

export function applyEventsStatus(events: EventsStatus) {
  setState({ events })
}

export function applyStats(items: StatsItem[]) {
  const statsById = { ...state.statsById }
  let cpu = 0
  let mem = 0
  for (const it of items) {
    statsById[it.id] = it
    if (it.idShort) statsById[it.idShort] = it
    if (it.stats) {
      cpu += it.stats.cpuPercent
      mem += it.stats.memoryBytes
    }
  }
  const point: HistoryPoint = { t: Date.now(), cpu, mem }
  const history = [...state.history, point].slice(-MAX_HISTORY)
  setState({ statsById, history })
}

export function applySnapshot(n: SnapshotNotice) {
  setState({ lastSnapshotVersion: n.version })
  for (const l of snapshotListeners) l(n)
}

export function applyDockerEvent(ev: DockerEvent) {
  setState({ lastEvent: ev })
}

function lookupStats(id: string, idShort?: string): ContainerStats | undefined {
  const live = state.statsById[id] || (idShort ? state.statsById[idShort] : undefined)
  return live?.stats ?? undefined
}

/**
 * Merge live WS stats into container rows.
 * Reuses previous row references when the live stats object is unchanged (P1).
 */
export function mergeContainerStats<T extends { id: string; idShort?: string; stats?: ContainerStats | null }>(
  containers: T[],
  previous?: T[],
): T[] {
  if (containers.length === 0) return previous?.length === 0 && previous ? previous : containers

  const prevById = previous ? new Map(previous.map((c) => [c.id, c] as const)) : null
  let reused = 0
  const out = containers.map((c) => {
    const liveStats = lookupStats(c.id, c.idShort)
    const prev = prevById?.get(c.id)
    if (liveStats) {
      if (prev && prev.stats === liveStats) {
        reused++
        return prev
      }
      return { ...c, stats: liveStats } as T
    }
    if (prev && prev.stats == null && prev === c) {
      reused++
      return prev
    }
    return c
  })

  if (previous && reused === out.length && previous.length === out.length) {
    return previous
  }
  return out
}

export function getContainerLiveStats(id: string, idShort?: string): ContainerStats | undefined {
  return lookupStats(id, idShort)
}
