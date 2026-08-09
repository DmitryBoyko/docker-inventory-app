import { memo } from 'react'
import { Link } from 'react-router-dom'
import type { Container } from '../api/types'
import { ExposureBadge } from './ExposureBadge'
import { formatByteMetric, formatBytes, formatCpu, formatUptime } from '../lib/format'

type TFn = (key: string, params?: Record<string, string | number>) => string

type Props = {
  container: Container
  t: TFn
}

function ContainerTableRowImpl({ container: c, t }: Props) {
  return (
    <tr className="cv-row">
      <td className="mono">
        <Link className="text-link" to={`/containers/${encodeURIComponent(c.idShort)}`}>
          {c.name}
        </Link>
      </td>
      <td>{c.stack}</td>
      <td>{c.service ?? '—'}</td>
      <td>
        <span className={`pill state-${c.state}`}>
          {t(`containers.state.${c.state}` as 'containers.state.running')}
        </span>
      </td>
      <td>
        <span className={`pill health-${c.health}`}>
          {t(`containers.health.${c.health}` as 'containers.health.healthy')}
        </span>
      </td>
      <td>
        <ExposureBadge container={c} t={t} compact />
      </td>
      <td className="num">{formatCpu(c.stats?.cpuPercent)}</td>
      <td className="num">{formatBytes(c.stats?.memoryBytes)}</td>
      <td className="num">{formatByteMetric(c.writableLayer)}</td>
      <td className="num">{c.restartCount}</td>
      <td className="num">{formatUptime(c.uptimeSeconds)}</td>
      <td className="mono truncate" title={c.image}>
        {c.image}
      </td>
      <td className="mono">{c.idShort}</td>
    </tr>
  )
}

export const ContainerTableRow = memo(ContainerTableRowImpl)
