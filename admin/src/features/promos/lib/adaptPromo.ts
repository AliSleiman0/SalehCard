import type { AdminPromo } from '../api/promos'

/** The flat shape the promo list renders. */
export interface PromoView {
  id: string
  code: string
  type: AdminPromo['type']
  value: number
  /** Display value, e.g. "10%", "$5", "$3 back". */
  valueLabel: string
  used: number
  limit: number
  unlimited: boolean
  /** Usage as a whole percentage (0 when unlimited). */
  usagePct: number
  status: AdminPromo['status']
  /** Human "valid period", e.g. "Jun 1 – Jun 30", "Until Jun 30", "Always". */
  period: string
  raw: AdminPromo
}

/** Short "Mon D" label for an ISO date, or '' when absent/unparseable. */
function shortDate(iso?: string): string {
  if (!iso) return ''
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' })
}

/** Build the valid-period label from optional start/expiry bounds. */
export function periodLabel(startsAt?: string, expiresAt?: string): string {
  const s = shortDate(startsAt)
  const e = shortDate(expiresAt)
  if (s && e) return `${s} – ${e}`
  if (s) return `From ${s}`
  if (e) return `Until ${e}`
  return 'Always'
}

/** Display label for a promo's value, keyed on its type. */
export function valueLabel(type: AdminPromo['type'], value: number): string {
  if (type === 'percent') return `${value}%`
  if (type === 'cashback') return `$${value} back`
  return `$${value}`
}

/** Map an admin promo to the flat view the list/table renders. */
export function adaptPromo(p: AdminPromo): PromoView {
  const unlimited = p.maxUses === 0
  const usagePct = unlimited ? 0 : Math.min(100, Math.round((p.uses / p.maxUses) * 100))
  return {
    id: p.id,
    code: p.code,
    type: p.type,
    value: p.value,
    valueLabel: valueLabel(p.type, p.value),
    used: p.uses,
    limit: p.maxUses,
    unlimited,
    usagePct,
    status: p.status,
    period: periodLabel(p.startsAt, p.expiresAt),
    raw: p,
  }
}
