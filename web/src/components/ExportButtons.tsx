import { useState } from 'react'
import { ApiError, downloadExport, type ExportFormat, type ExportScope } from '../api/client'

type Props = {
  compact?: boolean
}

export function ExportButtons({ compact }: Props) {
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState('')

  const run = async (format: ExportFormat, scope: ExportScope) => {
    setBusy(true)
    setErr('')
    try {
      await downloadExport(format, scope)
    } catch (e) {
      setErr(e instanceof ApiError ? e.message : (e as Error).message)
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className={compact ? 'export-bar compact' : 'export-bar'}>
      <div className="toolbar">
        <button type="button" className="btn" disabled={busy} onClick={() => run('json', 'all')}>
          Export JSON
        </button>
        <button type="button" className="btn ghost" disabled={busy} onClick={() => run('csv', 'containers')}>
          CSV containers
        </button>
        <button type="button" className="btn ghost" disabled={busy} onClick={() => run('csv', 'stacks')}>
          CSV stacks
        </button>
      </div>
      {err ? <p className="warn tiny">{err}</p> : null}
      {!compact ? (
        <p className="muted tiny">
          Structured inventory export (parity schema) — PowerShell script replacement.
        </p>
      ) : null}
    </div>
  )
}
