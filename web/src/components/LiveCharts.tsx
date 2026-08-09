import ReactECharts from 'echarts-for-react'
import { useMemo } from 'react'
import { useT } from '../i18n'
import { formatBytes, formatCpu } from '../lib/format'
import { useLiveState } from '../realtime/useLiveState'
import { HistoryCharts } from './HistoryCharts'

/** Live WS tip (~1 min) plus SQLite history (ADR-015). */
export function LiveCharts() {
  const t = useT()
  const live = useLiveState()
  const option = useMemo(() => {
    const labels = live.history.map((p) =>
      new Date(p.t).toLocaleTimeString([], { minute: '2-digit', second: '2-digit' }),
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
          axisLabel: { color: '#8b9bb0', formatter: (v: number) => `${v}` },
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
          data: live.history.map((p) => Number(p.cpu.toFixed(2))),
          lineStyle: { color: '#3db8ff', width: 2 },
          areaStyle: { color: 'rgba(61,184,255,0.12)' },
        },
        {
          name: t('charts.memSeries'),
          type: 'line',
          yAxisIndex: 1,
          showSymbol: false,
          data: live.history.map((p) => p.mem),
          lineStyle: { color: '#3ecf8e', width: 2 },
        },
      ],
    }
  }, [live.history, t])

  const last = live.history[live.history.length - 1]

  return (
    <div className="charts-stack">
      <HistoryCharts
        scope="host"
        title={t('charts.hostTitle')}
        rangeHours={1}
        liveTip={live.history}
      />
      <section className="panel">
        <div className="panel-head">
          <h2>{t('charts.liveTip')}</h2>
          <span className="muted tiny">
            {live.connected ? t('charts.websocket') : t('charts.wsReconnect')}
            {last ? ` · ${formatCpu(last.cpu)} · ${formatBytes(last.mem)}` : null}
          </span>
        </div>
        {live.history.length < 2 ? (
          <p className="muted">{t('charts.waiting')}</p>
        ) : (
          <ReactECharts option={option} style={{ height: 220 }} opts={{ renderer: 'canvas' }} />
        )}
      </section>
    </div>
  )
}
