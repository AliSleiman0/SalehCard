import type { AdminOffer } from '../api/offers'

/** The flat shape the offer list renders. */
export interface OfferView {
  id: string
  productId: string
  productName: string
  category: string
  discountType: AdminOffer['discountType']
  discountValue: number
  /** Display value, e.g. "25%" or "−$3". */
  valueLabel: string
  originalPrice: number
  offerPrice: number
  /** "$10 → $7" was/now label. */
  priceLabel: string
  status: AdminOffer['status']
  /** Human "valid period", e.g. "Jun 1 – Jun 30", "Until Jun 30", "Always". */
  period: string
  sortOrder: number
  raw: AdminOffer
}

/** Short "Mon D" label for an ISO date, or '' when absent/unparseable. */
function shortDate(iso?: string): string {
  if (!iso) return ''
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' })
}

/** Build the valid-period label from optional start/end bounds. */
export function periodLabel(startsAt?: string, endsAt?: string): string {
  const s = shortDate(startsAt)
  const e = shortDate(endsAt)
  if (s && e) return `${s} – ${e}`
  if (s) return `From ${s}`
  if (e) return `Until ${e}`
  return 'Always'
}

/** Display label for an offer's discount, keyed on its type. */
export function valueLabel(type: AdminOffer['discountType'], value: number): string {
  return type === 'percent' ? `${value}%` : `−$${value}`
}

/** "$10 → $7" was/now label (omits the arrow when prices are unknown). */
export function priceLabel(original: number, offer: number): string {
  if (original <= 0) return '—'
  return `$${original.toFixed(2)} → $${offer.toFixed(2)}`
}

/** Map an admin offer to the flat view the list/table renders. */
export function adaptOffer(o: AdminOffer): OfferView {
  return {
    id: o.id,
    productId: o.productId,
    productName: o.product?.title.en || o.productId,
    category: o.product?.category || '',
    discountType: o.discountType,
    discountValue: o.discountValue,
    valueLabel: valueLabel(o.discountType, o.discountValue),
    originalPrice: o.originalFromPrice,
    offerPrice: o.offerFromPrice,
    priceLabel: priceLabel(o.originalFromPrice, o.offerFromPrice),
    status: o.status,
    period: periodLabel(o.startsAt, o.endsAt),
    sortOrder: o.sortOrder,
    raw: o,
  }
}
