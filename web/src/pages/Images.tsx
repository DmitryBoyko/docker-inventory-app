import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link, useSearchParams } from 'react-router-dom'
import { fetchImages } from '../api/client'
import { CliCommandsPanel } from '../components/CliCommandsPanel'
import { ProvenanceHint } from '../components/ProvenanceHint'
import { useT } from '../i18n'
import { formatAgeMs, formatBytes } from '../lib/format'
import { useLiveState } from '../realtime/useLiveState'

export function ImagesPage() {
  const t = useT()
  const live = useLiveState()
  const [params] = useSearchParams()
  const [q, setQ] = useState(params.get('q') ?? '')
  const [dangling, setDangling] = useState(params.get('dangling') ?? '')
  const [selected, setSelected] = useState('')

  const query = useQuery({
    queryKey: ['images', { q, dangling }],
    queryFn: () => fetchImages({ q: q || undefined, dangling: dangling || undefined }),
    refetchInterval: live.connected ? 20000 : 5000,
  })

  const rows = query.data?.data ?? []

  return (
    <div className="page">
      <div className="page-head">
        <h1>{t('images.title')}</h1>
        <p className="muted">
          {t('common.shown', { n: rows.length })} · {t('common.snapshot')} {formatAgeMs(query.data?.snapshotAgeMs)}
        </p>
      </div>

      <div className="toolbar">
        <input
          className="input"
          placeholder={t('images.search')}
          value={q}
          onChange={(e) => setQ(e.target.value)}
        />
        <select className="select" value={dangling} onChange={(e) => setDangling(e.target.value)}>
          <option value="">{t('images.all')}</option>
          <option value="false">{t('images.tagged')}</option>
          <option value="true">{t('images.danglingFilter')}</option>
        </select>
      </div>

      {query.isError ? <div className="banner danger">{(query.error as Error).message}</div> : null}

      <div className="table-wrap">
        <table className="table dense">
          <thead>
            <tr>
              <th>{t('images.tags')}</th>
              <th className="num">{t('common.size')}</th>
              <th className="num">{t('common.shared')}</th>
              <th className="num">{t('common.containers')}</th>
              <th>{t('images.dangling')}</th>
              <th>{t('common.id')}</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((img) => {
              const tag = (img.repoTags ?? []).find((x) => x !== '<none>:<none>') || img.id
              return (
              <tr key={img.id} className={selected === tag ? 'row-selected' : undefined}>
                <td className="mono">
                  <button type="button" className="text-link linkish" onClick={() => setSelected(tag)}>
                    {(img.repoTags ?? []).filter((x) => x !== '<none>:<none>').join(', ') || '—'}
                  </button>
                </td>
                <td className="num">
                  {formatBytes(img.sizeBytes)}{' '}
                  <ProvenanceHint provenanceId="image.size" displayedValue={formatBytes(img.sizeBytes)} />
                </td>
                <td className="num">
                  {img.sharedSizeBytes == null ? 'n/a' : formatBytes(img.sharedSizeBytes)}
                </td>
                <td className="num">
                  {(img.containers ?? []).map((c, i) => (
                    <span key={c}>
                      {i > 0 ? ', ' : ''}
                      <Link className="text-link" to={`/containers?q=${encodeURIComponent(c)}`}>
                        {c}
                      </Link>
                    </span>
                  ))}
                  {(img.containers ?? []).length === 0 ? String(img.containerCount) : ''}
                </td>
                <td>{img.dangling ? <span className="pill health-unhealthy">{t('images.dangling')}</span> : '—'}</td>
                <td className="mono">{img.idShort}</td>
              </tr>
              )
            })}
            {!query.isLoading && rows.length === 0 ? (
              <tr>
                <td colSpan={6} className="muted center">
                  {t('images.empty')}
                </td>
              </tr>
            ) : null}
          </tbody>
        </table>
      </div>
      {selected ? <CliCommandsPanel kind="image" entityRef={selected} /> : null}
    </div>
  )
}
