import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { ApiError, fetchReady } from '../api/client'
import { useT } from '../i18n'
import { useLiveState } from '../realtime/useLiveState'

export function StatusBanner() {
  const t = useT()
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
        {t('status.unauthorizedBefore')}{' '}
        <Link className="text-link" to="/settings">
          {t('nav.settings')}
        </Link>
      </div>
    )
  }

  const docker = live.docker ?? ready.data?.docker
  const events = live.events ?? (ready.data?.events
    ? { connected: ready.data.events.connected, error: ready.data.events.error ?? undefined }
    : null)

  if (ready.isLoading && !docker) {
    return <div className="banner info">{t('status.checking')}</div>
  }

  const connected = docker?.connected ?? ready.data?.ready
  if (!connected) {
    const msg =
      docker?.error ||
      ready.data?.error?.message ||
      (ready.error instanceof Error ? ready.error.message : t('status.unavailable'))
    return (
      <div className="banner danger">
        {t('status.disconnected')}
        {docker?.host ? ` (${docker.host})` : ''}: {msg}
      </div>
    )
  }

  const eventsBit = events
    ? events.connected
      ? ` · ${t('status.eventsLive')}`
      : ` · ${t('status.eventsPolling')}${events.error ? ` (${events.error})` : ''}`
    : ''

  const hostName = ready.data?.host
  return (
    <div className="banner ok">
      {t('status.connected')}
      {hostName ? ` · ${t('common.host')} ${hostName}` : ''}
      {docker?.host ? ` · ${docker.host}` : ''}
      {docker?.apiVersion ? ` · API ${docker.apiVersion}` : ''}
      {docker?.osType ? ` · ${docker.osType}` : ''}
      {live.connected ? ` · ${t('status.ws')}` : ` · ${t('status.wsReconnecting')}`}
      {eventsBit}
    </div>
  )
}
