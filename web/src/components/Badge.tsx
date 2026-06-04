import type { CSSProperties, ReactNode } from 'react'
import { cn } from '@/lib/utils'

export function Badge({
  variant = 'soft',
  children,
  className,
  style,
}: {
  variant?: 'instant' | 'secure' | 'agent' | 'disc' | 'soft'
  children: ReactNode
  className?: string
  style?: CSSProperties
}) {
  return (
    <span className={cn('badge', 'badge-' + variant, className)} style={style}>
      {children}
    </span>
  )
}
