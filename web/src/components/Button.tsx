import type { ButtonHTMLAttributes } from 'react'
import { cn } from '@/lib/utils'

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: 'primary' | 'cyan' | 'gold' | 'ghost' | 'outline' | 'danger'
  size?: 'sm' | 'md' | 'lg'
  block?: boolean
  loading?: boolean
}

export function Button({
  variant = 'primary',
  size = 'md',
  block = false,
  loading = false,
  className,
  children,
  disabled,
  ...rest
}: ButtonProps) {
  return (
    <button
      className={cn(
        'btn',
        variant && 'btn-' + variant,
        size === 'sm' && 'btn-sm',
        size === 'lg' && 'btn-lg',
        block && 'btn-block',
        className,
      )}
      disabled={disabled || loading}
      {...rest}
    >
      {loading && <span className="btn-spinner" aria-hidden="true" />}
      {children}
    </button>
  )
}
