import { useEffect, useRef, useState, useSyncExternalStore } from 'react'
import type { ContainerStats } from '../api/types'
import { getContainerLiveStats, getLiveState, subscribeLive } from './store'

/** WS connected flag only — avoids 1 Hz re-renders from stats ticks. */
export function useLiveConnected() {
  return useSyncExternalStore(
    subscribeLive,
    () => getLiveState().connected,
    () => getLiveState().connected,
  )
}

export function useLiveState() {
  return useSyncExternalStore(subscribeLive, getLiveState, getLiveState)
}

type BannerSlice = {
  connected: boolean
  docker: ReturnType<typeof getLiveState>['docker']
  events: ReturnType<typeof getLiveState>['events']
}

let bannerCache: BannerSlice | null = null

function pickBanner(): BannerSlice {
  const s = getLiveState()
  const next = { connected: s.connected, docker: s.docker, events: s.events }
  if (
    bannerCache &&
    bannerCache.connected === next.connected &&
    bannerCache.docker === next.docker &&
    bannerCache.events === next.events
  ) {
    return bannerCache
  }
  bannerCache = next
  return next
}

/** Docker/events connection slices for banners (skip stats/history). */
export function useLiveConnectionBanner() {
  return useSyncExternalStore(subscribeLive, pickBanner, pickBanner)
}

/**
 * Throttled view of statsById for tables (default 2s) — cuts 1 Hz remount pressure (P1).
 */
export function useThrottledStatsById(intervalMs = 2000) {
  const [tick, setTick] = useState(0)
  const latest = useRef(getLiveState().statsById)

  useEffect(() => {
    return subscribeLive(() => {
      latest.current = getLiveState().statsById
    })
  }, [])

  useEffect(() => {
    const id = window.setInterval(() => setTick((n) => n + 1), intervalMs)
    return () => window.clearInterval(id)
  }, [intervalMs])

  // tick forces read of latest ref for merge consumers
  void tick
  return latest.current
}

/** Live stats for one container — re-renders only when that id's stats object changes. */
export function useContainerLiveStats(id: string, idShort?: string): ContainerStats | undefined {
  const get = () => getContainerLiveStats(id, idShort)
  return useSyncExternalStore(
    subscribeLive,
    () => get(),
    () => get(),
  )
}
