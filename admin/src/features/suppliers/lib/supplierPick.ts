import i18n from '@/i18n/config'
import { quoted } from '@/lib/quoted'
import type { CatalogProductView } from '../api/suppliers'

/** Human summary of the quantities a supplier product accepts ("" when any). */
export function qtyHint(p: Pick<CatalogProductView, 'qtyMin' | 'qtyMax' | 'qtyValues'>): string {
  const range = qtyRange(p)
  return range ? i18n.t('pick_qty', { range }) : ''
}

function qtyRange(p: Pick<CatalogProductView, 'qtyMin' | 'qtyMax' | 'qtyValues'>): string {
  if (p.qtyValues && p.qtyValues.length > 0) return p.qtyValues.join(' / ')
  if (p.qtyMin != null && p.qtyMax != null) return `${fmtN(p.qtyMin)}–${fmtN(p.qtyMax)}`
  if (p.qtyMin != null) return `≥ ${fmtN(p.qtyMin)}`
  if (p.qtyMax != null) return `≤ ${fmtN(p.qtyMax)}`
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
  if (!p) return [i18n.t('pick_warn_not_found', { id })]
  const out: string[] = []
  if (!p.available) out.push(i18n.t('pick_warn_unavailable'))
  const range = qtyRange(p)
  if (ctx.qty != null && !acceptsQty(p, ctx.qty)) {
    out.push(i18n.t('pick_warn_qty', { n: fmtN(ctx.qty), range }))
  }
  if (ctx.sends) {
    // Imported products store a param's key slugified ("Player ID" → player_id)
    // and the adapter adds playerId: compare on letters+digits only. A prompt
    // with no ASCII letters/digits (e.g. Arabic) is stored as fieldN — it can't
    // be matched by name, so it is never flagged.
    const have = new Set(ctx.sends.map(paramName))
    const missing = (p.params ?? []).filter((k) => paramName(k) !== '' && !have.has(paramName(k)))
    if (missing.length > 0) {
      out.push(i18n.t('pick_warn_params', { names: quoted(missing) }))
    }
  }
  const r = ctx.amountRange
  if (r && range) {
    const outOfRange = [r.min, r.max].some((v) => v != null && !acceptsQty(p, v))
    if (outOfRange) {
      const min = r.min != null ? fmtN(r.min) : '…'
      const max = r.max != null ? fmtN(r.max) : '…'
      out.push(i18n.t('pick_warn_amount', { min, max, range }))
    }
  }
  return out
}

/** paramName normalizes a supplier param / field key for comparison:
 *  lowercase ASCII letters and digits only ("Player ID", "player_id" and
 *  "playerId" all become "playerid"). */
export function paramName(s: string): string {
  return s.toLowerCase().replace(/[^a-z0-9]/g, '')
}

/** Supplier cost estimate for `qty` units at the catalog price (null when unknown). */
export function estimateCost(p: Pick<CatalogProductView, 'price'> | undefined, qty: number): number | null {
  if (!p || !Number.isFinite(p.price) || p.price <= 0) return null
  return Math.round(p.price * qty * 100) / 100
}
