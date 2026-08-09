import { useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link, useSearchParams } from 'react-router-dom'
import { fetchVolumes } from '../api/client'
import { qk } from '../api/queryClient'
import { CliCommandsPanel } from '../components/CliCommandsPanel'
import { ProvenanceHint } from '../components/ProvenanceHint'
import { useT } from '../i18n'
import { formatAgeMs, formatByteMetric } from '../lib/format'
import { useDebouncedValue } from '../lib/useDebouncedValue'
import { useLiveConnected } from '../realtime/useLiveState'

export function VolumesPage() {
  const t = useT()
  const wsConnected = useLiveConnected()
  const [params] = useSearchParams()
  const [q, setQ] = useState(params.get('q') ?? '')
  const [stack, setStack] = useState(params.get('stack') ?? '')
  const [selected, setSelected] = useState('')
  const qDebounced = useDebouncedValue(q, 250)

  const query = useQuery({
    queryKey: qk.volumes({ q: qDebounced || undefined, stack: stack || undefined }),
    queryFn: () => fetchVolumes({ q: qDebounced || undefined, stack: stack || undefined }),
    refetchInterval: wsConnected ? 20_000 : 8_000,
    placeholderData: (prev) => prev,
  })

  const stacks = useMemo(() => {
    const set = new Set<string>()
    for (const v of query.data?.data ?? []) {
      for (const s of v.stacks ?? []) set.add(s)
    }
    return [...set].sort()
  }, [query.data])

  const rows = query.data?.data ?? []

  return (
    <div className="page">
      <div className="page-head">
        <h1>{t('volumes.title')}</h1>
        <p className="muted">
          {t('common.shown', { n: rows.length })} · {t('common.snapshot')} {formatAgeMs(query.data?.snapshotAgeMs)}
        </p>
      </div>

      <div className="toolbar">
        <input
          className="input"
          placeholder={t('volumes.search')}
          value={q}
          onChange={(e) => setQ(e.target.value)}
        />
        <select className="select" value={stack} onChange={(e) => setStack(e.target.value)}>
          <option value="">{t('common.allStacks')}</option>
          {stacks.map((s) => (
            <option key={s} value={s}>
              {s}
            </option>
          ))}
        </select>
      </div>

      {query.isError ? <div className="banner danger">{(query.error as Error).message}</div> : null}

      <div className="table-wrap">
        <table className="table">
          <thead>
            <tr>
              <th>{t('common.name')}</th>
              <th>{t('common.driver')}</th>
              <th className="num">{t('common.usage')}</th>
              <th className="num">{t('common.containers')}</th>
              <th>{t('common.stacks')}</th>
              <th>{t('common.shared')}</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((v) => (
              <tr key={v.name} className={selected === v.name ? 'row-selected' : undefined}>
                <td className="mono">
                  <button type="button" className="text-link linkish" onClick={() => setSelected(v.name)}>
                    {v.name}
                  </button>
                </td>
                <td>{v.driver}</td>
                <td className="num">
                  {formatByteMetric(v.usage)}{' '}
                  <ProvenanceHint provenanceId="volume.size" displayedValue={formatByteMetric(v.usage)} />
                </td>
                <td className="num">
                  {(v.containers ?? []).map((c, i) => (
                    <span key={c}>
                      {i > 0 ? ', ' : ''}
                      <Link className="text-link" to={`/containers?q=${encodeURIComponent(c)}`}>
                        {c}
                      </Link>
                    </span>
                  ))}
                  {(v.containers ?? []).length === 0 ? '0' : ''}
                </td>
                <td>
                  {(v.stacks ?? []).length === 0
                    ? '—'
                    : (v.stacks ?? []).map((s, i) => (
                        <span key={s}>
                          {i > 0 ? ', ' : ''}
                          <Link className="text-link" to={`/containers?stack=${encodeURIComponent(s)}`}>
                            {s}
                          </Link>
                        </span>
                      ))}
                </td>
                <td>{v.shared ? t('common.yes') : '—'}</td>
              </tr>
            ))}
            {!query.isLoading && rows.length === 0 ? (
              <tr>
                <td colSpan={6} className="muted center">
                  {t('volumes.empty')}
                </td>
              </tr>
            ) : null}
          </tbody>
        </table>
      </div>
      {selected ? <CliCommandsPanel kind="volume" entityRef={selected} /> : null}
    </div>
  )
}
