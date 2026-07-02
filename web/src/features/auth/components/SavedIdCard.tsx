import { Icon } from '@/components'

// SavedIdCard renders a single saved player ID (a plain string). The backend
// stores saved IDs as bare strings, so there is no game/art/reorder affordance.
export function SavedIdCard({ value, onDelete }: { value: string; onDelete?: () => void }) {
  return (
    <div className="panel card-pad" style={{ padding: 14 }}>
      <div className="row between" style={{ gap: 10 }}>
        <span
          className="num"
          style={{ fontWeight: 700, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}
        >
          {value}
        </span>
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
