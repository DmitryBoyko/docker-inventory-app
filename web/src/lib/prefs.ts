export type Theme = 'dark' | 'light'
export type Locale = 'en' | 'ru'

const KEY_THEME = 'dv.theme'
const KEY_TOKEN = 'dv.authToken'
const KEY_REDACT = 'dv.inspectRedact'
const KEY_HOST = 'dv.dockerHost'
const KEY_LOCALE = 'dv.locale'
const KEY_SHELL = 'dv.cliShell'

/** Dispatched when the selected Docker host changes (ADR-014). */
export const HOST_CHANGE_EVENT = 'dv:host-change'

export type CliShell = 'bash' | 'powershell' | 'cmd'

export function detectBrowserLocale(): Locale {
  const lang = (navigator.language || 'en').toLowerCase()
  if (lang.startsWith('ru')) return 'ru'
  return 'en'
}

export function getLocale(): Locale {
  const v = localStorage.getItem(KEY_LOCALE)
  if (v === 'ru' || v === 'en') return v
  return detectBrowserLocale()
}

export function setLocale(locale: Locale) {
  localStorage.setItem(KEY_LOCALE, locale)
}

export function getCliShell(): CliShell {
  const v = localStorage.getItem(KEY_SHELL)
  if (v === 'powershell' || v === 'cmd' || v === 'bash') return v
  return 'bash'
}

export function setCliShell(shell: CliShell) {
  localStorage.setItem(KEY_SHELL, shell)
}

export function getTheme(): Theme {
  const v = localStorage.getItem(KEY_THEME)
  return v === 'light' ? 'light' : 'dark'
}

export function setTheme(theme: Theme) {
  localStorage.setItem(KEY_THEME, theme)
  applyTheme(theme)
}

export function applyTheme(theme: Theme = getTheme()) {
  document.documentElement.dataset.theme = theme
  document.documentElement.style.colorScheme = theme
}

export function getAuthToken(): string {
  return localStorage.getItem(KEY_TOKEN) ?? ''
}

export function setAuthToken(token: string) {
  const t = token.trim()
  if (t) localStorage.setItem(KEY_TOKEN, t)
  else localStorage.removeItem(KEY_TOKEN)
}

export function clearAuthToken() {
  localStorage.removeItem(KEY_TOKEN)
}

export function getInspectRedactDefault(): boolean {
  const v = localStorage.getItem(KEY_REDACT)
  if (v === null) return true
  return v !== '0' && v !== 'false'
}

export function setInspectRedactDefault(on: boolean) {
  localStorage.setItem(KEY_REDACT, on ? '1' : '0')
}

/** Selected Docker host name; empty ⇒ server default. */
export function getSelectedHost(): string {
  return localStorage.getItem(KEY_HOST) ?? ''
}

export function setSelectedHost(name: string) {
  const n = name.trim()
  if (n) localStorage.setItem(KEY_HOST, n)
  else localStorage.removeItem(KEY_HOST)
  window.dispatchEvent(new CustomEvent(HOST_CHANGE_EVENT, { detail: n }))
}

export function authHeaders(): HeadersInit {
  const token = getAuthToken()
  if (!token) return { Accept: 'application/json' }
  return {
    Accept: 'application/json',
    Authorization: `Bearer ${token}`,
  }
}
