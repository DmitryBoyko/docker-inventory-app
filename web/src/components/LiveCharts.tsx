import ReactECharts from 'echarts-for-react'
import { useMemo } from 'react'
import { formatBytes, formatCpu } from '../lib/format'
import { useLiveState } from '../realtime/useLiveState'
import { HistoryCharts } from './HistoryCharts'

/** Live WS tip (~1 min) plus SQLite history (ADR-015). */
export function LiveCharts() {
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
        data: ['CPU %', 'Memory'],
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
          name: 'CPU',
          axisLabel: { color: '#8b9bb0', formatter: (v: number) => `${v}` },
          splitLine: { lineStyle: { color: '#2a3544' } },
        },
        {
          type: 'value',
          name: 'Mem',
          axisLabel: {
            color: '#8b9bb0',
            formatter: (v: number) => formatBytes(v),
          },
          splitLine: { show: false },
        },
      ],
      series: [
        {
          name: 'CPU %',
          type: 'line',
          showSymbol: false,
          data: live.history.map((p) => Number(p.cpu.toFixed(2))),
          lineStyle: { color: '#3db8ff', width: 2 },
          areaStyle: { color: 'rgba(61,184,255,0.12)' },
        },
        {
          name: 'Memory',
          type: 'line',
          yAxisIndex: 1,
          showSymbol: false,
          data: live.history.map((p) => p.mem),
          lineStyle: { color: '#3ecf8e', width: 2 },
        },
      ],
    }
  }, [live.history])

  const last = live.history[live.history.length - 1]

  return (
    <div className="charts-stack">
      <HistoryCharts
        scope="host"
        title="Host CPU / RAM (1h)"
        rangeHours={1}
        liveTip={live.history}
      />
      <section className="panel">
        <div className="panel-head">
          <h2>Live tip</h2>
          <span className="muted tiny">
            {live.connected ? 'websocket' : 'ws reconnecting…'}
            {last ? ` · ${formatCpu(last.cpu)} · ${formatBytes(last.mem)}` : null}
          </span>
        </div>
        {live.history.length < 2 ? (
          <p className="muted">Waiting for stats samples…</p>
        ) : (
          <ReactECharts option={option} style={{ height: 220 }} opts={{ renderer: 'canvas' }} />
        )}
      </section>
    </div>
  )
}
