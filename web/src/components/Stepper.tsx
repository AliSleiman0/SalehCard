import { useState, type CSSProperties } from 'react'
import { Icon } from './Icon'
import { Button } from './Button'

export function Stepper({
  value,
  set,
  min = 1,
  max = 99,
}: {
  value: number
  set: (n: number) => void
  min?: number
  max?: number
}) {
  // While editing, `draft` holds the raw text (so the field can be transiently
  // empty as the user types). `null` means "not editing" → the input mirrors the
  // `value` prop, keeping it in sync with the +/- buttons.
  const [draft, setDraft] = useState<string | null>(null)

  const clamp = (n: number) => Math.min(max, Math.max(min, n))

  // Commit the typed value. Stepper is the single clamp point — callers store
  // whatever `set` receives verbatim — so clamp here; revert empty/NaN to `value`.
  const commit = (raw: string) => {
    const n = parseInt(raw, 10)
    if (!Number.isNaN(n)) set(clamp(n))
    setDraft(null)
  }

  const btn: CSSProperties = {
    width: 40,
    height: 40,
    display: 'grid',
    placeItems: 'center',
    borderRadius: 12,
  }
  return (
    <div
      className="row"
      style={{
        gap: 8,
        background: 'var(--surface-2)',
        padding: 4,
        borderRadius: 16,
        border: '1px solid var(--border)',
      }}
    >
      <Button variant="ghost" style={btn} onClick={() => set(clamp(value - 1))}>
        <Icon name="minus" size={16} />
      </Button>
      <input
        className="num"
        inputMode="numeric"
        aria-label="Quantity"
        value={draft ?? String(value)}
        onChange={(e) => setDraft(e.target.value.replace(/\D/g, ''))}
        onFocus={(e) => e.currentTarget.select()}
        onBlur={(e) => commit(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === 'Enter') e.currentTarget.blur()
        }}
        style={{
          width: 40,
          padding: 0,
          textAlign: 'center',
          fontWeight: 700,
          fontSize: 17,
          border: 0,
          background: 'transparent',
          outline: 'none',
          color: 'var(--text)',
          fontFamily: 'inherit',
        }}
      />
      <Button variant="ghost" style={btn} onClick={() => set(clamp(value + 1))}>
        <Icon name="plus" size={16} />
      </Button>
    </div>
  )
}
