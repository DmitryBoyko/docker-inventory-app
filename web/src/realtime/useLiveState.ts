import { useSyncExternalStore } from 'react'
import { getLiveState, subscribeLive } from './store'

export function useLiveState() {
  return useSyncExternalStore(subscribeLive, getLiveState, getLiveState)
}
