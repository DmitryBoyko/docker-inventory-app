import cytoscape, { type Core, type ElementDefinition } from 'cytoscape'
import fcose from 'cytoscape-fcose'
import { useEffect, useMemo, useRef, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { fetchGraph, fetchStacks } from '../api/client'
import { qk } from '../api/queryClient'
import type { Graph as GraphModel } from '../api/types'
import { useT } from '../i18n'
import { formatAgeMs } from '../lib/format'
import { getTheme, type Theme } from '../lib/prefs'
import { useGrowingAgeMs } from '../lib/useGrowingAgeMs'
import { useLiveConnected } from '../realtime/useLiveState'

cytoscape.use(fcose)

function graphStylesheet(theme: Theme): cytoscape.StylesheetJson {
  const light = theme === 'light'
  const label = light ? '#15202b' : '#f2f7fc'
  const edge = light ? '#8a97a8' : '#6a788c'
  const edgeLabel = light ? '#5c6b7a' : '#9aabc0'

  const fills = light
    ? {
        default: '#d9e2ec',
        stack: '#b8daf0',
        service: '#c9d4f5',
        container: '#c5e8d8',
        network: '#f3e2b3',
        volume: '#e4d4f5',
        image: '#d9d2f0',
      }
    : {
        default: '#2f3d4f',
        stack: '#1f7aad',
        service: '#334066',
        container: '#245a48',
        network: '#6a5520',
        volume: '#4a3560',
        image: '#3d3558',
      }

  const borders = light
    ? {
        default: '#0077c2',
        stack: '#0077c2',
        service: '#4a5fd4',
        container: '#1f8a55',
        network: '#b8860b',
        volume: '#8b5cf6',
        image: '#7c3aed',
      }
    : {
        default: '#3db8ff',
        stack: '#3db8ff',
        service: '#6a7dff',
        container: '#3ecf8e',
        network: '#e6b84d',
        volume: '#c084fc',
        image: '#a78bfa',
      }

  return [
    {
      selector: 'node',
      style: {
        label: 'data(label)',
        'font-size': 12,
        'font-weight': 600,
        color: label,
        'text-valign': 'center',
        'text-halign': 'center',
        'text-wrap': 'wrap',
        'text-max-width': 88,
        'text-overflow-wrap': 'anywhere',
        'text-outline-width': light ? 0 : 2,
        'text-outline-color': fills.default,
        'background-color': fills.default,
        shape: 'round-rectangle',
        width: 96,
        height: 56,
        'border-width': 2,
        'border-color': borders.default,
      },
    },
    {
      selector: 'node[type = "stack"]',
      style: {
        width: 120,
        height: 58,
        'text-max-width': 108,
        'background-color': fills.stack,
        'border-color': borders.stack,
        'text-outline-color': fills.stack,
        'font-size': 13,
        'font-weight': 700,
      },
    },
    {
      selector: 'node[type = "service"]',
      style: {
        width: 104,
        height: 54,
        'text-max-width': 92,
        'background-color': fills.service,
        'border-color': borders.service,
        'text-outline-color': fills.service,
      },
    },
    {
      selector: 'node[type = "container"]',
      style: {
        width: 110,
        height: 54,
        'text-max-width': 98,
        'background-color': fills.container,
        'border-color': borders.container,
        'text-outline-color': fills.container,
      },
    },
    {
      selector: 'node[type = "network"]',
      style: {
        width: 100,
        height: 52,
        'text-max-width': 88,
        'background-color': fills.network,
        'border-color': borders.network,
        'text-outline-color': fills.network,
      },
    },
    {
      selector: 'node[type = "volume"]',
      style: {
        width: 100,
        height: 52,
        'text-max-width': 88,
        'background-color': fills.volume,
        'border-color': borders.volume,
        'text-outline-color': fills.volume,
      },
    },
    {
      selector: 'node[type = "image"]',
      style: {
        width: 110,
        height: 54,
        'text-max-width': 98,
        'background-color': fills.image,
        'border-color': borders.image,
        'text-outline-color': fills.image,
      },
    },
    {
      selector: 'edge',
      style: {
        width: 1.5,
        'line-color': edge,
        'target-arrow-color': edge,
        'target-arrow-shape': 'triangle',
        'curve-style': 'bezier',
        label: 'data(type)',
        'font-size': 9,
        color: edgeLabel,
        'text-rotation': 'autorotate',
        'text-background-opacity': 0.9,
        'text-background-padding': 2,
        'text-background-shape': 'round-rectangle',
        'text-background-color': light ? '#ffffff' : '#0f1419',
      },
    },
  ] as unknown as cytoscape.StylesheetJson
}

function toElements(g: GraphModel): ElementDefinition[] {
  const nodes: ElementDefinition[] = g.nodes.map((n) => ({
    data: {
      id: n.id,
      label: n.label,
      type: n.type,
      ...n.data,
    },
  }))
  const edges: ElementDefinition[] = g.edges.map((e) => ({
    data: {
      id: e.id,
      source: e.source,
      target: e.target,
      type: e.type,
    },
  }))
  return [...nodes, ...edges]
}

export function GraphPage() {
  const t = useT()
  const wsConnected = useLiveConnected()
  const navigate = useNavigate()
  const [params, setParams] = useSearchParams()
  const scope = params.get('scope') === 'stack' ? 'stack' : 'all'
  const stack = params.get('stack') ?? ''
  const [showImages, setShowImages] = useState(true)
  const [paused, setPaused] = useState(false)
  /** Snapshot of API graph while paused (same ref as last live data — avoids relayout on pause). */
  const [frozenGraph, setFrozenGraph] = useState<GraphModel | null>(null)
  const [theme, setThemeState] = useState<Theme>(() => getTheme())

  useEffect(() => {
    const sync = () => setThemeState(getTheme())
    const obs = new MutationObserver(sync)
    obs.observe(document.documentElement, { attributes: true, attributeFilter: ['data-theme'] })
    return () => obs.disconnect()
  }, [])

  const stacksQ = useQuery({
    queryKey: qk.stacks,
    queryFn: fetchStacks,
    refetchInterval: paused ? false : wsConnected ? 20_000 : 8_000,
  })

  const graphQ = useQuery({
    queryKey: qk.graph(scope, stack),
    queryFn: () => fetchGraph({ scope, stack: scope === 'stack' ? stack : undefined }),
    enabled: scope === 'all' || stack.length > 0,
    refetchInterval: paused ? false : wsConnected ? 15_000 : 8_000,
    refetchOnWindowFocus: !paused,
    placeholderData: (prev) => prev,
  })

  const hostRef = useRef<HTMLDivElement | null>(null)
  const cyRef = useRef<Core | null>(null)

  const sourceGraph = paused && frozenGraph ? frozenGraph : (graphQ.data?.data ?? null)

  const filtered = useMemo(() => {
    const g = sourceGraph
    if (!g) return null
    if (showImages) return g
    const drop = new Set(g.nodes.filter((n) => n.type === 'image').map((n) => n.id))
    return {
      ...g,
      nodes: g.nodes.filter((n) => n.type !== 'image'),
      edges: g.edges.filter((e) => e.type !== 'uses_image' && !drop.has(e.target) && !drop.has(e.source)),
    }
  }, [sourceGraph, showImages])

  function togglePause() {
    if (!paused) {
      // Keep the exact object currently on screen — no cytoscape destroy/relayout.
      if (graphQ.data?.data) setFrozenGraph(graphQ.data.data)
      setPaused(true)
    } else {
      setPaused(false)
      setFrozenGraph(null)
      void graphQ.refetch()
    }
  }

  // Scope/stack change must leave pause so the new graph can load.
  useEffect(() => {
    setPaused(false)
    setFrozenGraph(null)
  }, [scope, stack])

  const topologyKey = useMemo(() => {
    if (!filtered) return ''
    return [
      showImages ? '1' : '0',
      ...filtered.nodes.map((n) => n.id),
      ...filtered.edges.map((e) => e.id),
    ].join('\0')
  }, [filtered, showImages])

  const filteredRef = useRef(filtered)
  filteredRef.current = filtered

  useEffect(() => {
    if (!hostRef.current || !filteredRef.current) return
    const data = filteredRef.current

    if (cyRef.current) {
      cyRef.current.destroy()
      cyRef.current = null
    }

    const cy = cytoscape({
      container: hostRef.current,
      elements: toElements(data),
      style: graphStylesheet(theme),
      layout: {
        name: 'fcose',
        animate: false,
        padding: 36,
        nodeSeparation: 120,
        idealEdgeLength: 150,
      } as unknown as cytoscape.LayoutOptions,
      wheelSensitivity: 0.25,
    })
    cyRef.current = cy

    cy.on('tap', 'node', (evt) => {
      const n = evt.target
      const type = String(n.data('type') ?? '')
      const label = String(n.data('label') ?? '')
      if (type === 'container') {
        const short = String(n.data('idShort') ?? '')
        navigate(short ? `/containers/${encodeURIComponent(short)}` : `/containers?q=${encodeURIComponent(label)}`)
      } else if (type === 'stack') {
        navigate(`/containers?stack=${encodeURIComponent(label)}`)
      } else if (type === 'network') {
        navigate(`/networks?q=${encodeURIComponent(label)}`)
      } else if (type === 'volume') {
        navigate(`/volumes?q=${encodeURIComponent(label)}`)
      } else if (type === 'image') {
        navigate(`/images?q=${encodeURIComponent(label)}`)
      } else if (type === 'service') {
        const st = String(n.data('stack') ?? stack)
        navigate(`/containers?stack=${encodeURIComponent(st)}&q=${encodeURIComponent(label)}`)
      }
    })

    return () => {
      cy.destroy()
      cyRef.current = null
    }
  }, [topologyKey, navigate, stack, theme])

  const stackNames = (stacksQ.data?.data ?? []).map((s) => s.name)
  const dataAgeMs = useGrowingAgeMs(graphQ.data?.snapshotAgeMs, graphQ.dataUpdatedAt)

  return (
    <div className="page">
      <div className="page-head">
        <h1>{t('graph.title')}</h1>
        <p className="muted">
          {filtered ? t('graph.nodesEdges', { nodes: filtered.nodes.length, edges: filtered.edges.length }) : '—'}
          {' · '}
          {t('common.dataUpdated', { age: formatAgeMs(dataAgeMs) })}
          {paused ? ` · ${t('graph.paused')}` : ` · ${t('graph.live')}`}
        </p>
      </div>

      <div className="toolbar">
        <select
          className="select"
          value={scope}
          disabled={paused}
          onChange={(e) => {
            const next = e.target.value
            const p = new URLSearchParams()
            p.set('scope', next)
            if (next === 'stack' && stack) p.set('stack', stack)
            else if (next === 'stack' && stackNames[0]) p.set('stack', stackNames[0])
            setParams(p, { replace: true })
          }}
        >
          <option value="all">{t('common.allStacks')}</option>
          <option value="stack">{t('graph.oneStack')}</option>
        </select>
        {scope === 'stack' ? (
          <select
            className="select"
            value={stack}
            disabled={paused}
            onChange={(e) => {
              const p = new URLSearchParams()
              p.set('scope', 'stack')
              p.set('stack', e.target.value)
              setParams(p, { replace: true })
            }}
          >
            <option value="">{t('graph.selectStack')}</option>
            {stackNames.map((n) => (
              <option key={n} value={n}>
                {n}
              </option>
            ))}
          </select>
        ) : null}
        <label className="check-row">
          <input type="checkbox" checked={showImages} onChange={(e) => setShowImages(e.target.checked)} />
          {t('graph.showImages')}
        </label>
        <button
          type="button"
          className={`btn${paused ? ' active' : ''}`}
          onClick={togglePause}
          title={paused ? t('graph.resumeHint') : t('graph.pauseHint')}
        >
          {paused ? t('graph.resume') : t('graph.pause')}
        </button>
        <button
          type="button"
          className="btn"
          disabled={!filtered}
          onClick={() =>
            cyRef.current
              ?.layout({
                name: 'fcose',
                animate: true,
                padding: 36,
                nodeSeparation: 120,
                idealEdgeLength: 150,
              } as unknown as cytoscape.LayoutOptions)
              .run()
          }
        >
          {t('graph.relayout')}
        </button>
      </div>

      {paused ? <div className="banner info">{t('graph.pausedBanner')}</div> : null}
      {graphQ.isError ? <div className="banner danger">{(graphQ.error as Error).message}</div> : null}
      {scope === 'stack' && !stack ? <div className="banner info">{t('graph.chooseStack')}</div> : null}

      <div className="graph-legend muted tiny">{t('graph.legend')}</div>
      <div className={`graph-host${paused ? ' graph-host-paused' : ''}`} ref={hostRef} />
    </div>
  )
}
