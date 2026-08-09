import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect, useState } from 'react'
import { fetchHosts } from '../api/client'
import { getSelectedHost, setSelectedHost } from '../lib/prefs'

export function HostPicker() {
  const qc = useQueryClient()
  const [selected, setSelected] = useState(() => getSelectedHost())
  const hostsQ = useQuery({
    queryKey: ['hosts'],
    queryFn: fetchHosts,
    refetchInterval: 15000,
    retry: false,
  })

  const hosts = hostsQ.data?.data ?? []
  const defaultHost = hostsQ.data?.defaultHost ?? 'default'

  useEffect(() => {
    if (!hosts.length) return
    const names = new Set(hosts.map((h) => h.name))
    if (selected && !names.has(selected)) {
      setSelected('')
      setSelectedHost('')
      void qc.invalidateQueries()
    }
  }, [hosts, selected, qc])

  if (hostsQ.isError || hosts.length <= 1) {
    return null
  }

  const value = selected || defaultHost

  return (
    <label className="host-picker">
      <span className="host-picker-label">Host</span>
      <select
        className="select host-select"
        value={value}
        aria-label="Docker host"
        onChange={(e) => {
          const name = e.target.value
          setSelected(name)
          setSelectedHost(name)
          void qc.clear()
        }}
      >
        {hosts.map((h) => (
          <option key={h.name} value={h.name}>
            {h.name}
            {h.isDefault ? ' (default)' : ''}
            {h.connected ? '' : ' · offline'}
          </option>
        ))}
      </select>
    </label>
  )
}
