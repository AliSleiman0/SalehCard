import { useId } from 'react'

interface SparkProps {
  data: number[]
  color?: string
  w?: number
  h?: number
  fill?: boolean
}

export function Spark({ data, color = 'var(--brand-1)', w = 120, h = 34, fill = true }: SparkProps) {
  const gid = 'sg' + useId().replace(/:/g, '')
  const max = Math.max(...data)
  const min = Math.min(...data)
  const rng = max - min || 1
  const pts = data.map((v, i) => [(i / (data.length - 1)) * w, h - 3 - ((v - min) / rng) * (h - 6)])
  const line = pts.map((p, i) => (i ? 'L' : 'M') + p[0].toFixed(1) + ' ' + p[1].toFixed(1)).join(' ')
  const area = line + ` L${w} ${h} L0 ${h} Z`
  return (
    <svg className="spark" viewBox={`0 0 ${w} ${h}`} preserveAspectRatio="none">
      <defs>
        <linearGradient id={gid} x1="0" y1="0" x2="0" y2="1">
          <stop offset="0" stopColor={color} stopOpacity="0.28" />
          <stop offset="1" stopColor={color} stopOpacity="0" />
        </linearGradient>
      </defs>
      {fill && <path d={area} fill={`url(#${gid})`} />}
      <path d={line} fill="none" stroke={color} strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  )
}

interface AreaChartProps {
  data: number[]
  labels: string[]
  height?: number
}

export function AreaChart({ data, labels, height = 220 }: AreaChartProps) {
  const fillId = 'revfill' + useId().replace(/:/g, '')
  const lineId = 'revline' + useId().replace(/:/g, '')
  const w = 760
  const h = height
  const pl = 8
  const pr = 8
  const pt = 16
  const pb = 26
  const iw = w - pl - pr
  const ih = h - pt - pb
  // With fewer than two points the path math below dereferences pts[0]/divides
  // by (len-1); render an empty chart frame instead (e.g. while data loads).
  if (data.length < 2) {
    return (
      <svg className="chart-area" viewBox={`0 0 ${w} ${h}`} preserveAspectRatio="none" style={{ height }} />
    )
  }
  const max = Math.max(...data) * 1.12
  const min = 0
  const rng = max - min || 1
  const x = (i: number) => pl + (i / (data.length - 1)) * iw
  const y = (v: number) => pt + ih - ((v - min) / rng) * ih
  const pts = data.map((v, i) => [x(i), y(v)])
  let line = `M${pts[0][0]} ${pts[0][1]}`
  for (let i = 1; i < pts.length; i++) {
    const [x0, y0] = pts[i - 1]
    const [x1, y1] = pts[i]
    const cx = (x0 + x1) / 2
    line += ` C${cx} ${y0} ${cx} ${y1} ${x1} ${y1}`
  }
  const area = line + ` L${pts[pts.length - 1][0]} ${pt + ih} L${pts[0][0]} ${pt + ih} Z`
  const grid = [0, 0.25, 0.5, 0.75, 1].map((g) => pt + ih - g * ih)
  return (
    <svg className="chart-area" viewBox={`0 0 ${w} ${h}`} preserveAspectRatio="none" style={{ height }}>
      <defs>
        <linearGradient id={fillId} x1="0" y1="0" x2="0" y2="1">
          <stop offset="0" stopColor="#8a3bff" stopOpacity="0.34" />
          <stop offset="1" stopColor="#3b5bff" stopOpacity="0" />
        </linearGradient>
        <linearGradient id={lineId} x1="0" y1="0" x2="1" y2="0">
          <stop offset="0" stopColor="#3b5bff" />
          <stop offset="0.5" stopColor="#8a3bff" />
          <stop offset="1" stopColor="#d633ff" />
        </linearGradient>
      </defs>
      {grid.map((gy, i) => (
        <line key={i} x1={pl} y1={gy} x2={w - pr} y2={gy} stroke="var(--border)" strokeWidth="1" strokeDasharray="3 5" />
      ))}
      <path d={area} fill={`url(#${fillId})`} />
      <path d={line} fill="none" stroke={`url(#${lineId})`} strokeWidth="2.6" strokeLinecap="round" strokeLinejoin="round" />
      {pts.map((p, i) =>
        labels[i] ? <circle key={i} cx={p[0]} cy={p[1]} r="3.5" fill="#d633ff" stroke="var(--card)" strokeWidth="2" /> : null
      )}
      {labels.map((l, i) =>
        l ? (
          <text key={i} x={x(i)} y={h - 7} fontSize="11" fontWeight="700" fill="var(--text-faint)" textAnchor="middle">
            {l}
          </text>
        ) : null
      )}
    </svg>
  )
}

export interface DonutDatum {
  label: string
  value: number
  color: string
  key?: string
}

interface DonutProps {
  data: DonutDatum[]
  size?: number
  thickness?: number
  center?: React.ReactNode
}

export function Donut({ data, size = 150, thickness = 22, center }: DonutProps) {
  const r = (size - thickness) / 2
  const C = 2 * Math.PI * r
  let off = 0
  const total = data.reduce((s, d) => s + d.value, 0) || 1
  return (
    <div style={{ position: 'relative', width: size, height: size, flex: 'none' }}>
      <svg width={size} height={size} viewBox={`0 0 ${size} ${size}`} style={{ transform: 'rotate(-90deg)' }}>
        <circle cx={size / 2} cy={size / 2} r={r} fill="none" stroke="var(--surface-3)" strokeWidth={thickness} />
        {data.map((d, i) => {
          const len = (d.value / total) * C
          const seg = (
            <circle
              key={i}
              cx={size / 2}
              cy={size / 2}
              r={r}
              fill="none"
              strokeWidth={thickness}
              strokeDasharray={`${len} ${C - len}`}
              strokeDashoffset={-off}
              strokeLinecap="butt"
              // via style so CSS custom properties (var(--ff-*)) resolve — they
              // do not when set as the SVG `stroke` presentation attribute.
              style={{ stroke: d.color }}
            />
          )
          off += len
          return seg
        })}
      </svg>
      {center && (
        <div style={{ position: 'absolute', inset: 0, display: 'grid', placeItems: 'center', textAlign: 'center' }}>
          {center}
        </div>
      )}
    </div>
  )
}

export interface BarDatum {
  label: string
  value: number
  color?: string
}

export function Bars({ data, max }: { data: BarDatum[]; max?: number }) {
  const mx = max || Math.max(...data.map((d) => d.value))
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 13 }}>
      {data.map((d, i) => (
        <div key={i}>
          <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 6, fontSize: 12.5, fontWeight: 700 }}>
            <span style={{ color: 'var(--text-dim)' }}>{d.label}</span>
            <span className="num">{d.value}%</span>
          </div>
          <div className="meter">
            <i style={{ width: (d.value / mx) * 100 + '%', background: d.color || 'var(--grad)' }} />
          </div>
        </div>
      ))}
    </div>
  )
}
