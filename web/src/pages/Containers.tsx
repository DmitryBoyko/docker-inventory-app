import { useEffect, useMemo, useRef, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useSearchParams } from 'react-router-dom'
import { fetchContainers } from '../api/client'
import { qk } from '../api/queryClient'
import { ContainerTableRow } from '../components/ContainerTableRow'
import { useT } from '../i18n'
import { formatAgeMs } from '../lib/format'
import { useDebouncedValue } from '../lib/useDebouncedValue'
import { useGrowingAgeMs } from '../lib/useGrowingAgeMs'
import { mergeContainerStats } from '../realtime/store'
import { useLiveConnected, useThrottledStatsById } from '../realtime/useLiveState'

export function ContainersPage() {
  const t = useT()
  const wsConnected = useLiveConnected()
  const statsById = useThrottledStatsById(2000)
  const [params, setParams] = useSearchParams()
  const [q, setQ] = useState(params.get('q') ?? '')
  const [state, setState] = useState(params.get('state') ?? '')
  const [stack, setStack] = useState(params.get('stack') ?? '')
  const qDebounced = useDebouncedValue(q, 250)

  useEffect(() => {
    const next = new URLSearchParams()
    if (q) next.set('q', q)
    if (state) next.set('state', state)
    if (stack) next.set('stack', stack)
    setParams(next, { replace: true })
  }, [q, state, stack, setParams])

  const filters = useMemo(
    () => ({
      q: qDebounced || undefined,
      state: state || undefined,
      stack: stack || undefined,
    }),
    [qDebounced, state, stack],
  )

  const query = useQuery({
    queryKey: qk.containers(filters),
    queryFn: () => fetchContainers(filters),
    refetchInterval: wsConnected ? 20_000 : 8_000,
    placeholderData: (prev) => prev,
  })

  const stacks = useMemo(() => {
    const set = new Set((query.data?.data ?? []).map((c) => c.stack))
    return [...set].sort()
  }, [query.data])

  const prevRows = useRef<import('../api/types').Container[] | undefined>(undefined)
  const rows = useMemo(() => {
    void statsById
    const next = mergeContainerStats(query.data?.data ?? [], prevRows.current)
    prevRows.current = next
    return next
  }, [query.data, statsById])

  const dataAgeMs = useGrowingAgeMs(query.data?.snapshotAgeMs, query.dataUpdatedAt)

  return (
    <div className="page page-fill">
      <div className="page-head">
        <h1>{t('containers.title')}</h1>
        <p className="muted">
          {t('common.shown', { n: rows.length })} ·{' '}
          {t('common.dataUpdated', { age: formatAgeMs(dataAgeMs) })}
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

      <div className="table-wrap table-wrap-fill">
        <table className="table dense">
          <thead>
            <tr>
              <th>{t('common.name')}</th>
              <th>{t('common.stack')}</th>
              <th>{t('common.service')}</th>
              <th>{t('common.state')}</th>
              <th>{t('common.health')}</th>
              <th>{t('exposure.column')}</th>
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
              <ContainerTableRow key={c.id} container={c} t={t} />
            ))}
            {!query.isLoading && rows.length === 0 ? (
              <tr>
                <td colSpan={13} className="muted center">
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
