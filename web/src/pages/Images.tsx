import { useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link, useSearchParams } from 'react-router-dom'
import { fetchImages } from '../api/client'
import type { Image } from '../api/types'
import { qk } from '../api/queryClient'
import { CliCommandsPanel } from '../components/CliCommandsPanel'
import { ProvenanceHint } from '../components/ProvenanceHint'
import { useT } from '../i18n'
import { formatAgeMs, formatBytes } from '../lib/format'
import { useDebouncedValue } from '../lib/useDebouncedValue'
import { useGrowingAgeMs } from '../lib/useGrowingAgeMs'
import { useLiveConnected } from '../realtime/useLiveState'

function namedTags(img: Image): string[] {
  return (img.repoTags ?? []).filter((x) => x && x !== '<none>:<none>')
}

export function ImagesPage() {
  const t = useT()
  const wsConnected = useLiveConnected()
  const [params] = useSearchParams()
  const [q, setQ] = useState(params.get('q') ?? '')
  const [dangling, setDangling] = useState(params.get('dangling') ?? '')
  const [selected, setSelected] = useState('')
  const qDebounced = useDebouncedValue(q, 250)

  const query = useQuery({
    queryKey: qk.images({ q: qDebounced || undefined, dangling: dangling || undefined }),
    queryFn: () => fetchImages({ q: qDebounced || undefined, dangling: dangling || undefined }),
    refetchInterval: wsConnected ? 20_000 : 8_000,
    placeholderData: (prev) => prev,
  })

  const rows = query.data?.data ?? []
  const untaggedCount = useMemo(() => rows.filter((img) => img.dangling || namedTags(img).length === 0).length, [rows])
  const dataAgeMs = useGrowingAgeMs(query.data?.snapshotAgeMs, query.dataUpdatedAt)

  return (
    <div className="page page-fill">
      <div className="page-head">
        <div>
          <h1>{t('images.title')}</h1>
          <p className="muted page-lead">{t('images.lead')}</p>
          <p className="muted">
            {t('common.shown', { n: rows.length })}
            {untaggedCount > 0 ? ` · ${t('images.untaggedCount', { n: untaggedCount })}` : ''}
            {' · '}
            {t('common.dataUpdated', { age: formatAgeMs(dataAgeMs) })}
          </p>
        </div>
      </div>

      <div className="toolbar">
        <input
          className="input"
          placeholder={t('images.search')}
          value={q}
          onChange={(e) => setQ(e.target.value)}
          aria-label={t('images.search')}
        />
        <select
          className="select"
          value={dangling}
          onChange={(e) => setDangling(e.target.value)}
          aria-label={t('images.filterLabel')}
        >
          <option value="">{t('images.all')}</option>
          <option value="false">{t('images.tagged')}</option>
          <option value="true">{t('images.danglingFilter')}</option>
        </select>
      </div>

      {query.isError ? <div className="banner danger">{(query.error as Error).message}</div> : null}

      <div className="table-wrap table-wrap-fill">
        <table className="table dense">
          <thead>
            <tr>
              <th>{t('images.name')}</th>
              <th>{t('images.status')}</th>
              <th className="num">{t('common.size')}</th>
              <th className="num">{t('images.sharedSize')}</th>
              <th>{t('images.usedBy')}</th>
              <th>{t('common.id')}</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((img) => {
              const tags = namedTags(img)
              const untagged = img.dangling || tags.length === 0
              const selectRef = tags[0] || img.id
              const containers = img.containers ?? []
              return (
                <tr key={img.id} className={selected === selectRef ? 'row-selected' : undefined}>
                  <td className={untagged ? 'wrap' : 'mono wrap'}>
                    <button type="button" className="text-link linkish" onClick={() => setSelected(selectRef)}>
                      {untagged ? (
                        <span className="muted-strong">{t('images.noName')}</span>
                      ) : (
                        tags.join(', ')
                      )}
                    </button>
                    {untagged ? <div className="cell-hint muted">{t('images.noNameHint')}</div> : null}
                  </td>
                  <td>
                    {untagged ? (
                      <span className="pill health-unhealthy" title={t('images.statusUntaggedTitle')}>
                        {t('images.statusUntagged')}
                      </span>
                    ) : (
                      <span className="pill health-healthy">{t('images.statusNamed')}</span>
                    )}
                  </td>
                  <td className="num">
                    {formatBytes(img.sizeBytes)}{' '}
                    <ProvenanceHint provenanceId="image.size" displayedValue={formatBytes(img.sizeBytes)} />
                  </td>
                  <td className="num muted">
                    {img.sharedSizeBytes == null ? t('images.sharedUnknown') : formatBytes(img.sharedSizeBytes)}
                  </td>
                  <td className="wrap">
                    {containers.length > 0 ? (
                      containers.map((c, i) => (
                        <span key={c}>
                          {i > 0 ? ', ' : ''}
                          <Link className="text-link" to={`/containers?q=${encodeURIComponent(c)}`}>
                            {c}
                          </Link>
                        </span>
                      ))
                    ) : img.containerCount > 0 ? (
                      t('images.containersCount', { n: img.containerCount })
                    ) : (
                      <span className="muted">{t('images.unused')}</span>
                    )}
                  </td>
                  <td className="mono muted">{img.idShort}</td>
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
