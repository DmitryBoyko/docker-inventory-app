import { useQuery } from '@tanstack/react-query'
import { useEffect, useState } from 'react'
import { ApiError, fetchSystemSettings } from '../api/client'
import { CliCommandsPanel } from '../components/CliCommandsPanel'
import { ExportButtons } from '../components/ExportButtons'
import { useI18n, useT } from '../i18n'
import {
  clearAuthToken,
  getAuthToken,
  getInspectRedactDefault,
  getLocale,
  getTheme,
  setAuthToken,
  setInspectRedactDefault,
  setTheme,
  type Locale,
  type Theme,
} from '../lib/prefs'
import { useLiveState } from '../realtime/useLiveState'

export function SettingsPage() {
  const live = useLiveState()
  const { locale, setLocale } = useI18n()
  const t = useT()
  const [theme, setThemeState] = useState<Theme>(() => getTheme())
  const [tokenDraft, setTokenDraft] = useState(() => getAuthToken())
  const [redact, setRedact] = useState(() => getInspectRedactDefault())
  const [savedMsg, setSavedMsg] = useState('')
  const [lang, setLang] = useState<Locale>(() => getLocale())

  const settingsQ = useQuery({
    queryKey: ['system-settings'],
    queryFn: fetchSystemSettings,
    retry: false,
    refetchInterval: 30000,
  })

  useEffect(() => {
    setTheme(theme)
  }, [theme])

  const unauthorized =
    settingsQ.error instanceof ApiError && settingsQ.error.status === 401

  const saveToken = () => {
    setAuthToken(tokenDraft)
    setSavedMsg(t('settings.tokenSaved'))
    window.setTimeout(() => window.location.reload(), 400)
  }

  const clearToken = () => {
    clearAuthToken()
    setTokenDraft('')
    setSavedMsg(t('settings.tokenCleared'))
    window.setTimeout(() => window.location.reload(), 400)
  }

  const data = settingsQ.data?.data

  return (
    <div className="page">
      <div className="page-head">
        <div>
          <h1>{t('settings.title')}</h1>
          <p className="muted">{t('settings.subtitle')}</p>
        </div>
      </div>

      {unauthorized && (
        <div className="banner warn">
          {t('settings.authWarn')}
        </div>
      )}
      {settingsQ.isError && !unauthorized && (
        <div className="banner danger">{(settingsQ.error as Error).message}</div>
      )}
      {savedMsg && <div className="banner ok">{savedMsg}</div>}

      <section className="panel settings-panel">
        <h2>{t('settings.server')}</h2>
        {settingsQ.isLoading && !data ? (
          <p className="muted">{t('common.loading')}</p>
        ) : data ? (
          <dl className="kv">
            <div>
              <dt>{t('settings.listen')}</dt>
              <dd className="mono">{data.listen}</dd>
            </div>
            <div>
              <dt>{t('settings.loopback')}</dt>
              <dd>{data.listenLoopback ? t('common.yes') : t('settings.metricsOff')}</dd>
            </div>
            <div>
              <dt>{t('settings.auth')}</dt>
              <dd>{data.authEnabled ? t('settings.metricsOn') : t('settings.metricsOff')}</dd>
            </div>
            <div>
              <dt>{t('settings.dockerTimeout')}</dt>
              <dd className="mono">{data.dockerTimeout}</dd>
            </div>
            <div>
              <dt>{t('settings.inventoryInterval')}</dt>
              <dd className="mono">{data.intervals.inventory ?? '—'}</dd>
            </div>
            <div>
              <dt>{t('settings.statsInterval')}</dt>
              <dd className="mono">{data.intervals.stats ?? '—'}</dd>
            </div>
            <div>
              <dt>{t('settings.systemInterval')}</dt>
              <dd className="mono">{data.intervals.system ?? '—'}</dd>
            </div>
            <div>
              <dt>{t('common.version')}</dt>
              <dd className="mono">
                {data.version} ({data.commit})
              </dd>
            </div>
            <div>
              <dt>{t('settings.uiEmbed')}</dt>
              <dd>{data.uiEmbedded ? t('common.yes') : t('settings.metricsOff')}</dd>
            </div>
            <div>
              <dt>{t('settings.defaultHost')}</dt>
              <dd className="mono">{data.defaultHost ?? t('host.default')}</dd>
            </div>
            <div>
              <dt>{t('settings.metricsDb')}</dt>
              <dd className="mono">
                {data.metrics?.enabled
                  ? `${data.metrics.dbPath ?? '—'} · ${data.metrics.interval ?? '?'} · ${data.metrics.retention ?? '?'}`
                  : t('settings.metricsOff')}
              </dd>
            </div>
            <div>
              <dt>{t('settings.inspectRedactDefault')}</dt>
              <dd>{data.defaults.inspectRedact ? t('settings.metricsOn') : t('settings.metricsOff')}</dd>
            </div>
          </dl>
        ) : (
          <p className="muted">{t('settings.authNeeded')}</p>
        )}
        {data?.hosts && data.hosts.length > 0 && (
          <>
            <h3 className="settings-sub">{t('settings.dockerHosts')}</h3>
            <ul className="host-list">
              {data.hosts.map((h) => (
                <li key={h.name}>
                  <span className="mono">{h.name}</span>
                  {h.isDefault ? ` · ${t('host.default')}` : ''}
                  {' · '}
                  <span className="muted mono">{h.endpoint || h.source}</span>
                  {h.connected ? ` · ${t('common.connected')}` : ` · ${t('common.disconnected')}`}
                </li>
              ))}
            </ul>
            <p className="muted tiny">
              {t('settings.hostsHint')}
            </p>
          </>
        )}
        <p className="muted tiny">
          {t('settings.intervalsHint')}
        </p>
      </section>

      <section className="panel settings-panel">
        <h2>{t('settings.client')}</h2>
        <label className="field">
          <span>{t('settings.language')}</span>
          <select
            className="select"
            value={lang}
            onChange={(e) => {
              const next = e.target.value as Locale
              setLang(next)
              setLocale(next)
            }}
          >
            <option value="en">{t('settings.lang.en')}</option>
            <option value="ru">{t('settings.lang.ru')}</option>
          </select>
        </label>
        <p className="muted tiny">{t('settings.localeCurrent', { locale })}</p>

        <label className="field">
          <span>{t('settings.theme')}</span>
          <select
            className="select"
            value={theme}
            onChange={(e) => {
              const next = e.target.value as Theme
              setThemeState(next)
              setTheme(next)
            }}
          >
            <option value="dark">{t('settings.theme.dark')}</option>
            <option value="light">{t('settings.theme.light')}</option>
          </select>
        </label>

        <label className="check-row">
          <input
            type="checkbox"
            checked={redact}
            onChange={(e) => {
              setRedact(e.target.checked)
              setInspectRedactDefault(e.target.checked)
            }}
          />
          <span>{t('settings.redactDefault')}</span>
        </label>

        <label className="field">
          <span>{t('settings.apiToken')}</span>
          <input
            type="password"
            autoComplete="off"
            className="input mono"
            value={tokenDraft}
            placeholder={data?.authEnabled ? t('common.required') : t('common.optional')}
            onChange={(e) => setTokenDraft(e.target.value)}
          />
        </label>
        <div className="toolbar">
          <button type="button" className="btn" onClick={saveToken}>
            {t('settings.saveToken')}
          </button>
          <button type="button" className="btn ghost" onClick={clearToken}>
            {t('settings.clearToken')}
          </button>
        </div>
        <p className="muted tiny">
          {t('settings.tokenHint')}
        </p>
      </section>

      <section className="panel settings-panel">
        <h2>{t('settings.export')}</h2>
        <ExportButtons />
      </section>

      <section className="panel settings-panel">
        <h2>{t('settings.realtime')}</h2>
        <dl className="kv">
          <div>
            <dt>{t('settings.websocket')}</dt>
            <dd>{live.connected ? t('common.connected') : t('common.disconnected')}</dd>
          </div>
          <div>
            <dt>{t('settings.docker')}</dt>
            <dd>
              {live.docker?.connected
                ? `${live.docker.host} (${live.docker.source})`
                : live.docker?.error || t('common.unknown')}
            </dd>
          </div>
          <div>
            <dt>{t('settings.events')}</dt>
            <dd>{live.events?.connected ? t('common.connected') : live.events?.error || t('common.disconnected')}</dd>
          </div>
        </dl>
      </section>

      <CliCommandsPanel kind="system" entityRef="" />
    </div>
  )
}
