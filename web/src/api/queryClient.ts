import { QueryClient } from '@tanstack/react-query'

/** Shared React Query cache policy: Lazy routes + Async fetches + Cache reuse. */
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 10_000,
      gcTime: 5 * 60_000,
      refetchOnWindowFocus: true,
      retry: 1,
    },
  },
})

/** Stable query keys — keep cache hits across pages. */
export const qk = {
  ready: ['ready'] as const,
  hosts: ['hosts'] as const,
  containers: (filters?: { q?: string; state?: string; stack?: string }) =>
    ['containers', filters ?? {}] as const,
  container: (id: string) => ['container', id] as const,
  containerInspect: (id: string, redact: boolean) => ['container-inspect', id, redact] as const,
  containerLogs: (id: string, tail: number, timestamps: boolean) =>
    ['container-logs', id, tail, timestamps] as const,
  stacks: ['stacks'] as const,
  networks: (filters?: { q?: string; driver?: string }) => ['networks', filters ?? {}] as const,
  volumes: (filters?: { q?: string; stack?: string }) => ['volumes', filters ?? {}] as const,
  images: (filters?: { q?: string; dangling?: string }) => ['images', filters ?? {}] as const,
  graph: (scope: string, stack: string) => ['graph', { scope, stack }] as const,
  systemResources: ['system', 'resources'] as const,
  systemInfo: ['system', 'info'] as const,
  systemSettings: ['system', 'settings'] as const,
  diagnostics: ['diagnostics'] as const,
  snapshots: ['snapshots'] as const,
  provenance: (id?: string) => (id ? (['provenance', id] as const) : (['provenance'] as const)),
  commands: (kind?: string, ref?: string) => ['commands', kind, ref] as const,
  metricsHistory: (host: string, scope: string, id: string | undefined, hours: number) =>
    ['metrics', 'history', host, scope, id, hours] as const,
}
