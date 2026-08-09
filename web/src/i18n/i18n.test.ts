import { en } from './locales/en'
import { ru } from './locales/ru'

function assert(cond: boolean, msg: string) {
  if (!cond) throw new Error(msg)
}

const required = [
  'nav.dashboard',
  'nav.diagnostics',
  'cli.copy',
  'cmd.container.inspect.title',
  'diag.title',
  'snap.title',
  'dash.title',
  'containers.title',
  'settings.title',
  'status.connected',
]

for (const key of required) {
  assert(!!en[key], `en missing ${key}`)
  assert(!!ru[key], `ru missing ${key}`)
}

const missingInRu = Object.keys(en).filter((k) => !(k in ru))
const missingInEn = Object.keys(ru).filter((k) => !(k in en))
assert(missingInRu.length === 0, `ru missing keys: ${missingInRu.join(', ')}`)
assert(missingInEn.length === 0, `en missing keys: ${missingInEn.join(', ')}`)
assert(Object.keys(en).length >= 100, 'en catalog too small')

console.log(`i18n.test.ts ok (${Object.keys(en).length} keys)`)
