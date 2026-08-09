import { useQuery } from '@tanstack/react-query'
import ReactECharts from 'echarts-for-react'
import { useMemo } from 'react'
import { ApiError, fetchMetricsHistory } from '../api/client'
import { useT } from '../i18n'
import { formatBytes, formatCpu } from '../lib/format'
import { getSelectedHost } from '../lib/prefs'

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
  const window = rangeISO(rangeHours)
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
      grid: { left: 48, right: 48, top: 28, bottom: 28 },
      tooltip: { trigger: 'axis' },
      legend: {
        data: [t('charts.cpuSeries'), t('charts.memSeries')],
        textStyle: { color: '#8b9bb0' },
        top: 0,
      },
      xAxis: {
        type: 'category',
        data: labels,
        axisLabel: { color: '#8b9bb0' },
        axisLine: { lineStyle: { color: '#2a3544' } },
      },
      yAxis: [
        {
          type: 'value',
          name: t('common.cpu'),
          axisLabel: { color: '#8b9bb0' },
          splitLine: { lineStyle: { color: '#2a3544' } },
        },
        {
          type: 'value',
          name: t('common.memory'),
          axisLabel: {
            color: '#8b9bb0',
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
          lineStyle: { color: '#3db8ff', width: 2 },
          areaStyle: { color: 'rgba(61,184,255,0.12)' },
        },
        {
          name: t('charts.memSeries'),
          type: 'line',
          yAxisIndex: 1,
          showSymbol: false,
          data: points.map((p) => p.mem),
          lineStyle: { color: '#3ecf8e', width: 2 },
        },
      ],
    }
  }, [points, rangeHours, t])

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
        <ReactECharts option={option} style={{ height: 260 }} opts={{ renderer: 'canvas' }} />
      )}
    </section>
  )
}
