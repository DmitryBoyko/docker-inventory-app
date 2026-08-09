import { useQuery } from '@tanstack/react-query'
import { useEffect, useMemo, useRef, useState } from 'react'
import { Link, useParams, useSearchParams } from 'react-router-dom'
import {
  fetchContainer,
  fetchContainerInspect,
  fetchContainerLogs,
} from '../api/client'
import { HistoryCharts } from '../components/HistoryCharts'
import { CliCommandsPanel } from '../components/CliCommandsPanel'
import { ProvenanceHint } from '../components/ProvenanceHint'
import { useT } from '../i18n'
import { formatByteMetric, formatBytes, formatCpu, formatUptime } from '../lib/format'
import { startLogStream } from '../lib/logStream'
import { getInspectRedactDefault } from '../lib/prefs'
import { mergeContainerStats } from '../realtime/store'
import { useLiveState } from '../realtime/useLiveState'

const tabs = ['overview', 'ports', 'networks', 'volumes', 'stats', 'logs', 'inspect', 'commands'] as const
type Tab = (typeof tabs)[number]
const MAX_LIVE_LOG_CHARS = 512 * 1024

export function ContainerDetailPage() {
  const t = useT()
  const { id = '' } = useParams()
  const [params, setParams] = useSearchParams()
  const tab = (tabs.includes(params.get('tab') as Tab) ? params.get('tab') : 'overview') as Tab
  const live = useLiveState()
  const [redact, setRedact] = useState(() => getInspectRedactDefault())
  const [tail, setTail] = useState(200)
  const [timestamps, setTimestamps] = useState(false)
  const [followLive, setFollowLive] = useState(true)
  const [liveText, setLiveText] = useState('')
  const [liveStatus, setLiveStatus] = useState('')
  const logRef = useRef<HTMLPreElement>(null)

  const containerQ = useQuery({
    queryKey: ['container', id],
    queryFn: () => fetchContainer(id),
    enabled: !!id,
    refetchInterval: live.connected ? 15000 : 3000,
  })

  const inspectQ = useQuery({
    queryKey: ['container-inspect', id, redact],
    queryFn: () => fetchContainerInspect(id, redact),
    enabled: !!id && tab === 'inspect',
  })

  const logsQ = useQuery({
    queryKey: ['container-logs', id, tail, timestamps],
    queryFn: () => fetchContainerLogs(id, { tail, timestamps }),
    enabled: !!id && tab === 'logs' && !followLive,
  })

  useEffect(() => {
    if (tab !== 'logs' || !followLive || !id) {
      setLiveStatus('')
      return
    }
    setLiveText('')
    setLiveStatus('connecting…')
    const handle = startLogStream(
      id,
      { tail, timestamps },
      (chunk) => {
        setLiveText((prev) => {
          const next = prev + chunk
          return next.length > MAX_LIVE_LOG_CHARS ? next.slice(-MAX_LIVE_LOG_CHARS) : next
        })
      },
      (st) => setLiveStatus(st),
    )
    return () => handle.stop()
  }, [tab, followLive, id, tail, timestamps])

  useEffect(() => {
    if (followLive && logRef.current) {
      logRef.current.scrollTop = logRef.current.scrollHeight
    }
  }, [liveText, followLive])

  const c = useMemo(() => {
    const raw = containerQ.data?.data
    if (!raw) return null
    return mergeContainerStats([raw])[0]
  }, [containerQ.data, live.statsById])

  const setTab = (t: Tab) => {
    const next = new URLSearchParams(params)
    next.set('tab', t)
    setParams(next, { replace: true })
  }

  if (containerQ.isError) {
    return (
      <div className="page">
        <div className="banner danger">{(containerQ.error as Error).message}</div>
        <Link className="text-link" to="/containers">
          ← Containers
        </Link>
      </div>
    )
  }

  return (
    <div className="page">
      <div className="page-head">
        <div>
          <Link className="text-link muted tiny" to="/containers">
            ← Containers
          </Link>
          <h1 className="mono">{c?.name ?? id}</h1>
          <p className="muted">
            {c ? (
              <>
                <span className={`pill state-${c.state}`}>{c.state}</span>{' '}
                <span className={`pill health-${c.health}`}>{c.health}</span>
                {' · '}
                {c.stack}
                {c.service ? ` / ${c.service}` : ''}
                {' · '}
                <span className="mono">{c.idShort}</span>
              </>
            ) : (
              'Loading…'
            )}
          </p>
        </div>
      </div>

      <div className="tabs">
        {tabs.map((tabId) => (
          <button
            key={tabId}
            type="button"
            className={tabId === tab ? 'tab active' : 'tab'}
            onClick={() => setTab(tabId)}
          >
            {t(`tab.${tabId}`)}
          </button>
        ))}
      </div>

      {tab === 'overview' && c ? (
        <section className="panel">
          <dl className="kv">
            <div>
              <dt>Image</dt>
              <dd className="mono">{c.image}</dd>
            </div>
            <div>
              <dt>Stack / service</dt>
              <dd>
                <Link className="text-link" to={`/containers?stack=${encodeURIComponent(c.stack)}`}>
                  {c.stack}
                </Link>
                {c.service ? ` / ${c.service}` : ''}
              </dd>
            </div>
            <div>
              <dt>
                Restarts <ProvenanceHint provenanceId="container.restartCount" displayedValue={String(c.restartCount)} />
              </dt>
              <dd>{c.restartCount}</dd>
            </div>
            <div>
              <dt>Uptime</dt>
              <dd>{formatUptime(c.uptimeSeconds)}</dd>
            </div>
            <div>
              <dt>
                Writable layer{' '}
                <ProvenanceHint provenanceId="container.writableLayer" displayedValue={formatByteMetric(c.writableLayer)} />
              </dt>
              <dd>{formatByteMetric(c.writableLayer)}</dd>
            </div>
            <div>
              <dt>Full ID</dt>
              <dd className="mono">{c.id}</dd>
            </div>
          </dl>
        </section>
      ) : null}

      {tab === 'ports' && c ? (
        <section className="panel">
          {c.ports?.length ? (
            <table className="table">
              <thead>
                <tr>
                  <th>Host</th>
                  <th>Container</th>
                  <th>Proto</th>
                  <th>Exposure</th>
                </tr>
              </thead>
              <tbody>
                {c.ports.map((p, i) => (
                  <tr key={i}>
                    <td className="mono">
                      {p.hostIP ? `${p.hostIP}:` : ''}
                      {p.hostPort ?? '—'}
                    </td>
                    <td className="num">{p.containerPort}</td>
                    <td>{p.protocol}</td>
                    <td>{p.exposure ?? '—'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          ) : (
            <p className="muted">No published ports.</p>
          )}
        </section>
      ) : null}

      {tab === 'networks' && c ? (
        <section className="panel">
          {c.endpoints?.length ? (
            <table className="table">
              <thead>
                <tr>
                  <th>Network</th>
                  <th>IP</th>
                  <th>Gateway</th>
                </tr>
              </thead>
              <tbody>
                {c.endpoints.map((e, i) => (
                  <tr key={i}>
                    <td>
                      <Link
                        className="text-link mono"
                        to={`/networks?q=${encodeURIComponent(e.networkName)}`}
                      >
                        {e.networkName}
                      </Link>
                    </td>
                    <td className="mono">
                      {e.ipAddress || '—'}{' '}
                      {e.ipAddress ? <ProvenanceHint provenanceId="container.ip" displayedValue={e.ipAddress} /> : null}
                    </td>
                    <td className="mono">{e.gateway || '—'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          ) : (
            <p className="muted">No network endpoints.</p>
          )}
        </section>
      ) : null}

      {tab === 'volumes' && c ? (
        <section className="panel">
          {c.mounts?.length ? (
            <table className="table">
              <thead>
                <tr>
                  <th>Type</th>
                  <th>Name / source</th>
                  <th>Destination</th>
                  <th>RW</th>
                </tr>
              </thead>
              <tbody>
                {c.mounts.map((m, i) => (
                  <tr key={i}>
                    <td>{m.type}</td>
                    <td className="mono">
                      {m.type === 'volume' && m.name ? (
                        <Link className="text-link" to={`/volumes?q=${encodeURIComponent(m.name)}`}>
                          {m.name}
                        </Link>
                      ) : (
                        m.name || m.source || '—'
                      )}
                    </td>
                    <td className="mono">{m.destination}</td>
                    <td>{m.rw ? 'rw' : 'ro'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          ) : (
            <p className="muted">No mounts.</p>
          )}
        </section>
      ) : null}

      {tab === 'stats' && c ? (
        <>
          <section className="panel">
            {c.stats ? (
              <dl className="kv">
                <div>
                  <dt>
                    CPU <ProvenanceHint provenanceId="container.cpuPercent" displayedValue={formatCpu(c.stats.cpuPercent)} />
                  </dt>
                  <dd>{formatCpu(c.stats.cpuPercent)}</dd>
                </div>
                <div>
                  <dt>
                    Memory{' '}
                    <ProvenanceHint
                      provenanceId="container.memoryBytes"
                      displayedValue={formatBytes(c.stats.memoryBytes)}
                    />
                  </dt>
                  <dd>
                    {formatBytes(c.stats.memoryBytes)} / {formatBytes(c.stats.memoryLimitBytes)} (
                    {c.stats.memoryPercent.toFixed(1)}%)
                  </dd>
                </div>
                <div>
                  <dt>
                    Net I/O <ProvenanceHint provenanceId="container.networkIO" />
                  </dt>
                  <dd>
                    rx {formatBytes(c.stats.networkRxBytes)} · tx {formatBytes(c.stats.networkTxBytes)}
                  </dd>
                </div>
                <div>
                  <dt>
                    Block I/O <ProvenanceHint provenanceId="container.blockIO" />
                  </dt>
                  <dd>
                    read {formatBytes(c.stats.blockReadBytes)} · write{' '}
                    {formatBytes(c.stats.blockWriteBytes)}
                  </dd>
                </div>
              </dl>
            ) : (
              <p className="muted">No stats sample (container may be stopped).</p>
            )}
          </section>
          <HistoryCharts
            scope="container"
            id={c.id}
            title="Container CPU / RAM (1h)"
            rangeHours={1}
          />
        </>
      ) : null}

      {tab === 'logs' ? (
        <section className="panel">
          <div className="banner info" style={{ border: 'none', margin: '0 0 0.75rem', padding: 0 }}>
            Logs are not stored by this app.
            {followLive ? ` · stream: ${liveStatus || '…'}` : ' · snapshot mode'}
          </div>
          <div className="toolbar">
            <label className="check-row">
              Tail
              <input
                className="input"
                style={{ minWidth: 80, flex: 'none', width: 90 }}
                type="number"
                min={1}
                max={5000}
                value={tail}
                onChange={(e) => setTail(Number(e.target.value) || 200)}
              />
            </label>
            <label className="check-row">
              <input
                type="checkbox"
                checked={timestamps}
                onChange={(e) => setTimestamps(e.target.checked)}
              />
              Timestamps
            </label>
            <label className="check-row">
              <input
                type="checkbox"
                checked={followLive}
                onChange={(e) => setFollowLive(e.target.checked)}
              />
              Live stream
            </label>
            {!followLive ? (
              <button type="button" className="btn" onClick={() => void logsQ.refetch()}>
                Refresh
              </button>
            ) : null}
          </div>
          {!followLive && logsQ.isError ? (
            <div className="banner danger">{(logsQ.error as Error).message}</div>
          ) : null}
          {!followLive && logsQ.data?.data.truncated ? (
            <div className="banner info">Output truncated to size limit.</div>
          ) : null}
          <pre className="log-view" ref={logRef}>
            {followLive
              ? liveText || (liveStatus === 'live' ? '(waiting for lines…)' : 'Connecting…')
              : logsQ.isLoading
                ? 'Loading…'
                : logsQ.data?.data.text || '(empty)'}
          </pre>
        </section>
      ) : null}

      {tab === 'inspect' ? (
        <section className="panel">
          <div className="toolbar">
            <label className="check-row">
              <input type="checkbox" checked={redact} onChange={(e) => setRedact(e.target.checked)} />
              Redact secrets (recommended)
            </label>
            <button type="button" className="btn" onClick={() => void inspectQ.refetch()}>
              Refresh
            </button>
          </div>
          {!redact ? (
            <div className="banner danger">
              Redaction is off — Env and secret labels may be visible.
            </div>
          ) : null}
          {inspectQ.isError ? (
            <div className="banner danger">{(inspectQ.error as Error).message}</div>
          ) : null}
          {inspectQ.data?.data.redactedFields?.length ? (
            <p className="muted tiny">
              Redacted: {inspectQ.data.data.redactedFields.slice(0, 12).join(', ')}
              {inspectQ.data.data.redactedFields.length > 12 ? '…' : ''}
            </p>
          ) : null}
          <pre className="log-view">
            {inspectQ.isLoading
              ? 'Loading…'
              : JSON.stringify(inspectQ.data?.data.inspect ?? null, null, 2)}
          </pre>
        </section>
      ) : null}

      {tab === 'commands' && c ? <CliCommandsPanel kind="container" entityRef={c.name || c.id} /> : null}
    </div>
  )
}
