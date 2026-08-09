import type { ConnectionStatus } from '../api/types'
import type { DockerEvent, EventsStatus, SnapshotNotice, StatsItem } from './types'

export type WsEnvelope =
  | { type: 'container.stats'; timestamp: string; data: StatsItem[] }
  | { type: 'snapshot.updated'; timestamp: string; data: SnapshotNotice }
  | { type: 'docker.event'; timestamp: string; data: DockerEvent }
  | { type: 'connection.status'; timestamp: string; data: ConnectionStatus }
  | { type: 'events.status'; timestamp: string; data: EventsStatus }
  | { type: 'ping' | 'pong' | 'error'; timestamp: string; data?: unknown }
