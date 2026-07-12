import { useState, type CSSProperties } from 'react'
import { ART } from '@/lib/art'

export function ImageArt({
  art = 'soft',
  src,
  word = '',
  sub = '',
  h = 160,
  wordSize,
  radius,
  style,
  className,
}: {
  art?: string
  /** real image URL; the gradient art renders behind it as loading state and error fallback */
  src?: string
  word?: string
  sub?: string
  h?: number
  wordSize?: number
  radius?: number
  style?: CSSProperties
  className?: string
}) {
  const [broken, setBroken] = useState(false)
  const [a, b] = ART[art] || ART.soft
  const showImg = !!src && !broken
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
      {showImg && (
        <img
          src={src}
          alt={word}
          loading="lazy"
          onError={() => setBroken(true)}
          style={{
            position: 'absolute',
            inset: 0,
            zIndex: 2,
            width: '100%',
            height: '100%',
            objectFit: 'cover',
            borderRadius: radius,
          }}
        />
      )}
      {!showImg && (
        <div className="art-label">
          <div className="art-word" style={{ fontSize: ws, letterSpacing: '-.01em' }}>
            {word}
          </div>
          {sub && <div className="art-sub">{sub}</div>}
        </div>
      )}
    </div>
  )
}
