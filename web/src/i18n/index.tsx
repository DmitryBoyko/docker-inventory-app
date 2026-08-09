import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from 'react'
import { getLocale, setLocale as persistLocale, type Locale } from '../lib/prefs'
import { en } from './locales/en'
import { ru } from './locales/ru'

const catalogs: Record<Locale, Record<string, string>> = { en, ru }

type I18nValue = {
  locale: Locale
  setLocale: (locale: Locale) => void
  t: (key: string, params?: Record<string, string | number>) => string
}

const I18nContext = createContext<I18nValue | null>(null)

function translate(locale: Locale, key: string, params?: Record<string, string | number>): string {
  const primary = catalogs[locale]?.[key]
  const fallback = catalogs.en[key]
  let text = primary ?? fallback ?? key
  if (params) {
    for (const [k, v] of Object.entries(params)) {
      text = text.replaceAll(`{${k}}`, String(v))
    }
  }
  return text
}

/** Prefer localized key; if missing in catalogs, use provided English/API fallback. */
export function tOr(t: (key: string, params?: Record<string, string | number>) => string, key: string, fallback: string, params?: Record<string, string | number>) {
  const v = t(key, params)
  return v === key ? fallback : v
}

export function I18nProvider({ children }: { children: ReactNode }) {
  const [locale, setLocaleState] = useState<Locale>(() => getLocale())

  const setLocale = useCallback((next: Locale) => {
    persistLocale(next)
    setLocaleState(next)
    document.documentElement.lang = next
  }, [])

  const t = useCallback(
    (key: string, params?: Record<string, string | number>) => translate(locale, key, params),
    [locale],
  )

  const value = useMemo(() => ({ locale, setLocale, t }), [locale, setLocale, t])
  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>
}

export function useI18n() {
  const ctx = useContext(I18nContext)
  if (!ctx) throw new Error('useI18n requires I18nProvider')
  return ctx
}

export function useT() {
  return useI18n().t
}
