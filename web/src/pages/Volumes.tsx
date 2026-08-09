import { useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link, useSearchParams } from 'react-router-dom'
import { fetchVolumes } from '../api/client'
import { formatAgeMs, formatByteMetric } from '../lib/format'
import { useLiveState } from '../realtime/useLiveState'

export function VolumesPage() {
  const live = useLiveState()
  const [params] = useSearchParams()
  const [q, setQ] = useState(params.get('q') ?? '')
  const [stack, setStack] = useState(params.get('stack') ?? '')

  const query = useQuery({
    queryKey: ['volumes', { q, stack }],
    queryFn: () => fetchVolumes({ q: q || undefined, stack: stack || undefined }),
    refetchInterval: live.connected ? 20000 : 5000,
  })

  const stacks = useMemo(() => {
    const set = new Set<string>()
    for (const v of query.data?.data ?? []) {
      for (const s of v.stacks ?? []) set.add(s)
    }
    return [...set].sort()
  }, [query.data])

  const rows = query.data?.data ?? []

  return (
    <div className="page">
      <div className="page-head">
        <h1>Volumes</h1>
        <p className="muted">
          {rows.length} shown · snapshot {formatAgeMs(query.data?.snapshotAgeMs)}
        </p>
      </div>

      <div className="toolbar">
        <input
          className="input"
          placeholder="Search volume name…"
          value={q}
          onChange={(e) => setQ(e.target.value)}
        />
        <select className="select" value={stack} onChange={(e) => setStack(e.target.value)}>
          <option value="">All stacks</option>
          {stacks.map((s) => (
            <option key={s} value={s}>
              {s}
            </option>
          ))}
        </select>
      </div>

      {query.isError ? <div className="banner danger">{(query.error as Error).message}</div> : null}

      <div className="table-wrap">
        <table className="table">
          <thead>
            <tr>
              <th>Name</th>
              <th>Driver</th>
              <th className="num">Usage</th>
              <th className="num">Containers</th>
              <th>Stacks</th>
              <th>Shared</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((v) => (
              <tr key={v.name}>
                <td className="mono">{v.name}</td>
                <td>{v.driver}</td>
                <td className="num">{formatByteMetric(v.usage)}</td>
                <td className="num">
                  {(v.containers ?? []).map((c, i) => (
                    <span key={c}>
                      {i > 0 ? ', ' : ''}
                      <Link className="text-link" to={`/containers?q=${encodeURIComponent(c)}`}>
                        {c}
                      </Link>
                    </span>
                  ))}
                  {(v.containers ?? []).length === 0 ? '0' : ''}
                </td>
                <td>
                  {(v.stacks ?? []).length === 0
                    ? '—'
                    : (v.stacks ?? []).map((s, i) => (
                        <span key={s}>
                          {i > 0 ? ', ' : ''}
                          <Link className="text-link" to={`/containers?stack=${encodeURIComponent(s)}`}>
                            {s}
                          </Link>
                        </span>
                      ))}
                </td>
                <td>{v.shared ? 'yes' : '—'}</td>
              </tr>
            ))}
            {!query.isLoading && rows.length === 0 ? (
              <tr>
                <td colSpan={6} className="muted center">
                  No volumes match filters.
                </td>
              </tr>
            ) : null}
          </tbody>
        </table>
      </div>
    </div>
  )
}
