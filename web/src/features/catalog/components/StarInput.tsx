import { useState } from 'react'

// StarInput is an interactive 1–5 star picker (the DS `Stars` is display-only).
export function StarInput({
  value,
  onChange,
  size = 34,
}: {
  value: number
  onChange: (v: number) => void
  size?: number
}) {
  const [hover, setHover] = useState(0)
  const shown = hover || value
  return (
    <div className="row" style={{ gap: 6 }} onMouseLeave={() => setHover(0)}>
      {[1, 2, 3, 4, 5].map((i) => (
        <button
          key={i}
          type="button"
          aria-label={`${i} star${i > 1 ? 's' : ''}`}
          onMouseEnter={() => setHover(i)}
          onClick={() => onChange(i)}
          style={{ background: 'none', border: 0, padding: 2, cursor: 'pointer', lineHeight: 0 }}
        >
          <svg
            width={size}
            height={size}
            viewBox="0 0 24 24"
            fill={i <= shown ? '#ffb02e' : 'var(--border-strong)'}
          >
            <path d="M12 3l2.6 5.6 6.1.7-4.5 4.2 1.2 6L12 16.8 6.6 19.5l1.2-6L3.3 9.3l6.1-.7z" />
          </svg>
        </button>
      ))}
    </div>
  )
}
