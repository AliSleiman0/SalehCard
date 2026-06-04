import type { CSSProperties } from 'react'
import { cn } from '@/lib/utils'

export function Skeleton({
  w = '100%',
  h = 14,
  r = 8,
  className,
  style,
}: {
  w?: number | string
  h?: number
  r?: number
  className?: string
  style?: CSSProperties
}) {
  return (
    <div
      className={cn('skel', className)}
      style={{ width: w, height: h, borderRadius: r, ...style }}
    />
  )
}
