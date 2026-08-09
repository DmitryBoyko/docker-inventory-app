import { useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useSearchParams } from 'react-router-dom'
import { fetchNetworks } from '../api/client'
import { qk } from '../api/queryClient'
import { CliCommandsPanel } from '../components/CliCommandsPanel'
import { EntityListCell } from '../components/EntityListCell'
import { useT } from '../i18n'
import { formatAgeMs } from '../lib/format'
import { useDebouncedValue } from '../lib/useDebouncedValue'
import { useGrowingAgeMs } from '../lib/useGrowingAgeMs'
import { useLiveConnected } from '../realtime/useLiveState'

export function NetworksPage() {
  const t = useT()
  const wsConnected = useLiveConnected()
  const [params] = useSearchParams()
  const [q, setQ] = useState(params.get('q') ?? '')
  const [driver, setDriver] = useState(params.get('driver') ?? '')
  const [selected, setSelected] = useState('')
  const qDebounced = useDebouncedValue(q, 250)

  const query = useQuery({
    queryKey: qk.networks({ q: qDebounced || undefined, driver: driver || undefined }),
    queryFn: () => fetchNetworks({ q: qDebounced || undefined, driver: driver || undefined }),
    refetchInterval: wsConnected ? 20_000 : 8_000,
    placeholderData: (prev) => prev,
  })

  const drivers = useMemo(() => {
    const set = new Set((query.data?.data ?? []).map((n) => n.driver).filter(Boolean))
    return [...set].sort()
  }, [query.data])

  const rows = query.data?.data ?? []
  const dataAgeMs = useGrowingAgeMs(query.data?.snapshotAgeMs, query.dataUpdatedAt)

  return (
    <div className="page page-fill">
      <div className="page-head">
        <h1>{t('networks.title')}</h1>
        <p className="muted">
          {t('common.shown', { n: rows.length })} · {t('common.dataUpdated', { age: formatAgeMs(dataAgeMs) })}
        </p>
      </div>

      <div className="toolbar">
        <input
          className="input"
          placeholder={t('networks.search')}
          value={q}
          onChange={(e) => setQ(e.target.value)}
        />
        <select className="select" value={driver} onChange={(e) => setDriver(e.target.value)}>
          <option value="">{t('networks.allDrivers')}</option>
          {drivers.map((d) => (
            <option key={d} value={d}>
              {d}
            </option>
          ))}
        </select>
      </div>

      {query.isError ? <div className="banner danger">{(query.error as Error).message}</div> : null}

      <div className="table-wrap table-wrap-fill">
        <table className="table">
          <thead>
            <tr>
              <th>{t('common.name')}</th>
              <th>{t('common.driver')}</th>
              <th>{t('common.scope')}</th>
              <th>{t('common.flags')}</th>
              <th className="num">{t('common.containers')}</th>
              <th>{t('common.stacks')}</th>
              <th>{t('common.id')}</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((n) => (
              <tr key={n.id} className={selected === n.name ? 'row-selected' : undefined} onClick={() => setSelected(n.name)}>
                <td className="mono">
                  <button type="button" className="text-link linkish" onClick={() => setSelected(n.name)}>
                    {n.name}
                  </button>
                </td>
                <td>{n.driver}</td>
                <td>{n.scope || '—'}</td>
                <td>
                  {n.internal ? <span className="pill">{t('networks.internal')}</span> : null}{' '}
                  {n.ingress ? <span className="pill">{t('networks.ingress')}</span> : null}{' '}
                  {n.attachable ? <span className="pill">{t('networks.attachable')}</span> : null}
                  {!n.internal && !n.ingress && !n.attachable ? <span className="muted">—</span> : null}
                </td>
                <td className="cell-entities">
                  <EntityListCell
                    names={n.containers ?? []}
                    to={(c) => `/containers?q=${encodeURIComponent(c)}`}
                    empty="0"
                  />
                </td>
                <td className="cell-entities">
                  <EntityListCell
                    names={n.stacks ?? []}
                    to={(s) => `/containers?stack=${encodeURIComponent(s)}`}
                    empty="—"
                    preview={1}
                  />
                </td>
                <td className="mono">{n.idShort}</td>
              </tr>
            ))}
            {!query.isLoading && rows.length === 0 ? (
              <tr>
                <td colSpan={7} className="muted center">
                  {t('networks.empty')}
                </td>
              </tr>
            ) : null}
          </tbody>
        </table>
      </div>
      {selected ? <CliCommandsPanel kind="network" entityRef={selected} /> : null}
    </div>
  )
}
