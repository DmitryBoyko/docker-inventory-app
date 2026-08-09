import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { Link } from 'react-router-dom'
import { fetchStacks } from '../api/client'
import { CliCommandsPanel } from '../components/CliCommandsPanel'
import { formatAgeMs, formatByteMetric, formatBytes, formatCpu } from '../lib/format'
import { useLiveState } from '../realtime/useLiveState'

export function StacksPage() {
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
        <h1>Stacks</h1>
        <p className="muted">
          {stacks.length} stacks · snapshot {formatAgeMs(query.data?.snapshotAgeMs)}
        </p>
      </div>

      {query.isError ? (
        <div className="banner danger">{(query.error as Error).message}</div>
      ) : null}

      <div className="table-wrap">
        <table className="table">
          <thead>
            <tr>
              <th>Stack</th>
              <th className="num">Running</th>
              <th className="num">CPU</th>
              <th className="num">Memory</th>
              <th className="num">Writable</th>
              <th className="num">Volumes</th>
              <th className="num">Unhealthy</th>
              <th className="num">Restarted</th>
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
                    {s.containers.length} containers ·{' '}
                    <Link className="text-link" to={`/containers?stack=${encodeURIComponent(s.name)}`}>
                      containers
                    </Link>
                    {' · '}
                    <Link className="text-link" to={`/graph?scope=stack&stack=${encodeURIComponent(s.name)}`}>
                      graph
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
                  No stacks yet.
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
