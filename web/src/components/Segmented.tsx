import { cn } from '@/lib/utils'

export function Segmented<T extends string>({
  options,
  value,
  onChange,
  plain = false,
}: {
  options: { value: T; label: string }[]
  value: T
  onChange: (v: T) => void
  plain?: boolean
}) {
  return (
    <div className={cn('seg', plain && 'plain')}>
      {options.map((o) => (
        <button
          key={o.value}
          className={o.value === value ? 'on' : ''}
          onClick={() => onChange(o.value)}
        >
          {o.label}
        </button>
      ))}
    </div>
  )
}
