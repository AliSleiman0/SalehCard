import { Icon } from '@/components'
import type { SavedPlayerId } from '@/types'

// SavedIdCard renders a single saved player ID (label + value). The backend
// stores saved IDs as {label, value} objects.
export function SavedIdCard({ id, onDelete }: { id: SavedPlayerId; onDelete?: () => void }) {
  return (
    <div className="panel card-pad" style={{ padding: 14 }}>
      <div className="row between" style={{ gap: 10 }}>
        <div className="col" style={{ gap: 2, minWidth: 0 }}>
          <span
            style={{ fontWeight: 700, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}
          >
            {id.label}
          </span>
          <span
            className="num tiny faint"
            style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}
          >
            {id.value}
          </span>
        </div>
        {onDelete && (
          <button
            className="icon-btn"
            style={{ width: 32, height: 32, flex: 'none' }}
            onClick={onDelete}
            aria-label="delete"
          >
            <Icon name="trash" size={15} />
          </button>
        )}
      </div>
    </div>
  )
}
