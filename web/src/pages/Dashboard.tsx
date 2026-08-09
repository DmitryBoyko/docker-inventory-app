import { useQuery } from '@tanstack/react-query'
import { useMemo } from 'react'
import { Link } from 'react-router-dom'
import {
  fetchContainers,
  fetchStacks,
  fetchSystemInfo,
  fetchSystemResources,
} from '../api/client'
import { LiveCharts } from '../components/LiveCharts'
import { StatCard } from '../components/StatCard'
import { formatAgeMs, formatByteMetric, formatBytes, formatCpu } from '../lib/format'
import { mergeContainerStats } from '../realtime/store'
import { useLiveState } from '../realtime/useLiveState'

export function DashboardPage() {
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
        <h1>Dashboard</h1>
        <p className="muted">
          Snapshot age {formatAgeMs(containers.data?.snapshotAgeMs)}
          {live.connected ? ' · live stats via WS' : ' · polling'}
          {containers.data?.collectError ? (
            <span className="warn"> · collect: {containers.data.collectError}</span>
          ) : null}
        </p>
      </div>

      {resources.isError ? (
        <div className="banner danger">Failed to load resources: {(resources.error as Error).message}</div>
      ) : null}

      <div className="stat-grid">
        <StatCard
          label="CPU (sum)"
          value={formatCpu(liveCpu ?? r?.cpuPercent)}
          hint={liveCpu != null ? 'live' : 'running containers'}
        />
        <StatCard
          label="Memory"
          value={formatBytes(liveMem ?? r?.memoryBytes)}
          hint={liveMem != null ? 'live' : 'running containers'}
        />
        <StatCard label="Writable layers" value={formatByteMetric(r?.writableLayer)} />
        <StatCard label="Volume data" value={formatByteMetric(r?.volumeData)} />
        <StatCard
          label="Containers"
          value={r ? `${r.runningCount}/${r.containerCount}` : '—'}
          hint="running / total"
        />
        <StatCard label="Stacks" value={String(stacks.data?.data?.length ?? '—')} />
      </div>

      <div className="stack-gap">
        <LiveCharts />
      </div>

      <div className="split">
        <section className="panel">
          <h2>Engine</h2>
          {info.isError ? (
            <p className="muted">Info not available yet (wait for system collect).</p>
          ) : engine ? (
            <dl className="kv">
              <div>
                <dt>Version</dt>
                <dd>
                  {engine.serverVersion}
                  {engine.apiVersion ? ` (API ${engine.apiVersion})` : ''}
                </dd>
              </div>
              <div>
                <dt>Host</dt>
                <dd>
                  {engine.name || '—'} · {engine.os} / {engine.architecture}
                </dd>
              </div>
              <div>
                <dt>CPUs / RAM</dt>
                <dd>
                  {engine.cpus} · {formatBytes(engine.memoryBytes)}
                </dd>
              </div>
              <div>
                <dt>Images</dt>
                <dd>{engine.images}</dd>
              </div>
              <div>
                <dt>Driver</dt>
                <dd>{engine.driver || '—'}</dd>
              </div>
            </dl>
          ) : (
            <p className="muted">Loading…</p>
          )}
        </section>

        <section className="panel">
          <div className="panel-head">
            <h2>Top memory</h2>
            <Link to="/containers" className="text-link">
              All containers
            </Link>
          </div>
          {topMem.length === 0 ? (
            <p className="muted">No stats yet.</p>
          ) : (
            <table className="table">
              <thead>
                <tr>
                  <th>Name</th>
                  <th>Stack</th>
                  <th className="num">CPU</th>
                  <th className="num">Memory</th>
                </tr>
              </thead>
              <tbody>
                {topMem.map((c) => (
                  <tr key={c.id}>
                    <td>
                      <span className="mono">{c.name}</span>
                    </td>
                    <td>{c.stack}</td>
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
