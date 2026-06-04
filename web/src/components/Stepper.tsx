import type { CSSProperties } from 'react'
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
      <Button variant="ghost" style={btn} onClick={() => set(Math.max(min, value - 1))}>
        <Icon name="minus" size={16} />
      </Button>
      <span
        className="num"
        style={{ minWidth: 26, textAlign: 'center', fontWeight: 700, fontSize: 17 }}
      >
        {value}
      </span>
      <Button variant="ghost" style={btn} onClick={() => set(Math.min(max, value + 1))}>
        <Icon name="plus" size={16} />
      </Button>
    </div>
  )
}
