import type { HTMLAttributes, ReactNode } from 'react'
import { cn } from '@/lib/utils'

export function Card({
  children,
  pad = true,
  hover = false,
  className,
  style,
  ...rest
}: { children: ReactNode; pad?: boolean; hover?: boolean } & HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn('card', pad && 'card-pad', hover && 'hover-pop', className)}
      style={style}
      {...rest}
    >
      {children}
    </div>
  )
}

export function Panel({
  children,
  pad = true,
  className,
  style,
  ...rest
}: { children: ReactNode; pad?: boolean } & HTMLAttributes<HTMLDivElement>) {
  return (
    <div className={cn('panel', pad && 'card-pad', className)} style={style} {...rest}>
      {children}
    </div>
  )
}
