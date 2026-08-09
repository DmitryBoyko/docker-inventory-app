import { Link } from 'react-router-dom'

type Props = {
  names: string[]
  /** Build href for one entity name (e.g. container or stack). */
  to: (name: string) => string
  /** How many names to show before "+N". */
  preview?: number
  empty?: string
}

/**
 * Compact table cell for long name lists: count + short preview, full list in title.
 */
export function EntityListCell({ names, to, preview = 2, empty = '0' }: Props) {
  const list = names.filter(Boolean)
  const count = list.length
  if (count === 0) {
    return <span className="muted">{empty}</span>
  }

  const shown = list.slice(0, preview)
  const rest = count - shown.length
  const full = list.join(', ')

  return (
    <div className="entity-list-cell" title={full}>
      <span className="entity-list-count mono">{count}</span>
      <span className="entity-list-preview">
        {shown.map((name, i) => (
          <span key={name}>
            {i > 0 ? ', ' : ''}
            <Link
              className="text-link"
              to={to(name)}
              onClick={(e) => e.stopPropagation()}
            >
              {name}
            </Link>
          </span>
        ))}
        {rest > 0 ? <span className="muted"> +{rest}</span> : null}
      </span>
    </div>
  )
}
