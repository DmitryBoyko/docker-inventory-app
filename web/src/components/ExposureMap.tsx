import { useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import type { ExposureRouteRow } from '../lib/exposure'
import { countByScope } from '../lib/exposure'

type TFn = (key: string, params?: Record<string, string | number>) => string

type Props = {
  routes: ExposureRouteRow[]
  t: TFn
}

export function ExposureMap({ routes, t }: Props) {
  const [stack, setStack] = useState('')

  const stacks = useMemo(() => {
    const set = new Set(routes.map((r) => r.stack).filter(Boolean))
    return [...set].sort((a, b) => a.localeCompare(b))
  }, [routes])

  const filtered = useMemo(
    () => (stack ? routes.filter((r) => r.stack === stack) : routes),
    [routes, stack],
  )

  const counts = countByScope(filtered)

  return (
    <section className="panel exposure-map">
      <div className="panel-head exposure-map-head">
        <h2>{t('exposure.mapTitle')}</h2>
        <div className="exposure-map-controls">
          <select
            className="select"
            value={stack}
            onChange={(e) => setStack(e.target.value)}
            aria-label={t('exposure.filterStack')}
          >
            <option value="">{t('exposure.allStacks')}</option>
            {stacks.map((s) => (
              <option key={s} value={s}>
                {s}
              </option>
            ))}
          </select>
          <span className="muted tiny">
            {t('exposure.summaryCounts', {
              external: counts.external,
              localhost: counts.localhost,
              lan: counts.lan,
            })}
          </span>
        </div>
      </div>

      {filtered.length === 0 ? (
        <p className="muted">{stack ? t('exposure.noneInStack') : t('exposure.nonePublished')}</p>
      ) : (
        <div className="table-wrap exposure-map-table">
          <table className="table dense">
            <thead>
              <tr>
                <th>{t('exposure.column')}</th>
                <th>{t('exposure.host')}</th>
                <th className="num">{t('exposure.hostPort')}</th>
                <th>{t('common.name')}</th>
                {!stack ? <th>{t('common.stack')}</th> : null}
                <th>{t('exposure.containerPort')}</th>
              </tr>
            </thead>
            <tbody>
              {filtered.map((r, i) => (
                <tr key={`${r.containerId}-${r.hostIP}-${r.hostPort}-${r.containerPort}-${i}`}>
                  <td>
                    <span className={`pill exposure-${r.scope}`}>{t(`exposure.scope.${r.scope}`)}</span>
                  </td>
                  <td className="mono">{r.hostIP}</td>
                  <td className="num mono">{r.hostPort}</td>
                  <td className="mono">
                    <Link
                      className="text-link"
                      to={`/containers/${encodeURIComponent(r.containerIdShort)}`}
                    >
                      {r.containerName}
                    </Link>
                  </td>
                  {!stack ? <td className="truncate">{r.stack}</td> : null}
                  <td className="mono">{r.containerPort}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  )
}
