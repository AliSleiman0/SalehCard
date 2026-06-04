import type { CSSProperties } from 'react'
import { ART } from '@/lib/art'

export function ImageArt({
  art = 'soft',
  word = '',
  sub = '',
  h = 160,
  wordSize,
  radius,
  style,
  className,
}: {
  art?: string
  word?: string
  sub?: string
  h?: number
  wordSize?: number
  radius?: number
  style?: CSSProperties
  className?: string
}) {
  const [a, b] = ART[art] || ART.soft
  const ws = wordSize || Math.max(20, Math.min(40, 220 / Math.max(word.length, 4)))
  return (
    <div
      className={className ? `art ${className}` : 'art'}
      style={{
        height: h,
        width: '100%',
        borderRadius: radius,
        background: `radial-gradient(120% 120% at 20% 0%, ${b}, ${a})`,
        ...style,
      }}
    >
      {/* subtle pattern */}
      <div
        style={{
          position: 'absolute',
          inset: 0,
          opacity: 0.18,
          zIndex: 1,
          background: 'repeating-linear-gradient(135deg, #fff 0 2px, transparent 2px 16px)',
        }}
      />
      <div className="art-label">
        <div className="art-word" style={{ fontSize: ws, letterSpacing: '-.01em' }}>
          {word}
        </div>
        {sub && <div className="art-sub">{sub}</div>}
      </div>
    </div>
  )
}
