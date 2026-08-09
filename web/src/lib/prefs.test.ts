import {
  clearAuthToken,
  getAuthToken,
  getInspectRedactDefault,
  getSelectedHost,
  getTheme,
  setAuthToken,
  setInspectRedactDefault,
  setSelectedHost,
  setTheme,
} from './prefs'

function assert(cond: boolean, msg: string) {
  if (!cond) throw new Error(msg)
}

const memory = new Map<string, string>()
const storage = {
  getItem: (k: string) => (memory.has(k) ? memory.get(k)! : null),
  setItem: (k: string, v: string) => {
    memory.set(k, String(v))
  },
  removeItem: (k: string) => {
    memory.delete(k)
  },
  clear: () => memory.clear(),
  key: (i: number) => [...memory.keys()][i] ?? null,
  get length() {
    return memory.size
  },
}
Object.defineProperty(globalThis, 'localStorage', { value: storage, configurable: true })
Object.defineProperty(globalThis, 'document', {
  value: { documentElement: { dataset: {} as Record<string, string>, style: { colorScheme: '' } } },
  configurable: true,
})
Object.defineProperty(globalThis, 'window', {
  value: { dispatchEvent: () => true },
  configurable: true,
})

clearAuthToken()
assert(getAuthToken() === '', 'token empty')
setAuthToken(' abc ')
assert(getAuthToken() === 'abc', 'token trimmed')
clearAuthToken()
assert(getAuthToken() === '', 'token cleared')

setTheme('light')
assert(getTheme() === 'light', 'theme light')
setTheme('dark')
assert(getTheme() === 'dark', 'theme dark')

setInspectRedactDefault(false)
assert(getInspectRedactDefault() === false, 'redact off')
setInspectRedactDefault(true)
assert(getInspectRedactDefault() === true, 'redact on')

setSelectedHost(' lab ')
assert(getSelectedHost() === 'lab', 'host trimmed')
setSelectedHost('')
assert(getSelectedHost() === '', 'host cleared')

console.log('prefs.test.ts ok')
