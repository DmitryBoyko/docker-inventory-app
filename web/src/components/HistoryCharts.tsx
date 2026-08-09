import { useQuery } from '@tanstack/react-query'
import ReactECharts from 'echarts-for-react'
import { useEffect, useMemo, useState } from 'react'
import { ApiError, fetchMetricsHistory } from '../api/client'
import { useT } from '../i18n'
import { formatBytes, formatCpu } from '../lib/format'
import { getSelectedHost, getTheme, type Theme } from '../lib/prefs'

type Props = {
  scope: 'host' | 'container'
  id?: string
  title?: string
  rangeHours?: number
  /** Merge live tip points (host dashboard). */
  liveTip?: Array<{ t: number; cpu: number; mem: number }>
}

function rangeISO(hours: number) {
  const to = new Date()
  const from = new Date(to.getTime() - hours * 3600_000)
  return { from: from.toISOString(), to: to.toISOString() }
}

function axisColors(theme: Theme) {
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

export function HistoryCharts({
  scope,
  id,
  title,
  rangeHours = 1,
  liveTip,
}: Props) {
  const t = useT()
  const chartTitle = title ?? t('charts.history')
  const host = getSelectedHost() || 'default'
  const [theme, setThemeState] = useState<Theme>(() => getTheme())
  const window = rangeISO(rangeHours)

  useEffect(() => {
    const sync = () => setThemeState(getTheme())
    const obs = new MutationObserver(sync)
    obs.observe(document.documentElement, { attributes: true, attributeFilter: ['data-theme'] })
    return () => obs.disconnect()
  }, [])

  const q = useQuery({
    queryKey: ['metrics', 'history', host, scope, id, rangeHours],
    queryFn: () =>
      fetchMetricsHistory({
        scope,
        id,
        from: window.from,
        to: window.to,
        step: rangeHours <= 1 ? '10s' : rangeHours <= 6 ? '1m' : '5m',
      }),
    refetchInterval: 30_000,
    retry: false,
  })

  const points = useMemo(() => {
    const base = q.data?.data.points ?? []
    if (!liveTip?.length) return base
    const lastT = base.length ? base[base.length - 1].t : 0
    const tip = liveTip.filter((p) => p.t > lastT)
    return [...base, ...tip]
  }, [q.data, liveTip])

  const option = useMemo(() => {
    const c = axisColors(theme)
    const labels = points.map((p) =>
      new Date(p.t).toLocaleTimeString([], {
        hour: '2-digit',
        minute: '2-digit',
        second: rangeHours <= 1 ? '2-digit' : undefined,
      }),
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
          axisLabel: { color: c.muted },
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
          data: points.map((p) => Number(p.cpu.toFixed(2))),
          lineStyle: { color: c.cpu, width: 2 },
          areaStyle: { color: c.cpuFill },
        },
        {
          name: t('charts.memSeries'),
          type: 'line',
          yAxisIndex: 1,
          showSymbol: false,
          data: points.map((p) => p.mem),
          lineStyle: { color: c.mem, width: 2 },
        },
      ],
    }
  }, [points, rangeHours, t, theme])

  const last = points[points.length - 1]
  const disabled = q.error instanceof ApiError && q.error.code === 'metrics_disabled'

  return (
    <section className="panel">
      <div className="panel-head">
        <h2>{chartTitle}</h2>
        <span className="muted tiny">
          {t('charts.lastHours', { n: rangeHours })}
          {last ? ` · ${formatCpu(last.cpu)} · ${formatBytes(last.mem)}` : null}
        </span>
      </div>
      {disabled ? (
        <p className="muted">{t('charts.disabled')}</p>
      ) : q.isError ? (
        <p className="muted">{(q.error as Error).message}</p>
      ) : points.length < 2 ? (
        <p className="muted">{t('charts.collecting')}</p>
      ) : (
        <ReactECharts option={option} style={{ height: 260 }} opts={{ renderer: 'canvas' }} notMerge lazyUpdate />
      )}
    </section>
  )
}
