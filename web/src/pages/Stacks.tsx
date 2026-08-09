import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { Link } from 'react-router-dom'
import { fetchStacks } from '../api/client'
import { CliCommandsPanel } from '../components/CliCommandsPanel'
import { useT } from '../i18n'
import { formatAgeMs, formatByteMetric, formatBytes, formatCpu } from '../lib/format'
import { useLiveState } from '../realtime/useLiveState'

export function StacksPage() {
  const t = useT()
  const live = useLiveState()
  const [selected, setSelected] = useState('')
  const query = useQuery({
    queryKey: ['stacks'],
    queryFn: fetchStacks,
    refetchInterval: live.connected ? 20000 : 2000,
  })

  const stacks = query.data?.data ?? []

  return (
    <div className="page">
      <div className="page-head">
        <h1>{t('stacks.title')}</h1>
        <p className="muted">
          {t('common.stacks')}: {stacks.length} · {t('common.snapshot')} {formatAgeMs(query.data?.snapshotAgeMs)}
        </p>
      </div>

      {query.isError ? (
        <div className="banner danger">{(query.error as Error).message}</div>
      ) : null}

      <div className="table-wrap">
        <table className="table">
          <thead>
            <tr>
              <th>{t('common.stack')}</th>
              <th className="num">{t('stacks.running')}</th>
              <th className="num">{t('common.cpu')}</th>
              <th className="num">{t('common.memory')}</th>
              <th className="num">{t('stacks.writable')}</th>
              <th className="num">{t('stacks.volumes')}</th>
              <th className="num">{t('stacks.unhealthy')}</th>
              <th className="num">{t('stacks.restarted')}</th>
            </tr>
          </thead>
          <tbody>
            {stacks.map((s) => (
              <tr key={s.name} className={selected === s.name ? 'row-selected' : undefined}>
                <td>
                  <button type="button" className="text-link linkish mono" onClick={() => setSelected(s.name)}>
                    {s.name}
                  </button>
                  <div className="muted tiny">
                    {t('stacks.containersCount', { n: s.containers.length })} ·{' '}
                    <Link className="text-link" to={`/containers?stack=${encodeURIComponent(s.name)}`}>
                      {t('common.containers').toLowerCase()}
                    </Link>
                    {' · '}
                    <Link className="text-link" to={`/graph?scope=stack&stack=${encodeURIComponent(s.name)}`}>
                      {t('common.graph')}
                    </Link>
                  </div>
                </td>
                <td className="num">
                  {s.resources.runningCount}/{s.resources.containerCount}
                </td>
                <td className="num">{formatCpu(s.resources.cpuPercent)}</td>
                <td className="num">{formatBytes(s.resources.memoryBytes)}</td>
                <td className="num">{formatByteMetric(s.resources.writableLayer)}</td>
                <td className="num">{formatByteMetric(s.volumeUsage)}</td>
                <td className="num">{s.unhealthyCount}</td>
                <td className="num">{s.restartedCount}</td>
              </tr>
            ))}
            {!query.isLoading && stacks.length === 0 ? (
              <tr>
                <td colSpan={8} className="muted center">
                  {t('stacks.empty')}
                </td>
              </tr>
            ) : null}
          </tbody>
        </table>
      </div>
      {selected ? <CliCommandsPanel kind="stack" entityRef={selected} /> : null}
    </div>
  )
}
