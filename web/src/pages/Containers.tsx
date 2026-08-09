import { useEffect, useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link, useSearchParams } from 'react-router-dom'
import { fetchContainers } from '../api/client'
import { useT } from '../i18n'
import { formatAgeMs, formatByteMetric, formatBytes, formatCpu, formatUptime } from '../lib/format'
import { mergeContainerStats } from '../realtime/store'
import { useLiveState } from '../realtime/useLiveState'

export function ContainersPage() {
  const t = useT()
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
        <h1>{t('containers.title')}</h1>
        <p className="muted">
          {t('common.shown', { n: rows.length })} · {t('common.snapshot')}{' '}
          {formatAgeMs(query.data?.snapshotAgeMs)}
        </p>
      </div>

      <div className="toolbar">
        <input
          className="input"
          placeholder={t('containers.search')}
          value={q}
          onChange={(e) => setQ(e.target.value)}
        />
        <select className="select" value={state} onChange={(e) => setState(e.target.value)}>
          <option value="">{t('containers.allStates')}</option>
          <option value="running">{t('containers.state.running')}</option>
          <option value="exited">{t('containers.state.exited')}</option>
          <option value="created">{t('containers.state.created')}</option>
          <option value="paused">{t('containers.state.paused')}</option>
          <option value="restarting">{t('containers.state.restarting')}</option>
          <option value="dead">{t('containers.state.dead')}</option>
        </select>
        <select className="select" value={stack} onChange={(e) => setStack(e.target.value)}>
          <option value="">{t('common.allStacks')}</option>
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
              <th>{t('common.name')}</th>
              <th>{t('common.stack')}</th>
              <th>{t('common.service')}</th>
              <th>{t('common.state')}</th>
              <th>{t('common.health')}</th>
              <th className="num">{t('common.cpu')}</th>
              <th className="num">{t('common.memory')}</th>
              <th className="num">{t('containers.writable')}</th>
              <th className="num">{t('containers.restarts')}</th>
              <th className="num">{t('containers.uptime')}</th>
              <th>{t('common.image')}</th>
              <th>{t('common.id')}</th>
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
                  <span className={`pill state-${c.state}`}>
                    {t(`containers.state.${c.state}` as 'containers.state.running')}
                  </span>
                </td>
                <td>
                  <span className={`pill health-${c.health}`}>
                    {t(`containers.health.${c.health}` as 'containers.health.healthy')}
                  </span>
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
                  {t('containers.empty')}
                </td>
              </tr>
            ) : null}
          </tbody>
        </table>
      </div>
    </div>
  )
}
