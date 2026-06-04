import { cn } from '@/lib/utils'

/** Toggle switch (`.tog`). */
export function Toggle({ on, onClick }: { on: boolean; onClick?: () => void }) {
  return <div className={cn('tog', on && 'on')} onClick={onClick} role="switch" aria-checked={on} />
}

export interface SegItem<T extends string> {
  k: T
  label: React.ReactNode
}

/** Admin segmented control (`.aseg`). */
export function Segmented<T extends string>({
  items,
  value,
  onChange,
  block,
}: {
  items: SegItem<T>[]
  value: T
  onChange: (v: T) => void
  block?: boolean
}) {
  return (
    <div className="aseg" style={block ? { width: '100%' } : undefined}>
      {items.map((it) => (
        <button
          key={it.k}
          className={value === it.k ? 'on' : ''}
          style={block ? { flex: 1 } : undefined}
          onClick={() => onChange(it.k)}
        >
          {it.label}
        </button>
      ))}
    </div>
  )
}

/** Detail-page tabs (`.atabs`). */
export function Tabs<T extends string>({
  items,
  value,
  onChange,
}: {
  items: { k: T; label: React.ReactNode }[]
  value: T
  onChange: (v: T) => void
}) {
  return (
    <div className="atabs">
      {items.map((it) => (
        <button key={it.k} className={value === it.k ? 'on' : ''} onClick={() => onChange(it.k)}>
          {it.label}
        </button>
      ))}
    </div>
  )
}

/** Filter chip (`.chip`). */
export function Chip({
  on,
  onClick,
  children,
  dotColor,
  style,
}: {
  on?: boolean
  onClick?: () => void
  children: React.ReactNode
  dotColor?: string
  style?: React.CSSProperties
}) {
  return (
    <div className={cn('chip', on && 'on')} onClick={onClick} style={style}>
      {dotColor && (
        <i style={{ width: 7, height: 7, borderRadius: 99, background: dotColor, flex: 'none' }} />
      )}
      {children}
    </div>
  )
}
