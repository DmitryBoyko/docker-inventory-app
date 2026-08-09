import { useQuery } from '@tanstack/react-query'
import ReactECharts from 'echarts-for-react'
import { useEffect, useMemo, useState } from 'react'
import { fetchSystemResources } from '../api/client'
import { qk } from '../api/queryClient'
import { useT } from '../i18n'
import { formatBytes, formatCpu } from '../lib/format'
import { getTheme, type Theme } from '../lib/prefs'
import type { HistoryPoint } from '../realtime/types'
import { useLiveState } from '../realtime/useLiveState'
import { HistoryCharts } from './HistoryCharts'

const POLL_MAX = 60

function chartTheme(theme: Theme) {
  const light = theme === 'light'
  return {
    muted: light ? '#5c6b7a' : '#8b9bb0',
    axis: light ? '#c5ced9' : '#2a3544',
    split: light ? '#e2e8f0' : '#2a3544',
    cpu: light ? '#0077c2' : '#3db8ff',
    cpuFill: light ? 'rgba(0,119,194,0.12)' : 'rgba(61,184,255,0.12)',
    mem: light ? '#1f8a55' : '#3ecf8e',
  }
}

function buildOption(
  history: HistoryPoint[],
  t: (key: string) => string,
  theme: Theme,
) {
  const c = chartTheme(theme)
  const labels = history.map((p) =>
    new Date(p.t).toLocaleTimeString([], { minute: '2-digit', second: '2-digit' }),
  )
  return {
    backgroundColor: 'transparent',
    animation: false,
    grid: { left: 52, right: 56, top: 32, bottom: 28 },
    tooltip: { trigger: 'axis' },
    legend: {
      data: [t('charts.cpuSeries'), t('charts.memSeries')],
      textStyle: { color: c.muted },
      top: 0,
    },
    xAxis: {
      type: 'category',
      data: labels,
      axisLabel: { color: c.muted },
      axisLine: { lineStyle: { color: c.axis } },
    },
    yAxis: [
      {
        type: 'value',
        name: t('common.cpu'),
        nameTextStyle: { color: c.muted },
        axisLabel: { color: c.muted, formatter: (v: number) => `${v}` },
        splitLine: { lineStyle: { color: c.split } },
      },
      {
        type: 'value',
        name: t('common.memory'),
        nameTextStyle: { color: c.muted },
        axisLabel: {
          color: c.muted,
          formatter: (v: number) => formatBytes(v),
        },
        splitLine: { show: false },
      },
    ],
    series: [
      {
        name: t('charts.cpuSeries'),
        type: 'line',
        showSymbol: false,
        data: history.map((p) => Number(p.cpu.toFixed(2))),
        lineStyle: { color: c.cpu, width: 2 },
        areaStyle: { color: c.cpuFill },
      },
      {
        name: t('charts.memSeries'),
        type: 'line',
        yAxisIndex: 1,
        showSymbol: false,
        data: history.map((p) => p.mem),
        lineStyle: { color: c.mem, width: 2 },
      },
    ],
  }
}

/** Live WS tip (~1 min) plus SQLite history (ADR-015). Falls back to REST resources while WS/stats catch up. */
export function LiveCharts() {
  const t = useT()
  const live = useLiveState()
  const [theme, setThemeState] = useState<Theme>(() => getTheme())
  const [pollHistory, setPollHistory] = useState<HistoryPoint[]>([])

  useEffect(() => {
    const sync = () => setThemeState(getTheme())
    const obs = new MutationObserver(sync)
    obs.observe(document.documentElement, { attributes: true, attributeFilter: ['data-theme'] })
    return () => obs.disconnect()
  }, [])

  const resources = useQuery({
    queryKey: qk.systemResources,
    queryFn: fetchSystemResources,
    refetchInterval: live.connected ? 12_000 : 5_000,
    placeholderData: (prev) => prev,
  })

  useEffect(() => {
    const d = resources.data?.data
    if (!d) return
    const cpu = d.cpuPercent ?? 0
    const mem = d.memoryBytes ?? 0
    setPollHistory((prev) => {
      const last = prev[prev.length - 1]
      // Avoid duplicate points when React Query refetches the same payload.
      if (last && Math.abs(last.cpu - cpu) < 0.01 && last.mem === mem && Date.now() - last.t < 2000) {
        return prev
      }
      return [...prev, { t: Date.now(), cpu, mem }].slice(-POLL_MAX)
    })
  }, [resources.dataUpdatedAt, resources.data?.data])

  const history = live.history.length >= 2 ? live.history : pollHistory
  const option = useMemo(() => buildOption(history, t, theme), [history, t, theme])
  const last = history[history.length - 1] ?? live.history[live.history.length - 1]
  const r = resources.data?.data
  const fallbackCpu = last?.cpu ?? r?.cpuPercent
  const fallbackMem = last?.mem ?? r?.memoryBytes

  const sourceLabel = live.connected
    ? live.history.length >= 2
      ? t('charts.websocket')
      : t('charts.wsWaitingStats')
    : t('charts.restPoll')

  return (
    <div className="charts-stack">
      <HistoryCharts
        scope="host"
        title={t('charts.hostTitle')}
        rangeHours={1}
        liveTip={history}
      />
      <section className="panel live-panel">
        <div className="panel-head">
          <h2>{t('charts.liveTip')}</h2>
          <span className="muted tiny live-panel-status">
            {sourceLabel}
            {fallbackCpu != null ? ` · ${formatCpu(fallbackCpu)}` : ''}
            {fallbackMem != null ? ` · ${formatBytes(fallbackMem)}` : ''}
          </span>
        </div>
        {history.length < 2 ? (
          <div className="live-panel-empty">
            <p className="muted">{t('charts.waiting')}</p>
            {(fallbackCpu != null || fallbackMem != null) && (
              <dl className="live-panel-now">
                <div>
                  <dt>{t('common.cpu')}</dt>
                  <dd className="mono">{formatCpu(fallbackCpu)}</dd>
                </div>
                <div>
                  <dt>{t('common.memory')}</dt>
                  <dd className="mono">{formatBytes(fallbackMem)}</dd>
                </div>
              </dl>
            )}
          </div>
        ) : (
          <ReactECharts
            option={option}
            style={{ height: 220 }}
            opts={{ renderer: 'canvas' }}
            notMerge
            lazyUpdate
          />
        )}
      </section>
    </div>
  )
}
