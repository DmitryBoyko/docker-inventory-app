import { useEffect, useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link, useSearchParams } from 'react-router-dom'
import { fetchContainers } from '../api/client'
import { formatAgeMs, formatByteMetric, formatBytes, formatCpu, formatUptime } from '../lib/format'
import { mergeContainerStats } from '../realtime/store'
import { useLiveState } from '../realtime/useLiveState'

export function ContainersPage() {
  const live = useLiveState()
  const [params, setParams] = useSearchParams()
  const [q, setQ] = useState(params.get('q') ?? '')
  const [state, setState] = useState(params.get('state') ?? '')
  const [stack, setStack] = useState(params.get('stack') ?? '')

  useEffect(() => {
    const next = new URLSearchParams()
    if (q) next.set('q', q)
    if (state) next.set('state', state)
    if (stack) next.set('stack', stack)
    setParams(next, { replace: true })
  }, [q, state, stack, setParams])

  const query = useQuery({
    queryKey: ['containers', { q, state, stack }],
    queryFn: () =>
      fetchContainers({
        q: q || undefined,
        state: state || undefined,
        stack: stack || undefined,
      }),
    refetchInterval: live.connected ? 20000 : 2000,
  })

  const stacks = useMemo(() => {
    const set = new Set((query.data?.data ?? []).map((c) => c.stack))
    return [...set].sort()
  }, [query.data])

  const rows = useMemo(
    () => mergeContainerStats(query.data?.data ?? []),
    [query.data, live.statsById],
  )

  return (
    <div className="page">
      <div className="page-head">
        <h1>Containers</h1>
        <p className="muted">
          {rows.length} shown · snapshot {formatAgeMs(query.data?.snapshotAgeMs)}
        </p>
      </div>

      <div className="toolbar">
        <input
          className="input"
          placeholder="Search name, image, id, stack…"
          value={q}
          onChange={(e) => setQ(e.target.value)}
        />
        <select className="select" value={state} onChange={(e) => setState(e.target.value)}>
          <option value="">All states</option>
          <option value="running">running</option>
          <option value="exited">exited</option>
          <option value="created">created</option>
          <option value="paused">paused</option>
          <option value="restarting">restarting</option>
          <option value="dead">dead</option>
        </select>
        <select className="select" value={stack} onChange={(e) => setStack(e.target.value)}>
          <option value="">All stacks</option>
          {stacks.map((s) => (
            <option key={s} value={s}>
              {s}
            </option>
          ))}
        </select>
      </div>

      {query.isError ? (
        <div className="banner danger">{(query.error as Error).message}</div>
      ) : null}

      <div className="table-wrap">
        <table className="table dense">
          <thead>
            <tr>
              <th>Name</th>
              <th>Stack</th>
              <th>Service</th>
              <th>State</th>
              <th>Health</th>
              <th className="num">CPU</th>
              <th className="num">Memory</th>
              <th className="num">Writable</th>
              <th className="num">Restarts</th>
              <th className="num">Uptime</th>
              <th>Image</th>
              <th>ID</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((c) => (
              <tr key={c.id}>
                <td className="mono">
                  <Link className="text-link" to={`/containers/${encodeURIComponent(c.idShort)}`}>
                    {c.name}
                  </Link>
                </td>
                <td>{c.stack}</td>
                <td>{c.service ?? '—'}</td>
                <td>
                  <span className={`pill state-${c.state}`}>{c.state}</span>
                </td>
                <td>
                  <span className={`pill health-${c.health}`}>{c.health}</span>
                </td>
                <td className="num">{formatCpu(c.stats?.cpuPercent)}</td>
                <td className="num">{formatBytes(c.stats?.memoryBytes)}</td>
                <td className="num">{formatByteMetric(c.writableLayer)}</td>
                <td className="num">{c.restartCount}</td>
                <td className="num">{formatUptime(c.uptimeSeconds)}</td>
                <td className="mono truncate" title={c.image}>
                  {c.image}
                </td>
                <td className="mono">{c.idShort}</td>
              </tr>
            ))}
            {!query.isLoading && rows.length === 0 ? (
              <tr>
                <td colSpan={12} className="muted center">
                  No containers match filters.
                </td>
              </tr>
            ) : null}
          </tbody>
        </table>
      </div>
    </div>
  )
}
