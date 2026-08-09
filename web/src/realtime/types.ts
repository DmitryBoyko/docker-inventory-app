import type { ConnectionStatus, ContainerStats } from '../api/types'

export type StatsItem = {
  id: string
  idShort: string
  name: string
  stack: string
  stats: ContainerStats | null
}

export type SnapshotNotice = {
  version: number
  snapshotAt?: string
  statsAt?: string
  systemAt?: string
  kind: string
}

export type DockerEvent = {
  type: string
  action: string
  actorId: string
  actorName?: string
  time: string
}

export type EventsStatus = {
  connected: boolean
  error?: string
}

export type HistoryPoint = {
  t: number
  cpu: number
  mem: number
}

export type LiveState = {
  connected: boolean
  docker: ConnectionStatus | null
  events: EventsStatus | null
  statsById: Record<string, StatsItem>
  history: HistoryPoint[]
  lastSnapshotVersion: number
  lastEvent: DockerEvent | null
}

export type { ConnectionStatus }
