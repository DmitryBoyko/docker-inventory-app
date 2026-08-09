import type { Container, ContainerPort, ExposureScope, ExternalExposure } from '../api/types'

export type { ExposureScope, ExternalExposure }

export type ExposureRouteRow = ExternalExposure & {
  containerId: string
  containerIdShort: string
  containerName: string
  stack: string
  health: string
  state: string
}

const scopeRank: Record<ExposureScope, number> = {
  external: 3,
  lan: 2,
  localhost: 1,
  internal: 0,
}

/** Map Engine Port.exposure → UI scope. */
export function portExposureToScope(exposure?: string): ExposureScope {
  switch (exposure) {
    case 'public':
    case 'external':
      return 'external'
    case 'localhost':
      return 'localhost'
    case 'specific':
    case 'lan':
      return 'lan'
    default:
      return 'internal'
  }
}

export function containerExposureScope(c: Container): ExposureScope {
  if (c.exposureScope) return c.exposureScope
  let best: ExposureScope = 'internal'
  for (const p of c.ports ?? []) {
    const s = portExposureToScope(p.exposure)
    if (scopeRank[s] > scopeRank[best]) best = s
  }
  if ((c.externalExposure?.length ?? 0) > 0 && best === 'internal') {
    for (const r of c.externalExposure!) {
      if (scopeRank[r.scope] > scopeRank[best]) best = r.scope
    }
  }
  return best
}

export function routesForContainer(c: Container): ExternalExposure[] {
  if (c.externalExposure?.length) return c.externalExposure
  const out: ExternalExposure[] = []
  for (const p of c.ports ?? []) {
    const scope = portExposureToScope(p.exposure)
    if (!p.hostPort || scope === 'internal') continue
    out.push({
      hostIP: displayHostIP(p),
      hostPort: p.hostPort,
      containerPort: `${p.containerPort}/${p.protocol || 'tcp'}`,
      scope,
    })
  }
  return out
}

function displayHostIP(p: ContainerPort): string {
  if (p.exposure === 'public' || !p.hostIP || p.hostIP === '0.0.0.0' || p.hostIP === '::') {
    return '*'
  }
  return p.hostIP
}

export function collectExposureRoutes(containers: Container[]): ExposureRouteRow[] {
  const rows: ExposureRouteRow[] = []
  for (const c of containers) {
    for (const r of routesForContainer(c)) {
      rows.push({
        ...r,
        containerId: c.id,
        containerIdShort: c.idShort,
        containerName: c.name,
        stack: c.stack || 'standalone',
        health: c.health,
        state: c.state,
      })
    }
  }
  rows.sort((a, b) => {
    const d = scopeRank[b.scope] - scopeRank[a.scope]
    if (d !== 0) return d
    if (a.hostPort !== b.hostPort) return a.hostPort - b.hostPort
    return a.containerName.localeCompare(b.containerName)
  })
  return rows
}

export function countByScope(routes: ExposureRouteRow[]) {
  return {
    external: routes.filter((r) => r.scope === 'external').length,
    localhost: routes.filter((r) => r.scope === 'localhost').length,
    lan: routes.filter((r) => r.scope === 'lan').length,
  }
}
