import { Link } from 'react-router-dom'
import type { ExposureRouteRow, ExposureScope } from '../lib/exposure'
import { countByScope } from '../lib/exposure'

type TFn = (key: string, params?: Record<string, string | number>) => string

const scopes: ExposureScope[] = ['external', 'localhost', 'lan']

type Props = {
  routes: ExposureRouteRow[]
  t: TFn
}

export function ExposureMap({ routes, t }: Props) {
  const counts = countByScope(routes)
  const groups = scopes.map((scope) => ({
    scope,
    rows: routes.filter((r) => r.scope === scope),
  }))

  return (
    <section className="panel">
      <div className="panel-head">
        <h2>{t('exposure.mapTitle')}</h2>
        <span className="muted">
          {t('exposure.summaryCounts', {
            external: counts.external,
            localhost: counts.localhost,
            lan: counts.lan,
          })}
        </span>
      </div>
      <p className="muted exposure-hint">{t('exposure.mapHint')}</p>

      {routes.length === 0 ? (
        <p className="muted">{t('exposure.nonePublished')}</p>
      ) : (
        <div className="exposure-groups">
          {groups.map(({ scope, rows }) => (
            <div key={scope} className="exposure-group">
              <h3 className="exposure-group-title">
                <span className={`pill exposure-${scope}`}>{t(`exposure.scope.${scope}`)}</span>
                <span className="muted"> · {rows.length}</span>
              </h3>
              {rows.length === 0 ? (
                <p className="muted exposure-empty">{t('exposure.noRoutesInScope')}</p>
              ) : (
                <table className="table dense">
                  <thead>
                    <tr>
                      <th>{t('exposure.host')}</th>
                      <th className="num">{t('exposure.hostPort')}</th>
                      <th>{t('common.name')}</th>
                      <th>{t('exposure.containerPort')}</th>
                      <th>{t('common.health')}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {rows.map((r, i) => (
                      <tr key={`${r.containerId}-${r.hostIP}-${r.hostPort}-${r.containerPort}-${i}`}>
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
                        <td className="mono">{r.containerPort}</td>
                        <td>
                          <span className={`pill health-${r.health}`}>{r.health}</span>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </div>
          ))}
        </div>
      )}
    </section>
  )
}
