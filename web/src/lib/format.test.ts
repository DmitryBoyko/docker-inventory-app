import { formatByteMetric, formatBytes, formatCpu } from './format'

function assert(cond: boolean, msg: string) {
  if (!cond) throw new Error(msg)
}

assert(formatBytes(1024) === '1.00 KiB', 'bytes kib')
assert(formatCpu(12.345) === '12.3%', 'cpu')
assert(formatByteMetric({ available: false, bytes: null, reason: 'pending' }) === 'n/a (pending)', 'metric')
assert(
  formatByteMetric({ available: true, bytes: 2048, partial: true, unknownCount: 1 }) === '2.00 KiB (partial)',
  'aggregate',
)

console.log('format.test.ts ok')
