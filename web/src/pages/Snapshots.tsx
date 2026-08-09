import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { ApiError, createSnapshot, deleteSnapshot, fetchSnapshotDiff, fetchSnapshots } from '../api/client'
import type { SnapshotChange, SnapshotDiff } from '../api/types'
import { useT } from '../i18n'

function ChangesBlock({ title, items, t }: { title: string; items: SnapshotChange[]; t: (k: string) => string }) {
  if (!items.length) return null
  return (
    <div className="diff-block">
      <h4>{title}</h4>
      <ul>
        {items.map((c) => (
          <li key={`${c.kind}-${c.id}`} className={`diff-${c.kind}`}>
            <span className="mono">
              {c.kind === 'added' ? '+' : c.kind === 'removed' ? '-' : '~'} {c.name}
            </span>
            <span className="muted small"> ({t(`snap.${c.kind}`)})</span>
            {c.fields?.map((f) => (
              <div key={f.field} className="muted small mono">
                {f.field}: {JSON.stringify(f.from)} → {JSON.stringify(f.to)}
              </div>
            ))}
          </li>
        ))}
      </ul>
    </div>
  )
}

function DiffView({ diff, t }: { diff: SnapshotDiff; t: (k: string) => string }) {
  return (
    <div className="panel">
      <h3>
        {diff.leftId} → {diff.rightId}
      </h3>
      <ChangesBlock title={t('common.containers')} items={diff.containers} t={t} />
      <ChangesBlock title={t('nav.images')} items={diff.images} t={t} />
      <ChangesBlock title={t('nav.volumes')} items={diff.volumes} t={t} />
      <ChangesBlock title={t('nav.networks')} items={diff.networks} t={t} />
      <ChangesBlock title={t('common.stacks')} items={diff.stacks} t={t} />
    </div>
  )
}

export function SnapshotsPage() {
  const t = useT()
  const qc = useQueryClient()
  const [label, setLabel] = useState('')
  const [diff, setDiff] = useState<SnapshotDiff | null>(null)

  const listQ = useQuery({
    queryKey: ['snapshots'],
    queryFn: fetchSnapshots,
  })

  const createM = useMutation({
    mutationFn: () => createSnapshot(label),
    onSuccess: () => {
      setLabel('')
      void qc.invalidateQueries({ queryKey: ['snapshots'] })
    },
  })

  const delM = useMutation({
    mutationFn: (id: string) => deleteSnapshot(id),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['snapshots'] }),
  })

  const disabled =
    listQ.isError &&
    listQ.error instanceof ApiError &&
    (listQ.error as ApiError).code === 'snapshots_disabled'

  return (
    <div className="page page-fill">
      <div className="page-head">
        <div>
          <h1>{t('snap.title')}</h1>
          <p className="muted">{t('snap.subtitle')}</p>
        </div>
      </div>

      {disabled && <div className="banner warn">{t('snap.disabled')}</div>}
      {listQ.isError && !disabled && <div className="banner danger">{(listQ.error as Error).message}</div>}

      <div className="panel toolbar-row">
        <input
          className="input"
          placeholder={t('snap.label')}
          value={label}
          onChange={(e) => setLabel(e.target.value)}
        />
        <button type="button" className="btn primary" disabled={createM.isPending || disabled} onClick={() => createM.mutate()}>
          {t('snap.create')}
        </button>
      </div>

      {!listQ.isLoading && (listQ.data?.data?.length ?? 0) === 0 && !disabled && (
        <p className="muted">{t('snap.empty')}</p>
      )}

      <div className="table-wrap table-wrap-fill">
        <table className="table">
          <thead>
            <tr>
              <th>{t('common.id')}</th>
              <th>{t('common.created')}</th>
              <th>{t('common.host')}</th>
              <th>{t('common.label')}</th>
              <th>{t('common.counts')}</th>
              <th />
            </tr>
          </thead>
          <tbody>
            {(listQ.data?.data ?? []).map((s) => (
              <tr key={s.id}>
                <td className="mono">{s.id}</td>
                <td>{new Date(s.createdAt).toLocaleString()}</td>
                <td>{s.hostName}</td>
                <td>{s.label || '—'}</td>
                <td className="mono small">
                  c{s.counts.containers}/i{s.counts.images}/v{s.counts.volumes}/n{s.counts.networks}
                </td>
                <td className="row-actions">
                  <button
                    type="button"
                    className="btn"
                    onClick={async () => {
                      const res = await fetchSnapshotDiff(s.id, 'current')
                      setDiff(res.data)
                    }}
                  >
                    {t('snap.diff')}
                  </button>
                  <button type="button" className="btn" onClick={() => delM.mutate(s.id)}>
                    {t('snap.delete')}
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {diff && <DiffView diff={diff} t={t} />}
    </div>
  )
}
