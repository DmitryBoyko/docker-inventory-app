import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { fetchDiagnostics } from '../api/client'
import type { FindingSeverity } from '../api/types'
import { useT } from '../i18n'

const sevClass: Record<FindingSeverity, string> = {
  INFO: 'sev-info',
  WARNING: 'sev-warn',
  CRITICAL: 'sev-crit',
}

function entityPath(kind: string, id?: string, name?: string) {
  const ref = id || name || ''
  switch (kind) {
    case 'container':
      return `/containers/${encodeURIComponent(ref)}`
    case 'image':
      return `/images?q=${encodeURIComponent(name || ref)}`
    case 'volume':
      return `/volumes?q=${encodeURIComponent(name || ref)}`
    case 'network':
      return `/networks?q=${encodeURIComponent(name || ref)}`
    default:
      return '/'
  }
}

export function DiagnosticsPage() {
  const t = useT()
  const q = useQuery({
    queryKey: ['diagnostics'],
    queryFn: fetchDiagnostics,
    refetchInterval: 10000,
  })

  const findings = q.data?.data ?? []

  return (
    <div className="page">
      <div className="page-head">
        <div>
          <h1>{t('diag.title')}</h1>
          <p className="muted">{t('diag.subtitle')}</p>
        </div>
      </div>
      {q.isError && <div className="banner danger">{(q.error as Error).message}</div>}
      {!q.isLoading && findings.length === 0 && <div className="panel">{t('diag.empty')}</div>}
      <ul className="findings-list">
        {findings.map((f) => (
          <li key={f.id} className="panel finding-card">
            <div className="finding-head">
              <span className={`pill ${sevClass[f.severity]}`}>
                {t(`diag.severity.${f.severity.toLowerCase()}` as 'diag.severity.info')}
              </span>
              <strong>{f.title}</strong>
            </div>
            <p>{f.description}</p>
            <p className="muted">
              <strong>{t('diag.why')}:</strong> {f.reason}
            </p>
            <p className="muted">
              <strong>{t('diag.recommend')}:</strong> {f.recommendation}
            </p>
            {f.relatedCommands && f.relatedCommands.length > 0 && (
              <p className="mono small muted">{f.relatedCommands.join(' · ')}</p>
            )}
            <Link className="text-link" to={entityPath(f.entity.kind, f.entity.id, f.entity.name)}>
              {t('diag.open')} {f.entity.name || f.entity.id}
            </Link>
          </li>
        ))}
      </ul>
    </div>
  )
}
