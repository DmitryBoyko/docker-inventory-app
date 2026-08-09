import { formatByteMetric, formatBytes, formatCpu } from './format'

function assert(cond: boolean, msg: string) {
  if (!cond) throw new Error(msg)
}

const memory = new Map<string, string>()
Object.defineProperty(globalThis, 'localStorage', {
  value: {
    getItem: (k: string) => (memory.has(k) ? memory.get(k)! : null),
    setItem: (k: string, v: string) => memory.set(k, String(v)),
    removeItem: (k: string) => memory.delete(k),
  },
  configurable: true,
})
Object.defineProperty(globalThis, 'navigator', {
  value: { language: 'en-US' },
  configurable: true,
})
memory.set('dv.locale', 'en')

assert(formatBytes(1024) === '1.00 KiB', 'bytes kib')
assert(formatCpu(12.345) === '12.3%', 'cpu')
assert(formatByteMetric({ available: false, bytes: null, reason: 'pending' }) === 'n/a (pending)', 'metric')
assert(
  formatByteMetric({ available: true, bytes: 2048, partial: true, unknownCount: 1 }) === '2.00 KiB (partial)',
  'aggregate',
)

memory.set('dv.locale', 'ru')
assert(formatByteMetric({ available: false, bytes: null, reason: 'pending' }) === 'н/д (pending)', 'metric ru')

console.log('format.test.ts ok')
