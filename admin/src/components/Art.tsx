// Gradient box-art thumbnails — the design deliberately uses no real brand logos.
// Palette pairs ported from the prototype's data.js ART map.
export const ART: Record<string, [string, string]> = {
  pubg: ['#1a1140', '#d633ff'],
  tiktok: ['#0a1a2a', '#22e3c8'],
  bigo: ['#2a0f3a', '#ff5db1'],
  itunes: ['#1d1030', '#a06bff'],
  psn: ['#0a1838', '#3b5bff'],
  xbox: ['#06220f', '#2fd47a'],
  netflix: ['#280a0a', '#ff4d6d'],
  steam: ['#0c1322', '#66c0f4'],
  mlbb: ['#241046', '#ffb02e'],
  usdt: ['#062018', '#26a17b'],
  office: ['#2a0d0d', '#ff6a3d'],
  freefire: ['#2a1606', '#ff9b3d'],
  recharge: ['#101a2e', '#5b8bff'],
  spotify: ['#08220f', '#1db954'],
  visa: ['#0a1430', '#3b5bff'],
  soft: ['#22223a', '#3b5bff'],
}

interface ArtProps {
  art: string
  size?: number
  radius?: number
  label?: string
}

export function Art({ art, size = 34, radius = 8, label }: ArtProps) {
  const [a, b] = ART[art] || ['#22223a', '#3b5bff']
  return (
    <div
      style={{
        width: size,
        height: size,
        borderRadius: radius,
        flex: 'none',
        background: `radial-gradient(120% 120% at 25% 10%, ${b}, ${a})`,
        position: 'relative',
        overflow: 'hidden',
        display: 'grid',
        placeItems: 'center',
      }}
    >
      <div
        style={{
          position: 'absolute',
          inset: 0,
          background: 'linear-gradient(135deg, rgba(255,255,255,.18), transparent 45%)',
        }}
      />
      {label && (
        <span
          style={{
            position: 'relative',
            color: '#fff',
            fontWeight: 800,
            fontSize: size * 0.32,
            fontFamily: 'var(--font-display)',
          }}
        >
          {label}
        </span>
      )}
    </div>
  )
}

/** Pick a deterministic art key for a product/category when no explicit art is set. */
export function artForCategory(category: string): string {
  const keys = Object.keys(ART)
  let h = 0
  for (let i = 0; i < category.length; i++) h = (h * 31 + category.charCodeAt(i)) >>> 0
  return keys[h % keys.length]
}
