import type { InputHTMLAttributes } from 'react'
import { cn } from '@/lib/utils'

export interface InputProps extends InputHTMLAttributes<HTMLInputElement> {
  label?: string
  error?: string
}

export function Input({ label, error, className, ...rest }: InputProps) {
  return (
    <div className="col" style={{ gap: 6 }}>
      {label && <label className="label">{label}</label>}
      <input className={cn('field', className)} {...rest} />
      {error && (
        <span className="tiny" style={{ color: 'var(--danger)', fontWeight: 600 }}>
          {error}
        </span>
      )}
    </div>
  )
}
