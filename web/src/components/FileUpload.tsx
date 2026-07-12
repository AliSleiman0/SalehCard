import { useRef } from 'react'
import { Icon } from './Icon'

// FileUpload is a controlled, presentational photo picker (no fetch inside — the
// feature hook owns the upload). Shows a dashed drop tile until a URL is set,
// then a preview with replace/remove.
export function FileUpload({
  label,
  value,
  uploading = false,
  error,
  accept = 'image/*',
  onSelect,
  onClear,
}: {
  label: string
  value?: string | null
  uploading?: boolean
  error?: string
  accept?: string
  onSelect: (file: File) => void
  onClear?: () => void
}) {
  const ref = useRef<HTMLInputElement>(null)

  const pick = () => ref.current?.click()
  const onChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const f = e.target.files?.[0]
    if (f) onSelect(f)
    e.target.value = '' // allow re-picking the same file
  }

  return (
    <div className="col" style={{ gap: 6 }}>
      <label className="label" style={{ margin: 0 }}>
        {label}
      </label>
      <input ref={ref} type="file" accept={accept} onChange={onChange} style={{ display: 'none' }} />
      {value ? (
        <div className="panel" style={{ padding: 10, position: 'relative' }}>
          <img
            src={value}
            alt={label}
            style={{ width: '100%', maxHeight: 200, objectFit: 'contain', borderRadius: 8 }}
          />
          <div className="row" style={{ gap: 8, marginTop: 10 }}>
            <button className="btn btn-ghost btn-sm" type="button" onClick={pick} disabled={uploading}>
              {uploading ? '…' : 'Replace'}
            </button>
            {onClear && (
              <button className="btn btn-ghost btn-sm" type="button" onClick={onClear} disabled={uploading}>
                <Icon name="trash" size={14} />
              </button>
            )}
          </div>
        </div>
      ) : (
        <button
          type="button"
          className="panel card-pad clickable"
          onClick={pick}
          disabled={uploading}
          style={{
            border: '1.5px dashed var(--border-strong)',
            display: 'grid',
            placeItems: 'center',
            minHeight: 120,
            background: 'transparent',
            width: '100%',
          }}
        >
          <div className="col center" style={{ gap: 8, color: 'var(--text-dim)' }}>
            <Icon name="plusc" size={28} />
            <span style={{ fontWeight: 700 }}>{uploading ? 'Uploading…' : label}</span>
          </div>
        </button>
      )}
      {error && (
        <span className="tiny" style={{ color: 'var(--danger)', fontWeight: 600 }}>
          {error}
        </span>
      )}
    </div>
  )
}
