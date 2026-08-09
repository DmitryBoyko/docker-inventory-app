import { useQuery } from '@tanstack/react-query'
import { useMemo } from 'react'
import { Link } from 'react-router-dom'
import {
  fetchContainers,
  fetchStacks,
  fetchSystemInfo,
  fetchSystemResources,
} from '../api/client'
import { ExportButtons } from '../components/ExportButtons'
import { ExposureBadge } from '../components/ExposureBadge'
import { ExposureMap } from '../components/ExposureMap'
import { LiveCharts } from '../components/LiveCharts'
import { StatCard } from '../components/StatCard'
import { useT } from '../i18n'
import { collectExposureRoutes, countByScope } from '../lib/exposure'
import { formatAgeMs, formatByteMetric, formatBytes, formatCpu } from '../lib/format'
import { mergeContainerStats } from '../realtime/store'
import { useLiveState } from '../realtime/useLiveState'

export function DashboardPage() {
  const t = useT()
  const live = useLiveState()
  const poll = live.connected ? 15000 : 2000

  const resources = useQuery({
    queryKey: ['system', 'resources'],
    queryFn: fetchSystemResources,
    refetchInterval: poll,
  })
  const info = useQuery({
    queryKey: ['system', 'info'],
    queryFn: fetchSystemInfo,
    refetchInterval: 30000,
    retry: false,
  })
  const containers = useQuery({
    queryKey: ['containers'],
    queryFn: () => fetchContainers(),
    refetchInterval: poll,
  })
  const stacks = useQuery({
    queryKey: ['stacks'],
    queryFn: fetchStacks,
    refetchInterval: poll,
  })

  const list = useMemo(
    () => mergeContainerStats(containers.data?.data ?? []),
    [containers.data, live.statsById],
  )

  const exposureRoutes = useMemo(() => collectExposureRoutes(list), [list])
  const exposureCounts = useMemo(() => countByScope(exposureRoutes), [exposureRoutes])

  const liveCpu = live.history[live.history.length - 1]?.cpu
  const liveMem = live.history[live.history.length - 1]?.mem
  const r = resources.data?.data
  const engine = info.data?.data
  const topMem = [...list]
    .filter((c) => c.stats)
    .sort((a, b) => (b.stats?.memoryBytes ?? 0) - (a.stats?.memoryBytes ?? 0))
    .slice(0, 5)

  return (
    <div className="page">
      <div className="page-head">
        <div>
          <h1>{t('dash.title')}</h1>
          <p className="muted">
            {t('common.snapshot')} {formatAgeMs(containers.data?.snapshotAgeMs)}
            {live.connected ? ` · ${t('dash.liveWs')}` : ` · ${t('dash.polling')}`}
            {containers.data?.collectError ? (
              <span className="warn">
                {' '}
                · {t('dash.collect')}: {containers.data.collectError}
              </span>
            ) : null}
          </p>
        </div>
        <ExportButtons compact />
      </div>

      {resources.isError ? (
        <div className="banner danger">
          {t('dash.failedResources', { message: (resources.error as Error).message })}
        </div>
      ) : null}

      <div className="stat-grid">
        <StatCard
          label={t('dash.cpuSum')}
          value={formatCpu(liveCpu ?? r?.cpuPercent)}
          hint={liveCpu != null ? t('dash.hintLive') : t('dash.hintRunning')}
        />
        <StatCard
          label={t('dash.memory')}
          value={formatBytes(liveMem ?? r?.memoryBytes)}
          hint={liveMem != null ? t('dash.hintLive') : t('dash.hintRunning')}
        />
        <StatCard label={t('dash.writable')} value={formatByteMetric(r?.writableLayer)} />
        <StatCard label={t('dash.volumeData')} value={formatByteMetric(r?.volumeData)} />
        <StatCard
          label={t('dash.containers')}
          value={r ? `${r.runningCount}/${r.containerCount}` : '—'}
          hint={t('dash.hintRunningTotal')}
        />
        <StatCard label={t('dash.stacks')} value={String(stacks.data?.data?.length ?? '—')} />
        <StatCard
          label={t('exposure.statExternal')}
          value={String(exposureCounts.external)}
          hint={t('exposure.statExternalHint')}
        />
        <StatCard
          label={t('exposure.statLocal')}
          value={String(exposureCounts.localhost + exposureCounts.lan)}
          hint={t('exposure.statLocalHint')}
        />
      </div>

      <div className="stack-gap">
        <ExposureMap routes={exposureRoutes} t={t} />
      </div>

      <div className="stack-gap">
        <LiveCharts />
      </div>

      <div className="split">
        <section className="panel">
          <h2>{t('dash.engine')}</h2>
          {info.isError ? (
            <p className="muted">{t('dash.engineUnavailable')}</p>
          ) : engine ? (
            <dl className="kv">
              <div>
                <dt>{t('common.version')}</dt>
                <dd>
                  {engine.serverVersion}
                  {engine.apiVersion ? ` (API ${engine.apiVersion})` : ''}
                </dd>
              </div>
              <div>
                <dt>{t('dash.host')}</dt>
                <dd>
                  {engine.name || '—'} · {engine.os} / {engine.architecture}
                </dd>
              </div>
              <div>
                <dt>{t('dash.cpusRam')}</dt>
                <dd>
                  {engine.cpus} · {formatBytes(engine.memoryBytes)}
                </dd>
              </div>
              <div>
                <dt>{t('dash.images')}</dt>
                <dd>{engine.images}</dd>
              </div>
              <div>
                <dt>{t('dash.driver')}</dt>
                <dd>{engine.driver || '—'}</dd>
              </div>
            </dl>
          ) : (
            <p className="muted">{t('common.loading')}</p>
          )}
        </section>

        <section className="panel">
          <div className="panel-head">
            <h2>{t('dash.topMemory')}</h2>
            <Link to="/containers" className="text-link">
              {t('dash.allContainers')}
            </Link>
          </div>
          {topMem.length === 0 ? (
            <p className="muted">{t('dash.noStats')}</p>
          ) : (
            <table className="table">
              <thead>
                <tr>
                  <th>{t('common.name')}</th>
                  <th>{t('common.stack')}</th>
                  <th>{t('exposure.column')}</th>
                  <th className="num">{t('common.cpu')}</th>
                  <th className="num">{t('common.memory')}</th>
                </tr>
              </thead>
              <tbody>
                {topMem.map((c) => (
                  <tr key={c.id}>
                    <td>
                      <Link
                        className="text-link mono"
                        to={`/containers/${encodeURIComponent(c.idShort)}`}
                      >
                        {c.name}
                      </Link>
                    </td>
                    <td>{c.stack}</td>
                    <td>
                      <ExposureBadge container={c} t={t} compact />
                    </td>
                    <td className="num">{formatCpu(c.stats?.cpuPercent)}</td>
                    <td className="num">{formatBytes(c.stats?.memoryBytes)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </section>
      </div>
    </div>
  )
}
