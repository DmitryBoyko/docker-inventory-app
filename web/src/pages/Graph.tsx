import cytoscape, { type Core, type ElementDefinition } from 'cytoscape'
import fcose from 'cytoscape-fcose'
import { useEffect, useMemo, useRef, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { fetchGraph, fetchStacks } from '../api/client'
import type { Graph as GraphModel } from '../api/types'
import { formatAgeMs } from '../lib/format'
import { useLiveState } from '../realtime/useLiveState'

cytoscape.use(fcose)

const style = [
  {
    selector: 'node',
    style: {
      label: 'data(label)',
      'font-size': '10px',
      color: '#e7eef7',
      'text-valign': 'center',
      'text-halign': 'center',
      'text-wrap': 'ellipsis',
      'text-max-width': '90px',
      'background-color': '#2a3544',
      width: '36px',
      height: '36px',
      'border-width': 2,
      'border-color': '#3db8ff',
    },
  },
  {
    selector: 'node[type = "stack"]',
    style: {
      shape: 'round-rectangle',
      width: '70px',
      height: '40px',
      'background-color': '#1a6f9c',
      'border-color': '#3db8ff',
      'font-weight': 700,
    },
  },
  {
    selector: 'node[type = "service"]',
    style: {
      shape: 'round-rectangle',
      'background-color': '#243044',
      'border-color': '#6a7dff',
    },
  },
  {
    selector: 'node[type = "container"]',
    style: {
      shape: 'ellipse',
      'background-color': '#1e3d32',
      'border-color': '#3ecf8e',
    },
  },
  {
    selector: 'node[type = "network"]',
    style: {
      shape: 'diamond',
      'background-color': '#3a2f1a',
      'border-color': '#e6b84d',
    },
  },
  {
    selector: 'node[type = "volume"]',
    style: {
      shape: 'barrel',
      'background-color': '#332840',
      'border-color': '#c084fc',
    },
  },
  {
    selector: 'node[type = "image"]',
    style: {
      shape: 'hexagon',
      'background-color': '#2a2438',
      'border-color': '#a78bfa',
    },
  },
  {
    selector: 'edge',
    style: {
      width: 1.5,
      'line-color': '#3a4658',
      'target-arrow-color': '#3a4658',
      'target-arrow-shape': 'triangle',
      'curve-style': 'bezier',
      label: 'data(type)',
      'font-size': '8px',
      color: '#8b9bb0',
      'text-rotation': 'autorotate',
    },
  },
] as unknown as cytoscape.StylesheetJson

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
  const live = useLiveState()
  const navigate = useNavigate()
  const [params, setParams] = useSearchParams()
  const scope = params.get('scope') === 'stack' ? 'stack' : 'all'
  const stack = params.get('stack') ?? ''
  const [showImages, setShowImages] = useState(true)

  const stacksQ = useQuery({
    queryKey: ['stacks'],
    queryFn: fetchStacks,
    refetchInterval: live.connected ? 20000 : 5000,
  })

  const graphQ = useQuery({
    queryKey: ['graph', { scope, stack }],
    queryFn: () => fetchGraph({ scope, stack: scope === 'stack' ? stack : undefined }),
    enabled: scope === 'all' || stack.length > 0,
    refetchInterval: live.connected ? 15000 : 5000,
  })

  const hostRef = useRef<HTMLDivElement | null>(null)
  const cyRef = useRef<Core | null>(null)

  const filtered = useMemo(() => {
    const g = graphQ.data?.data
    if (!g) return null
    if (showImages) return g
    const drop = new Set(g.nodes.filter((n) => n.type === 'image').map((n) => n.id))
    return {
      ...g,
      nodes: g.nodes.filter((n) => n.type !== 'image'),
      edges: g.edges.filter((e) => e.type !== 'uses_image' && !drop.has(e.target) && !drop.has(e.source)),
    }
  }, [graphQ.data, showImages])

  useEffect(() => {
    if (!hostRef.current || !filtered) return

    if (cyRef.current) {
      cyRef.current.destroy()
      cyRef.current = null
    }

    const cy = cytoscape({
      container: hostRef.current,
      elements: toElements(filtered),
      style,
      layout: {
        name: 'fcose',
        animate: false,
        padding: 30,
        nodeSeparation: 90,
        idealEdgeLength: 110,
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
  }, [filtered, navigate, stack])

  const stackNames = (stacksQ.data?.data ?? []).map((s) => s.name)

  return (
    <div className="page">
      <div className="page-head">
        <h1>Graph</h1>
        <p className="muted">
          {filtered ? `${filtered.nodes.length} nodes · ${filtered.edges.length} edges` : '—'}
          {' · '}
          snapshot {formatAgeMs(graphQ.data?.snapshotAgeMs)}
        </p>
      </div>

      <div className="toolbar">
        <select
          className="select"
          value={scope}
          onChange={(e) => {
            const next = e.target.value
            const p = new URLSearchParams()
            p.set('scope', next)
            if (next === 'stack' && stack) p.set('stack', stack)
            else if (next === 'stack' && stackNames[0]) p.set('stack', stackNames[0])
            setParams(p, { replace: true })
          }}
        >
          <option value="all">All stacks</option>
          <option value="stack">One stack</option>
        </select>
        {scope === 'stack' ? (
          <select
            className="select"
            value={stack}
            onChange={(e) => {
              const p = new URLSearchParams()
              p.set('scope', 'stack')
              p.set('stack', e.target.value)
              setParams(p, { replace: true })
            }}
          >
            <option value="">Select stack…</option>
            {stackNames.map((n) => (
              <option key={n} value={n}>
                {n}
              </option>
            ))}
          </select>
        ) : null}
        <label className="check-row">
          <input type="checkbox" checked={showImages} onChange={(e) => setShowImages(e.target.checked)} />
          Show images
        </label>
        <button
          type="button"
          className="btn"
          onClick={() =>
            cyRef.current
              ?.layout({ name: 'fcose', animate: true } as unknown as cytoscape.LayoutOptions)
              .run()
          }
        >
          Relayout
        </button>
      </div>

      {graphQ.isError ? <div className="banner danger">{(graphQ.error as Error).message}</div> : null}
      {scope === 'stack' && !stack ? <div className="banner info">Choose a stack to render the graph.</div> : null}

      <div className="graph-legend muted tiny">
        stack · service · container · network · volume · image — click a node to open the related list
      </div>
      <div className="graph-host" ref={hostRef} />
    </div>
  )
}
