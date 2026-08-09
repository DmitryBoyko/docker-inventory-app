import {
  collectExposureRoutes,
  containerExposureScope,
  portExposureToScope,
  routesForContainer,
} from './exposure'
import type { Container } from '../api/types'

function assert(cond: boolean, msg: string) {
  if (!cond) throw new Error(msg)
}

const c: Container = {
  id: 'abc',
  idShort: 'abc',
  name: 'web',
  stack: 'app',
  image: 'nginx',
  state: 'running',
  health: 'healthy',
  restartCount: 0,
  writableLayer: { bytes: 1, available: true },
  ports: [
    { hostIP: '0.0.0.0', hostPort: 443, containerPort: 443, protocol: 'tcp', exposure: 'public' },
    { containerPort: 80, protocol: 'tcp', exposure: 'internal' },
  ],
}

assert(portExposureToScope('public') === 'external', 'public→external')
assert(portExposureToScope('specific') === 'lan', 'specific→lan')
assert(containerExposureScope(c) === 'external', 'summary')
const routes = routesForContainer(c)
assert(routes.length === 1 && routes[0].hostIP === '*', 'display *')
assert(collectExposureRoutes([c])[0].stack === 'app', 'stack')
assert(collectExposureRoutes([c]).length === 1, 'collect')

console.log('exposure.test.ts ok')
