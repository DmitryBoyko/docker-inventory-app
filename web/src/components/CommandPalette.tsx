import { useQuery } from '@tanstack/react-query'
import { useEffect, useMemo, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { fetchContainers, fetchEntityCommands } from '../api/client'
import type { RenderedCommand } from '../api/types'
import { useT } from '../i18n'

type NavItem = { id: string; label: string; path: string; keywords: string }
type CmdItem = { id: string; label: string; path: string; command: string; keywords: string }

type Props = {
  open: boolean
  onClose: () => void
}

export function CommandPalette({ open, onClose }: Props) {
  const t = useT()
  const nav = useNavigate()
  const inputRef = useRef<HTMLInputElement>(null)
  const [q, setQ] = useState('')
  const [idx, setIdx] = useState(0)

  const containersQ = useQuery({
    queryKey: ['containers'],
    queryFn: () => fetchContainers(),
    enabled: open,
    staleTime: 10_000,
  })

  const sampleName = containersQ.data?.data?.[0]?.name ?? 'nginx'

  const cmdsQ = useQuery({
    queryKey: ['palette-cmds', sampleName],
    queryFn: async () => {
      const [ctr, sys] = await Promise.all([
        fetchEntityCommands('container', sampleName, 'bash'),
        fetchEntityCommands('system', '', 'bash'),
      ])
      return [...(ctr.data ?? []), ...(sys.data ?? [])] as RenderedCommand[]
    },
    enabled: open,
  })

  const navItems: NavItem[] = useMemo(() => {
    const base: NavItem[] = [
      { id: 'nav-dash', label: t('nav.dashboard'), path: '/', keywords: 'dashboard home' },
      { id: 'nav-ctr', label: t('nav.containers'), path: '/containers', keywords: 'containers' },
      { id: 'nav-stacks', label: t('nav.stacks'), path: '/stacks', keywords: 'stacks compose' },
      { id: 'nav-net', label: t('nav.networks'), path: '/networks', keywords: 'networks' },
      { id: 'nav-vol', label: t('nav.volumes'), path: '/volumes', keywords: 'volumes' },
      { id: 'nav-img', label: t('nav.images'), path: '/images', keywords: 'images' },
      { id: 'nav-graph', label: t('nav.graph'), path: '/graph', keywords: 'graph' },
      { id: 'nav-diag', label: t('nav.diagnostics'), path: '/diagnostics', keywords: 'diagnostics findings' },
      { id: 'nav-snap', label: t('nav.snapshots'), path: '/snapshots', keywords: 'snapshots diff' },
      { id: 'nav-set', label: t('nav.settings'), path: '/settings', keywords: 'settings' },
    ]
    for (const c of containersQ.data?.data ?? []) {
      base.push({
        id: `open-${c.id}`,
        label: `${t('diag.open')} ${c.name}`,
        path: `/containers/${encodeURIComponent(c.id)}`,
        keywords: `open container ${c.name} ${c.idShort}`,
      })
      base.push({
        id: `cli-${c.id}`,
        label: `CLI ${c.name}`,
        path: `/containers/${encodeURIComponent(c.id)}?tab=commands`,
        keywords: `inspect logs stats cli commands ${c.name}`,
      })
    }
    return base
  }, [containersQ.data, t])

  const cmdItems: CmdItem[] = useMemo(() => {
    const out: CmdItem[] = []
    for (const c of containersQ.data?.data ?? []) {
      for (const def of ['inspect', 'logs', 'stats'] as const) {
        out.push({
          id: `${def}-${c.id}`,
          label: `${def} ${c.name}`,
          path: `/containers/${encodeURIComponent(c.id)}?tab=commands`,
          command: `docker ${def === 'stats' ? 'stats --no-stream' : def} ${c.name}`,
          keywords: `${def} ${c.name} container`,
        })
      }
    }
    for (const r of cmdsQ.data ?? []) {
      if (r.entityKind === 'system') {
        out.push({
          id: `sys-${r.definitionId}`,
          label: t(r.titleKey) !== r.titleKey ? t(r.titleKey) : r.title,
          path: '/settings',
          command: r.command,
          keywords: `system ${r.definitionId} ${r.command}`,
        })
      }
    }
    return out
  }, [containersQ.data, cmdsQ.data, t])

  const needle = q.trim().toLowerCase()
  const filteredNav = useMemo(
    () => navItems.filter((i) => !needle || i.label.toLowerCase().includes(needle) || i.keywords.includes(needle)),
    [navItems, needle],
  )
  const filteredCmd = useMemo(
    () =>
      cmdItems.filter(
        (i) =>
          !needle ||
          i.label.toLowerCase().includes(needle) ||
          i.keywords.includes(needle) ||
          i.command.toLowerCase().includes(needle),
      ),
    [cmdItems, needle],
  )

  const flat = useMemo(
    () => [
      ...filteredCmd.slice(0, 12).map((i) => ({ type: 'cmd' as const, ...i })),
      ...filteredNav.slice(0, 12).map((i) => ({ type: 'nav' as const, ...i })),
    ],
    [filteredCmd, filteredNav],
  )

  useEffect(() => {
    if (open) {
      setQ('')
      setIdx(0)
      queueMicrotask(() => inputRef.current?.focus())
    }
  }, [open])

  useEffect(() => {
    setIdx(0)
  }, [q])

  useEffect(() => {
    if (!open) return
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.preventDefault()
        onClose()
      } else if (e.key === 'ArrowDown') {
        e.preventDefault()
        setIdx((v) => Math.min(v + 1, Math.max(flat.length - 1, 0)))
      } else if (e.key === 'ArrowUp') {
        e.preventDefault()
        setIdx((v) => Math.max(v - 1, 0))
      } else if (e.key === 'Enter') {
        e.preventDefault()
        const item = flat[idx]
        if (item) {
          nav(item.path)
          onClose()
        }
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [open, flat, idx, nav, onClose])

  if (!open) return null

  return (
    <div className="modal-backdrop palette-backdrop" onClick={onClose} role="presentation">
      <div className="palette panel" role="dialog" aria-modal="true" aria-label={t('palette.title')} onClick={(e) => e.stopPropagation()}>
        <input
          ref={inputRef}
          className="palette-input"
          placeholder={t('palette.placeholder')}
          value={q}
          onChange={(e) => setQ(e.target.value)}
        />
        {flat.length === 0 && <p className="muted">{t('palette.empty')}</p>}
        {filteredCmd.length > 0 && <div className="palette-section">{t('palette.commands')}</div>}
        <ul className="palette-list">
          {flat.map((item, i) =>
            item.type === 'cmd' ? (
              <li key={item.id}>
                <button
                  type="button"
                  className={i === idx ? 'palette-row active' : 'palette-row'}
                  onMouseEnter={() => setIdx(i)}
                  onClick={() => {
                    nav(item.path)
                    onClose()
                  }}
                >
                  <span>{item.label}</span>
                  <span className="mono muted small">{item.command}</span>
                </button>
              </li>
            ) : (
              <li key={item.id}>
                <button
                  type="button"
                  className={i === idx ? 'palette-row active' : 'palette-row'}
                  onMouseEnter={() => setIdx(i)}
                  onClick={() => {
                    nav(item.path)
                    onClose()
                  }}
                >
                  <span>{item.label}</span>
                  <span className="muted small">{item.path}</span>
                </button>
              </li>
            ),
          )}
        </ul>
      </div>
    </div>
  )
}
