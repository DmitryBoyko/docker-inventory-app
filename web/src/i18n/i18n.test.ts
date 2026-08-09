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
]

for (const key of required) {
  assert(!!en[key], `en missing ${key}`)
  assert(!!ru[key], `ru missing ${key}`)
}

assert(Object.keys(en).length >= 20, 'en catalog too small')
assert(Object.keys(ru).length >= 20, 'ru catalog too small')

console.log('i18n.test.ts ok')
