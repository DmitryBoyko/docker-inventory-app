export type Theme = 'dark' | 'light'

const KEY_THEME = 'dv.theme'
const KEY_TOKEN = 'dv.authToken'
const KEY_REDACT = 'dv.inspectRedact'

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

export function authHeaders(): HeadersInit {
  const token = getAuthToken()
  if (!token) return { Accept: 'application/json' }
  return {
    Accept: 'application/json',
    Authorization: `Bearer ${token}`,
  }
}
