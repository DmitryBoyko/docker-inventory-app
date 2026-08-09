import { useState } from 'react'
import { ApiError, downloadExport, type ExportFormat, type ExportScope } from '../api/client'
import { useT } from '../i18n'

type Props = {
  compact?: boolean
}

export function ExportButtons({ compact }: Props) {
  const t = useT()
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
          {t('export.json')}
        </button>
        <button type="button" className="btn ghost" disabled={busy} onClick={() => run('csv', 'containers')}>
          {t('export.csvContainers')}
        </button>
        <button type="button" className="btn ghost" disabled={busy} onClick={() => run('csv', 'stacks')}>
          {t('export.csvStacks')}
        </button>
      </div>
      {err ? <p className="warn tiny">{err}</p> : null}
      {!compact ? <p className="muted tiny">{t('export.hint')}</p> : null}
    </div>
  )
}
