import { useRef } from 'react'

// OtpInput renders 6 display boxes backed by one hidden numeric input, so it gets
// autofill / one-time-code support for free while staying simple and accessible.
export function OtpInput({
  value,
  onChange,
  hasError,
  autoFocus = true,
}: {
  value: string
  onChange: (code: string) => void
  hasError?: boolean
  autoFocus?: boolean
}) {
  const ref = useRef<HTMLInputElement>(null)
  const digits = value.slice(0, 6).split('')

  return (
    <div
      className="row"
      style={{ gap: 8, justifyContent: 'center', position: 'relative', cursor: 'text' }}
      onClick={() => ref.current?.focus()}
    >
      {Array.from({ length: 6 }).map((_, i) => (
        <div
          key={i}
          className="field"
          style={{
            width: 44,
            height: 52,
            display: 'grid',
            placeItems: 'center',
            fontSize: 22,
            fontWeight: 800,
            fontFamily: 'var(--font-mono, monospace)',
            borderColor: hasError
              ? 'var(--danger)'
              : i === digits.length
                ? 'var(--brand-1)'
                : undefined,
          }}
        >
          {digits[i] ?? ''}
        </div>
      ))}
      <input
        ref={ref}
        value={value}
        onChange={(e) => onChange(e.target.value.replace(/\D/g, '').slice(0, 6))}
        inputMode="numeric"
        autoComplete="one-time-code"
        autoFocus={autoFocus}
        maxLength={6}
        aria-label="Verification code"
        style={{
          position: 'absolute',
          inset: 0,
          opacity: 0,
          width: '100%',
          height: '100%',
          border: 0,
          cursor: 'text',
        }}
      />
    </div>
  )
}
