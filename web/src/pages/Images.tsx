import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link, useSearchParams } from 'react-router-dom'
import { fetchImages } from '../api/client'
import { formatAgeMs, formatBytes } from '../lib/format'
import { useLiveState } from '../realtime/useLiveState'

export function ImagesPage() {
  const live = useLiveState()
  const [params] = useSearchParams()
  const [q, setQ] = useState(params.get('q') ?? '')
  const [dangling, setDangling] = useState(params.get('dangling') ?? '')

  const query = useQuery({
    queryKey: ['images', { q, dangling }],
    queryFn: () => fetchImages({ q: q || undefined, dangling: dangling || undefined }),
    refetchInterval: live.connected ? 20000 : 5000,
  })

  const rows = query.data?.data ?? []

  return (
    <div className="page">
      <div className="page-head">
        <h1>Images</h1>
        <p className="muted">
          {rows.length} shown · snapshot {formatAgeMs(query.data?.snapshotAgeMs)}
        </p>
      </div>

      <div className="toolbar">
        <input
          className="input"
          placeholder="Search tag, id, container…"
          value={q}
          onChange={(e) => setQ(e.target.value)}
        />
        <select className="select" value={dangling} onChange={(e) => setDangling(e.target.value)}>
          <option value="">All images</option>
          <option value="false">Tagged</option>
          <option value="true">Dangling</option>
        </select>
      </div>

      {query.isError ? <div className="banner danger">{(query.error as Error).message}</div> : null}

      <div className="table-wrap">
        <table className="table dense">
          <thead>
            <tr>
              <th>Tags</th>
              <th className="num">Size</th>
              <th className="num">Shared</th>
              <th className="num">Containers</th>
              <th>Dangling</th>
              <th>ID</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((img) => (
              <tr key={img.id}>
                <td className="mono">
                  {(img.repoTags ?? []).filter((t) => t !== '<none>:<none>').join(', ') || '—'}
                </td>
                <td className="num">{formatBytes(img.sizeBytes)}</td>
                <td className="num">
                  {img.sharedSizeBytes == null ? 'n/a' : formatBytes(img.sharedSizeBytes)}
                </td>
                <td className="num">
                  {(img.containers ?? []).map((c, i) => (
                    <span key={c}>
                      {i > 0 ? ', ' : ''}
                      <Link className="text-link" to={`/containers?q=${encodeURIComponent(c)}`}>
                        {c}
                      </Link>
                    </span>
                  ))}
                  {(img.containers ?? []).length === 0 ? String(img.containerCount) : ''}
                </td>
                <td>{img.dangling ? <span className="pill health-unhealthy">dangling</span> : '—'}</td>
                <td className="mono">{img.idShort}</td>
              </tr>
            ))}
            {!query.isLoading && rows.length === 0 ? (
              <tr>
                <td colSpan={6} className="muted center">
                  No images match filters.
                </td>
              </tr>
            ) : null}
          </tbody>
        </table>
      </div>
    </div>
  )
}
