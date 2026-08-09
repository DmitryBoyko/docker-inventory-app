import { useQuery } from '@tanstack/react-query'
import { useEffect, useState } from 'react'
import { fetchProvenance } from '../api/client'
import { qk } from '../api/queryClient'
import type { ProvenanceSpec } from '../api/types'
import { tOr, useT } from '../i18n'

type Props = {
  provenanceId: string
  displayedValue?: string
}

export function ProvenanceHint({ provenanceId, displayedValue }: Props) {
  const t = useT()
  const [open, setOpen] = useState(false)
  const q = useQuery({
    queryKey: qk.provenance(provenanceId),
    queryFn: () => fetchProvenance(provenanceId) as Promise<{ data: ProvenanceSpec }>,
    enabled: open,
    staleTime: Infinity,
    gcTime: 30 * 60_000,
  })

  useEffect(() => {
    if (!open) return
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setOpen(false)
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [open])

  const spec = q.data?.data

  return (
    <span className="prov-wrap">
      <button
        type="button"
        className="prov-btn"
        title={t('prov.hint')}
        aria-label={t('prov.hint')}
        onClick={() => setOpen(true)}
      >
        ⓘ
      </button>
      {open && (
        <div className="modal-backdrop" onClick={() => setOpen(false)} role="presentation">
          <div
            className="modal panel"
            role="dialog"
            aria-modal="true"
            aria-label={t('prov.title')}
            onClick={(e) => e.stopPropagation()}
          >
            <div className="panel-head">
              <h3>{spec ? tOr(t, spec.titleKey, spec.title) : t('prov.title')}</h3>
              <button type="button" className="btn" onClick={() => setOpen(false)}>
                {t('prov.close')}
              </button>
            </div>
            {q.isLoading && <p className="muted">…</p>}
            {q.isError && <div className="banner danger">{(q.error as Error).message}</div>}
            {spec && (
              <dl className="kv">
                <div>
                  <dt>{t('prov.endpoint')}</dt>
                  <dd className="mono">{spec.apiEndpoint}</dd>
                </div>
                {spec.dockerField && (
                  <div>
                    <dt>{t('prov.field')}</dt>
                    <dd className="mono">{spec.dockerField}</dd>
                  </div>
                )}
                {spec.transformation && (
                  <div>
                    <dt>{t('prov.transform')}</dt>
                    <dd>{spec.transformationKey ? tOr(t, spec.transformationKey, spec.transformation) : spec.transformation}</dd>
                  </div>
                )}
                {displayedValue && (
                  <div>
                    <dt>{t('prov.ui')}</dt>
                    <dd>{displayedValue}</dd>
                  </div>
                )}
                {spec.chain && spec.chain.length > 0 && (
                  <div>
                    <dt>{t('prov.chain')}</dt>
                    <dd>{spec.chain.join(' → ')}</dd>
                  </div>
                )}
                <div>
                  <dt>{t('prov.title')}</dt>
                  <dd>{tOr(t, spec.descriptionKey, spec.description)}</dd>
                </div>
              </dl>
            )}
          </div>
        </div>
      )}
    </span>
  )
}
