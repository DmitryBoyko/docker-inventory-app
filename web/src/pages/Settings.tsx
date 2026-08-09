import { useQuery } from '@tanstack/react-query'
import { useEffect, useState } from 'react'
import { ApiError, fetchSystemSettings } from '../api/client'
import { ExportButtons } from '../components/ExportButtons'
import {
  clearAuthToken,
  getAuthToken,
  getInspectRedactDefault,
  getTheme,
  setAuthToken,
  setInspectRedactDefault,
  setTheme,
  type Theme,
} from '../lib/prefs'
import { useLiveState } from '../realtime/useLiveState'

export function SettingsPage() {
  const live = useLiveState()
  const [theme, setThemeState] = useState<Theme>(() => getTheme())
  const [tokenDraft, setTokenDraft] = useState(() => getAuthToken())
  const [redact, setRedact] = useState(() => getInspectRedactDefault())
  const [savedMsg, setSavedMsg] = useState('')

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
    setSavedMsg('Token saved. Reloading…')
    window.setTimeout(() => window.location.reload(), 400)
  }

  const clearToken = () => {
    clearAuthToken()
    setTokenDraft('')
    setSavedMsg('Token cleared. Reloading…')
    window.setTimeout(() => window.location.reload(), 400)
  }

  const data = settingsQ.data?.data

  return (
    <div className="page">
      <div className="page-head">
        <div>
          <h1>Settings</h1>
          <p className="muted">Server config (read-only) and local UI preferences</p>
        </div>
      </div>

      {unauthorized && (
        <div className="banner warn">
          API requires a Bearer token. Enter it below (ADR-013), then Save.
        </div>
      )}
      {settingsQ.isError && !unauthorized && (
        <div className="banner danger">{(settingsQ.error as Error).message}</div>
      )}
      {savedMsg && <div className="banner ok">{savedMsg}</div>}

      <section className="panel settings-panel">
        <h2>Server</h2>
        {settingsQ.isLoading && !data ? (
          <p className="muted">Loading…</p>
        ) : data ? (
          <dl className="kv">
            <div>
              <dt>Listen</dt>
              <dd className="mono">{data.listen}</dd>
            </div>
            <div>
              <dt>Loopback</dt>
              <dd>{data.listenLoopback ? 'yes' : 'no'}</dd>
            </div>
            <div>
              <dt>Auth</dt>
              <dd>{data.authEnabled ? 'enabled' : 'disabled'}</dd>
            </div>
            <div>
              <dt>Docker timeout</dt>
              <dd className="mono">{data.dockerTimeout}</dd>
            </div>
            <div>
              <dt>Inventory interval</dt>
              <dd className="mono">{data.intervals.inventory ?? '—'}</dd>
            </div>
            <div>
              <dt>Stats interval</dt>
              <dd className="mono">{data.intervals.stats ?? '—'}</dd>
            </div>
            <div>
              <dt>System interval</dt>
              <dd className="mono">{data.intervals.system ?? '—'}</dd>
            </div>
            <div>
              <dt>Version</dt>
              <dd className="mono">
                {data.version} ({data.commit})
              </dd>
            </div>
            <div>
              <dt>UI embed</dt>
              <dd>{data.uiEmbedded ? 'yes' : 'no'}</dd>
            </div>
            <div>
              <dt>Inspect redact default</dt>
              <dd>{data.defaults.inspectRedact ? 'on' : 'off'}</dd>
            </div>
          </dl>
        ) : (
          <p className="muted">Server settings unavailable until authenticated.</p>
        )}
        <p className="muted tiny">
          Intervals are process flags (`--inventory-interval`, etc.) — restart to change.
        </p>
      </section>

      <section className="panel settings-panel">
        <h2>Client</h2>
        <label className="field">
          <span>Theme</span>
          <select
            className="select"
            value={theme}
            onChange={(e) => {
              const t = e.target.value as Theme
              setThemeState(t)
              setTheme(t)
            }}
          >
            <option value="dark">Dark</option>
            <option value="light">Light</option>
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
          <span>Default redact on container inspect</span>
        </label>

        <label className="field">
          <span>API token (localStorage)</span>
          <input
            type="password"
            autoComplete="off"
            className="input mono"
            value={tokenDraft}
            placeholder={data?.authEnabled ? 'required' : 'optional'}
            onChange={(e) => setTokenDraft(e.target.value)}
          />
        </label>
        <div className="toolbar">
          <button type="button" className="btn" onClick={saveToken}>
            Save token
          </button>
          <button type="button" className="btn ghost" onClick={clearToken}>
            Clear token
          </button>
        </div>
        <p className="muted tiny">
          Used for REST `Authorization: Bearer` and WebSocket `?access_token=`. Never sent to
          third parties.
        </p>
      </section>

      <section className="panel settings-panel">
        <h2>Export</h2>
        <ExportButtons />
      </section>

      <section className="panel settings-panel">
        <h2>Realtime</h2>
        <dl className="kv">
          <div>
            <dt>WebSocket</dt>
            <dd>{live.connected ? 'connected' : 'disconnected'}</dd>
          </div>
          <div>
            <dt>Docker</dt>
            <dd>
              {live.docker == null ? '—' : live.docker.connected ? 'connected' : 'disconnected'}
            </dd>
          </div>
          <div>
            <dt>Events stream</dt>
            <dd>
              {live.events == null ? '—' : live.events.connected ? 'up' : 'down'}
            </dd>
          </div>
        </dl>
        <p className="muted tiny">Reconnect is automatic with exponential backoff.</p>
      </section>
    </div>
  )
}
