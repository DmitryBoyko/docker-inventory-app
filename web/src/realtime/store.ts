import type { DockerEvent, EventsStatus, HistoryPoint, LiveState, SnapshotNotice, StatsItem } from './types'
import type { ConnectionStatus } from '../api/types'

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
  return () => listeners.delete(listener)
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

export function mergeContainerStats<T extends { id: string; idShort?: string; stats?: unknown }>(
  containers: T[],
): T[] {
  return containers.map((c) => {
    const live = state.statsById[c.id] || (c.idShort ? state.statsById[c.idShort] : undefined)
    if (!live?.stats) return c
    return { ...c, stats: live.stats }
  })
}
