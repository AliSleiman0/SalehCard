// Small presentational primitives ported from the prototype: Stars, PayChip, Avatar.

export function Stars({ n, size = 14 }: { n: number; size?: number }) {
  return (
    <div style={{ display: 'inline-flex', gap: 1, color: 'var(--warn)' }}>
      {[1, 2, 3, 4, 5].map((i) => (
        <svg
          key={i}
          width={size}
          height={size}
          viewBox="0 0 24 24"
          fill={i <= n ? 'currentColor' : 'none'}
          stroke="currentColor"
          strokeWidth="2"
        >
          <path d="M12 3l2.6 5.6 6.1.7-4.5 4.2 1.2 6L12 16.8 6.6 19.5l1.2-6L3.3 9.3l6.1-.7z" />
        </svg>
      ))}
    </div>
  )
}

const PAY_MAP: Record<string, [string, string]> = {
  wallet: ['#8a3bff', 'Wallet'],
  visa: ['#3b5bff', 'Visa'],
  usdt: ['#22e3c8', 'USDT'],
}

export function PayChip({ p }: { p: string }) {
  const [c, label] = PAY_MAP[p] || ['#888', p]
  return (
    <span style={{ display: 'inline-flex', alignItems: 'center', gap: 6, fontWeight: 700, fontSize: 12.5 }}>
      <span style={{ width: 8, height: 8, borderRadius: 3, background: c, flex: 'none' }} />
      {label}
    </span>
  )
}

const AVA_COLORS = ['#3b5bff', '#8a3bff', '#d633ff', '#22e3c8', '#2fd47a', '#ffb02e', '#ff4d6d']

export function Avatar({ name, color }: { name: string; color?: string }) {
  const c = color || AVA_COLORS[(name.charCodeAt(0) + name.length) % AVA_COLORS.length]
  const initials = name
    .split(' ')
    .map((w) => w[0])
    .slice(0, 2)
    .join('')
  return (
    <div className="ava-sm" style={{ background: c }}>
      {initials}
    </div>
  )
}
