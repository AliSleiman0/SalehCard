import type { CatalogProductView } from '../api/suppliers'

/** Human summary of the quantities a supplier product accepts ("" when any). */
export function qtyHint(p: Pick<CatalogProductView, 'qtyMin' | 'qtyMax' | 'qtyValues'>): string {
  if (p.qtyValues && p.qtyValues.length > 0) return `qty ${p.qtyValues.join(' / ')}`
  if (p.qtyMin != null && p.qtyMax != null) return `qty ${fmtN(p.qtyMin)}–${fmtN(p.qtyMax)}`
  if (p.qtyMin != null) return `qty ≥ ${fmtN(p.qtyMin)}`
  if (p.qtyMax != null) return `qty ≤ ${fmtN(p.qtyMax)}`
  return ''
}

function fmtN(n: number): string {
  return n.toLocaleString('en-US')
}

/** Whether a supplier product accepts `qty` (true when it states no limits). */
export function acceptsQty(p: Pick<CatalogProductView, 'qtyMin' | 'qtyMax' | 'qtyValues'>, qty: number): boolean {
  if (p.qtyValues && p.qtyValues.length > 0) return p.qtyValues.some((v) => Number(v) === qty)
  if (p.qtyMin != null && qty < p.qtyMin) return false
  if (p.qtyMax != null && qty > p.qtyMax) return false
  return true
}

export interface PickContext {
  /** The supplier product id currently entered/selected. */
  value: string
  /** The loaded catalog, or undefined while it isn't available (loading, error, no access). */
  catalog?: CatalogProductView[]
  /** The quantity this order will send (order page). */
  qty?: number
  /** The amount range customers may type (product editor, per-unit products). */
  amountRange?: { min?: number; max?: number }
  /** Parameter names this order sends (its field keys, plus playerId when set). */
  sends?: string[]
}

/** Warnings about a chosen supplier product — the mistakes behind the
 *  2026-11-13 prod rejections ("Product Not Found", "Qty is Small"). */
export function pickWarnings(ctx: PickContext): string[] {
  const id = ctx.value.trim()
  if (!id || !ctx.catalog) return []
  const p = ctx.catalog.find((c) => c.upstreamId === id)
  if (!p) return [`"${id}" is not in this supplier's catalog — the supplier will refuse it (Product Not Found).`]
  const out: string[] = []
  if (!p.available) out.push('The supplier marks this product unavailable right now.')
  const hint = qtyHint(p)
  if (ctx.qty != null && !acceptsQty(p, ctx.qty)) {
    out.push(`This order sends quantity ${fmtN(ctx.qty)}, but the supplier accepts ${hint.replace(/^qty /, '')}.`)
  }
  if (ctx.sends) {
    const have = new Set(ctx.sends.map((k) => k.toLowerCase()))
    const missing = (p.params ?? []).filter((k) => !have.has(k.toLowerCase()))
    if (missing.length > 0) {
      out.push(
        `The supplier asks for ${missing.map((k) => `"${k}"`).join(', ')}, which this order doesn't carry by that name — check the supplier won't refuse it.`,
      )
    }
  }
  const r = ctx.amountRange
  if (r && hint) {
    const outOfRange = [r.min, r.max].some((v) => v != null && !acceptsQty(p, v))
    if (outOfRange) {
      out.push(
        `Customers may type ${r.min != null ? fmtN(r.min) : '…'}–${r.max != null ? fmtN(r.max) : '…'}, but the supplier accepts ${hint.replace(/^qty /, '')} — narrow the amount field's min/max.`,
      )
    }
  }
  return out
}

/** Supplier cost estimate for `qty` units at the catalog price (null when unknown). */
export function estimateCost(p: Pick<CatalogProductView, 'price'> | undefined, qty: number): number | null {
  if (!p || !Number.isFinite(p.price) || p.price <= 0) return null
  return Math.round(p.price * qty * 100) / 100
}
