import type { Container } from '../api/types'
import {
  containerExposureScope,
  routesForContainer,
  type ExposureScope,
} from '../lib/exposure'

type TFn = (key: string, params?: Record<string, string | number>) => string

type Props = {
  container: Container
  t: TFn
  compact?: boolean
}

export function ExposureBadge({ container, t, compact }: Props) {
  const scope: ExposureScope = containerExposureScope(container)
  const routes = routesForContainer(container)

  if (scope === 'internal' || routes.length === 0) {
    return (
      <span className="pill exposure-internal" title={t('exposure.scope.internal')}>
        {t('exposure.badge.internal')}
      </span>
    )
  }

  const preview = routes
    .slice(0, compact ? 1 : 3)
    .map((r) => `${r.hostIP}:${r.hostPort}→${r.containerPort}`)
    .join(', ')
  const more = routes.length > (compact ? 1 : 3) ? ` +${routes.length - (compact ? 1 : 3)}` : ''

  return (
    <span className="exposure-cell" title={routes.map((r) => `${r.hostIP}:${r.hostPort} → ${r.containerPort}`).join('\n')}>
      <span className={`pill exposure-${scope}`}>{t(`exposure.scope.${scope}`)}</span>
      <span className="mono exposure-preview">
        {preview}
        {more}
      </span>
    </span>
  )
}
