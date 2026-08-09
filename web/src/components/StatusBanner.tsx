import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { ApiError, fetchReady } from '../api/client'
import { useLiveState } from '../realtime/useLiveState'

export function StatusBanner() {
  const live = useLiveState()
  const ready = useQuery({
    queryKey: ['ready'],
    queryFn: fetchReady,
    refetchInterval: live.connected ? 30000 : 5000,
    retry: false,
  })

  if (ready.error instanceof ApiError && ready.error.status === 401) {
    return (
      <div className="banner warn">
        API unauthorized — set a token in <Link className="text-link" to="/settings">Settings</Link>
      </div>
    )
  }

  const docker = live.docker ?? ready.data?.docker
  const events = live.events ?? (ready.data?.events
    ? { connected: ready.data.events.connected, error: ready.data.events.error ?? undefined }
    : null)

  if (ready.isLoading && !docker) {
    return <div className="banner info">Checking Docker Engine…</div>
  }

  const connected = docker?.connected ?? ready.data?.ready
  if (!connected) {
    const msg =
      docker?.error ||
      ready.data?.error?.message ||
      (ready.error instanceof Error ? ready.error.message : 'Docker unavailable')
    return (
      <div className="banner danger">
        Docker disconnected{docker?.host ? ` (${docker.host})` : ''}: {msg}
      </div>
    )
  }

  const eventsBit = events
    ? events.connected
      ? ' · events live'
      : ` · events polling${events.error ? ` (${events.error})` : ''}`
    : ''

  const hostName = ready.data?.host
  return (
    <div className="banner ok">
      Connected
      {hostName ? ` · host ${hostName}` : ''}
      {docker?.host ? ` · ${docker.host}` : ''}
      {docker?.apiVersion ? ` · API ${docker.apiVersion}` : ''}
      {docker?.osType ? ` · ${docker.osType}` : ''}
      {live.connected ? ' · ws' : ' · ws…'}
      {eventsBit}
    </div>
  )
}
