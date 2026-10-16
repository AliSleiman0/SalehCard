import type { InputField, Product } from '@/types'

export interface QtyBounds {
  min: number
  max: number
}

const DEFAULT_QTY: QtyBounds = { min: 1, max: 99 }

/** Purchase-quantity bounds from a product's legacy `quantity` input field.
 * Mirrors the API's validateQuantityField tolerances: a missing bound, the
 * known-corrupt {min:0,max:0} shape, or min > max is "unbounded" (1..99).
 * The minimum is clamped to at least 1. */
export function qtyBounds(product?: Pick<Product, 'inputFields'> | null): QtyBounds {
  const f = (product?.inputFields ?? []).find((x) => x.type === 'quantity')
  const c = f?.constraints
  if (!c || c.min == null || c.max == null) return DEFAULT_QTY
  if (c.min === 0 && c.max === 0) return DEFAULT_QTY
  if (c.min > c.max) return DEFAULT_QTY
  const min = Math.max(1, Math.ceil(c.min))
  const max = Math.max(min, Math.floor(c.max))
  return { min, max }
}

/** Clamp a quantity into its bounds. */
export function clampQty(qty: number, b: QtyBounds): number {
  return Math.min(b.max, Math.max(b.min, qty))
}

/** The amount-type input field that drives a unit-priced product's price, or
 * undefined when the product is not (validly) unit-priced. */
export function unitAmountField(product?: Pick<Product, 'inputFields' | 'unitPricing'> | null): InputField | undefined {
  const up = product?.unitPricing
  if (!up || !(up.blockUnits > 0)) return undefined
  return (product?.inputFields ?? []).find((f) => f.key === up.fieldKey && f.type === 'amount')
}

/** Price of one per-unit line: block price scaled to the typed amount, rounded
 * to the cent (mirrors the API's product.UnitLinePrice). Display-only — the
 * server re-prices authoritatively. */
export function unitLinePrice(blockPrice: number, amount: number, blockUnits: number): number {
  if (!(blockUnits > 0)) return 0
  return Math.round(((blockPrice * amount) / blockUnits) * 100) / 100
}

/** Map Arabic-Indic (٠-٩) and Persian (۰-۹) digits to ASCII 0-9, and the
 * Arabic decimal separator (٫) to ".". */
export function normalizeDigits(raw: string): string {
  return raw
    .replace(/[٠-٩]/g, (c) => String(c.charCodeAt(0) - 0x0660))
    .replace(/[۰-۹]/g, (c) => String(c.charCodeAt(0) - 0x06f0))
    .replace(/٫/g, '.')
}

/** Clean what the customer typed/pasted into an amount box: native digits
 * normalised, grouping separators (commas, spaces, Arabic ٬, apostrophes) and
 * other characters dropped, at most one decimal point kept (so a fraction can
 * be flagged instead of silently merged into the integer), integer part capped
 * at 15 digits. "45,000.00" stays "45000.00" (= 45000, see parseAmount). */
export function sanitizeAmountInput(raw: string): string {
  const s = normalizeDigits(raw).replace(/[^0-9.]/g, '')
  const dot = s.indexOf('.')
  if (dot < 0) return s.slice(0, 15)
  const int = s.slice(0, dot).slice(0, 15)
  const frac = s.slice(dot + 1).replace(/\./g, '')
  return `${int}.${frac}`
}

/** Whether a typed amount carries a non-zero fractional part (amounts are
 * whole units only; a trailing ".00" is fine). */
export function amountHasFraction(raw: string): boolean {
  const s = sanitizeAmountInput(raw)
  const dot = s.indexOf('.')
  return dot >= 0 && /[1-9]/.test(s.slice(dot + 1))
}

/** Parse a typed amount into whole units. Separators are ignored and an
 * all-zero decimal part is dropped ("45,000.00" → 45000); NaN when empty or
 * when the amount has a real fraction ("45.5"). */
export function parseAmount(raw: string): number {
  if (amountHasFraction(raw)) return NaN
  const int = sanitizeAmountInput(raw).split('.')[0]
  return int ? parseInt(int, 10) : NaN
}

/** The server rejects order lines below one cent. */
export const MIN_LINE_PRICE = 0.01

/** Whether a computed line total is under the server's minimum line price. */
export function lineBelowMinimum(total: number): boolean {
  return total < MIN_LINE_PRICE
}

/** Whether `amount` is a positive number inside [min, max] (max <= 0 = unbounded). */
export function amountInRange(amount: number, min: number, max: number): boolean {
  if (!Number.isFinite(amount) || amount <= 0) return false
  if (min > 0 && amount < min) return false
  if (max > 0 && amount > max) return false
  return true
}

/** Thousands-separated display, e.g. 45000 → "45,000". */
export function fmtAmount(n: number): string {
  return n.toLocaleString('en-US', { maximumFractionDigits: 2 })
}
