export type ByteMetric = {
  bytes: number | null
  available: boolean
  reason?: string
}

export type AggregateBytes = {
  bytes: number | null
  available: boolean
  partial: boolean
  unknownCount: number
  reason?: string
}

export type ContainerStats = {
  timestamp: string
  cpuPercent: number
  memoryBytes: number
  memoryLimitBytes: number
  memoryPercent: number
  networkRxBytes: number
  networkTxBytes: number
  blockReadBytes: number
  blockWriteBytes: number
  countersAvailable: boolean
}

export type ContainerPort = {
  hostIP?: string
  hostPort?: number
  containerPort: number
  protocol: string
  exposure?: string
}

export type ExposureScope = 'external' | 'localhost' | 'lan' | 'internal'

export type ExternalExposure = {
  hostIP: string
  hostPort: number
  containerPort: string
  scope: Exclude<ExposureScope, 'internal'>
}

export type ContainerEndpoint = {
  networkId?: string
  networkName: string
  ipAddress?: string
  gateway?: string
  macAddress?: string
}

export type ContainerMount = {
  type: string
  name?: string
  source?: string
  destination: string
  rw: boolean
}

export type Container = {
  id: string
  idShort: string
  name: string
  stack: string
  service?: string | null
  image: string
  state: string
  status?: string
  health: string
  restartCount: number
  uptimeSeconds?: number | null
  writableLayer: ByteMetric
  stats?: ContainerStats | null
  ports?: ContainerPort[]
  externalExposure?: ExternalExposure[]
  exposureScope?: ExposureScope
  endpoints?: ContainerEndpoint[]
  mounts?: ContainerMount[]
}

export type ResourceSummary = {
  cpuPercent: number
  memoryBytes: number
  writableLayer: AggregateBytes
  volumeData: AggregateBytes
  containerCount: number
  runningCount: number
}

export type Stack = {
  name: string
  containers: Array<{
    id: string
    idShort: string
    name: string
    state: string
    health: string
  }>
  resources: ResourceSummary
  volumeUsage: AggregateBytes
  unhealthyCount: number
  restartedCount: number
}

export type Network = {
  id: string
  idShort: string
  name: string
  driver: string
  scope?: string
  internal: boolean
  attachable?: boolean
  ingress?: boolean
  containers?: string[]
  stacks?: string[]
}

export type Volume = {
  name: string
  driver: string
  mountpoint?: string
  scope?: string
  usage: ByteMetric & { links?: number | null }
  containers?: string[]
  stacks?: string[]
  shared?: boolean
}

export type GraphNode = {
  id: string
  type: 'stack' | 'service' | 'container' | 'network' | 'volume' | 'image' | string
  label: string
  data?: Record<string, unknown>
}

export type GraphEdge = {
  id: string
  type: 'contains' | 'runs' | 'attached' | 'mounts' | 'uses_image' | string
  source: string
  target: string
}

export type Graph = {
  scope: string
  stack?: string
  nodes: GraphNode[]
  edges: GraphEdge[]
}

export type Image = {
  id: string
  idShort: string
  repoTags?: string[]
  repoDigests?: string[]
  sizeBytes: number
  sharedSizeBytes?: number | null
  createdAt?: string | null
  containers?: string[]
  containerCount: number
  dangling: boolean
}

export type SystemResources = ResourceSummary & {
  networkRxBytes: number
  networkTxBytes: number
  blockReadBytes: number
  blockWriteBytes: number
}

export type SystemInfo = {
  id?: string
  name?: string
  serverVersion: string
  apiVersion?: string
  os: string
  osVersion?: string
  osType?: string
  architecture: string
  kernelVersion?: string
  cpus: number
  memoryBytes: number
  driver?: string
  dockerRootDir?: string
  containers: number
  containersRunning: number
  containersPaused: number
  containersStopped: number
  images: number
}

export type ApiEnvelope<T> = {
  timestamp: string
  snapshotAt?: string | null
  snapshotAgeMs?: number
  statsAt?: string | null
  systemAt?: string | null
  collectError?: string | null
  data: T
}

export type ConnectionStatus = {
  connected: boolean
  host: string
  source: string
  context?: string
  apiVersion?: string
  osType?: string
  checkedAt: string
  error?: string
}

export type HostInfo = {
  name: string
  endpoint: string
  source: string
  context?: string
  isDefault: boolean
  connected: boolean
  error?: string
}

export type ReadyResponse = {
  ready: boolean
  host?: string
  docker: ConnectionStatus
  events?: {
    connected: boolean
    error?: string | null
    host?: string
  }
  timestamp: string
  error?: {
    code: string
    message: string
  }
}

export type ApiErrorBody = {
  error: {
    code: string
    message: string
  }
  timestamp: string
}

export type MetricsPoint = {
  t: number
  cpu: number
  mem: number
  netRx?: number
  netTx?: number
}

export type MetricsHistory = {
  scope: string
  host: string
  id?: string
  from: string
  to: string
  stepSeconds: number
  points: MetricsPoint[]
}

export type SystemSettings = {
  listen: string
  listenLoopback: boolean
  authEnabled: boolean
  dockerTimeout: string
  intervals: Record<string, string>
  version: string
  commit: string
  uiEmbedded: boolean
  defaultHost?: string
  hosts?: HostInfo[]
  metrics?: {
    enabled: boolean
    dbPath?: string
    interval?: string
    retention?: string
  }
  defaults: {
    inspectRedact: boolean
  }
}

export type RiskLevel = 'READ_ONLY' | 'INTERACTIVE' | 'STATE_CHANGING' | 'DESTRUCTIVE'
export type FindingSeverity = 'INFO' | 'WARNING' | 'CRITICAL'
export type EntityKind = 'container' | 'network' | 'volume' | 'image' | 'system' | 'stack'

export type RenderedCommand = {
  definitionId: string
  titleKey: string
  descriptionKey: string
  title: string
  description: string
  category: string
  entityKind: EntityKind
  riskLevel: RiskLevel
  requiresTTY?: boolean
  shell: 'bash' | 'powershell' | 'cmd'
  command: string
  entityRef: string
}

export type Finding = {
  id: string
  ruleId: string
  severity: FindingSeverity
  entity: { kind: string; id?: string; name?: string }
  titleKey: string
  descriptionKey: string
  reasonKey: string
  recommendationKey: string
  title: string
  description: string
  reason: string
  recommendation: string
  evidence?: Record<string, unknown>
  relatedCommands?: string[]
}

export type ProvenanceSpec = {
  id: string
  title: string
  titleKey: string
  entityKind: string
  apiEndpoint: string
  dockerField?: string
  transformation?: string
  transformationKey?: string
  description: string
  descriptionKey: string
  relatedCommands?: string[]
  chain?: string[]
}

export type SnapshotMeta = {
  id: string
  createdAt: string
  hostName: string
  endpoint?: string
  context?: string
  dockerVersion?: string
  label?: string
  counts: {
    containers: number
    images: number
    networks: number
    volumes: number
    stacks: number
  }
}

export type SnapshotChange = {
  kind: 'added' | 'removed' | 'modified'
  id: string
  name: string
  fields?: { field: string; from?: unknown; to?: unknown }[]
}

export type SnapshotDiff = {
  leftId: string
  rightId: string
  leftAt?: string
  rightAt?: string
  containers: SnapshotChange[]
  images: SnapshotChange[]
  networks: SnapshotChange[]
  volumes: SnapshotChange[]
  stacks: SnapshotChange[]
}

