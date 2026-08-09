import { useQuery } from '@tanstack/react-query'
import { useMemo, useState } from 'react'
import { fetchEntityCommands } from '../api/client'
import type { RenderedCommand, RiskLevel } from '../api/types'
import { useT } from '../i18n'
import { getCliShell, setCliShell, type CliShell } from '../lib/prefs'

const riskClass: Record<RiskLevel, string> = {
  READ_ONLY: 'risk-ro',
  INTERACTIVE: 'risk-interactive',
  STATE_CHANGING: 'risk-state',
  DESTRUCTIVE: 'risk-destructive',
}

const riskKey: Record<RiskLevel, string> = {
  READ_ONLY: 'cli.risk.read_only',
  INTERACTIVE: 'cli.risk.interactive',
  STATE_CHANGING: 'cli.risk.state_changing',
  DESTRUCTIVE: 'cli.risk.destructive',
}

type Props = {
  kind: string
  entityRef: string
}

export function CliCommandsPanel({ kind, entityRef }: Props) {
  const t = useT()
  const [shell, setShell] = useState<CliShell>(() => getCliShell())
  const [copied, setCopied] = useState<string | null>(null)

  const q = useQuery({
    queryKey: ['commands', kind, entityRef, shell],
    queryFn: () => fetchEntityCommands(kind, entityRef, shell),
    enabled: kind === 'system' || !!entityRef,
    staleTime: 60_000,
  })

  const rows = useMemo(() => {
    const list = q.data?.data ?? []
    const byDef = new Map<string, RenderedCommand>()
    for (const c of list) {
      if (c.shell === shell) byDef.set(c.definitionId, c)
    }
    return [...byDef.values()]
  }, [q.data, shell])

  const onShell = (s: CliShell) => {
    setCliShell(s)
    setShell(s)
  }

  const copy = async (cmd: string, id: string) => {
    try {
      await navigator.clipboard.writeText(cmd)
      setCopied(id)
      window.setTimeout(() => setCopied(null), 1500)
    } catch {
      /* ignore */
    }
  }

  return (
    <section className="panel cli-panel">
      <div className="panel-head">
        <h3>{t('cli.title')}</h3>
        <div className="shell-toggle" role="group" aria-label={t('cli.shell')}>
          {(['bash', 'powershell', 'cmd'] as CliShell[]).map((s) => (
            <button
              key={s}
              type="button"
              className={shell === s ? 'btn active' : 'btn'}
              onClick={() => onShell(s)}
            >
              {s === 'powershell' ? 'PowerShell' : s === 'cmd' ? 'CMD' : 'Bash'}
            </button>
          ))}
        </div>
      </div>
      {q.isError && <div className="banner danger">{(q.error as Error).message}</div>}
      {!q.isLoading && rows.length === 0 && <p className="muted">{t('cli.empty')}</p>}
      <ul className="cli-list">
        {rows.map((row) => (
          <li key={row.definitionId} className="cli-item">
            <div className="cli-item-head">
              <strong>{t(row.titleKey) !== row.titleKey ? t(row.titleKey) : row.title}</strong>
              <span className={`pill ${riskClass[row.riskLevel]}`}>{t(riskKey[row.riskLevel])}</span>
            </div>
            <p className="muted small">
              {t(row.descriptionKey) !== row.descriptionKey ? t(row.descriptionKey) : row.description}
            </p>
            <pre className="cli-cmd mono">{row.command}</pre>
            <button type="button" className="btn" onClick={() => copy(row.command, row.definitionId)}>
              {copied === row.definitionId ? t('cli.copied') : t('cli.copy')}
            </button>
          </li>
        ))}
      </ul>
    </section>
  )
}
