import type { SupplierLink, SupplierLinkSync } from '@/types'
import type { LinkQuote, SupplierLinkInputRow, SupplierLinksInput } from '../api/products'

/** One row of the editor's supplier-links table. `key` is a stable React key
 *  (the link id, or a temporary one for a row added in this edit). */
export interface LinkRow {
  key: string
  id?: string
  provider: number
  upstreamId: string
  /** Supplier product name as found when the row was added (search hit). */
  name?: string
  enabled: boolean
  active: boolean
  /** Supplier sync watches (cost + availability). */
  priceSync: boolean
  statusSync: boolean
  /** "loss" when the sync job paused this link. */
  pausedReason?: string
  /** The sync job's last check (read-only). */
  sync?: SupplierLinkSync
}

let tmpSeq = 0

/** Rows from the admin product read (order = the admin's priority order). */
export function toLinkRows(links: SupplierLink[] | undefined, activeId: string | undefined): LinkRow[] {
  return (links ?? []).map((l) => ({
    key: l.id,
    id: l.id,
    provider: l.provider,
    upstreamId: l.upstreamId,
    enabled: l.enabled,
    active: l.id === activeId,
    priceSync: l.priceSync ?? true,
    statusSync: l.statusSync ?? true,
    pausedReason: l.pausedReason || undefined,
    sync: l.sync,
  }))
}

/** Adds a supplier product; the first link of a product becomes active.
 *  Returns the rows unchanged when that supplier product is already linked. */
export function addLink(rows: LinkRow[], provider: number, upstreamId: string, name?: string): LinkRow[] {
  if (rows.some((r) => r.provider === provider && r.upstreamId === upstreamId)) return rows
  const row: LinkRow = {
    key: `new-${++tmpSeq}`,
    provider,
    upstreamId,
    name,
    enabled: true,
    active: rows.length === 0,
    priceSync: true,
    statusSync: true,
  }
  return [...rows, row]
}

/** Switches one of a row's sync watches. */
export function setWatch(rows: LinkRow[], index: number, watch: 'priceSync' | 'statusSync', on: boolean): LinkRow[] {
  return rows.map((r, i) => (i === index ? { ...r, [watch]: on } : r))
}

/** Moves a row up (-1) or down (+1) — the fallback priority order. */
export function moveLink(rows: LinkRow[], index: number, dir: -1 | 1): LinkRow[] {
  const to = index + dir
  if (index < 0 || index >= rows.length || to < 0 || to >= rows.length) return rows
  const next = rows.slice()
  ;[next[index], next[to]] = [next[to], next[index]]
  return next
}

/** Makes one row the active link (it is also enabled). */
export function makeActive(rows: LinkRow[], index: number): LinkRow[] {
  return rows.map((r, i) => ({ ...r, active: i === index, enabled: i === index ? true : r.enabled }))
}

/** Enables/disables a row. The active row can't be disabled. */
export function setEnabled(rows: LinkRow[], index: number, enabled: boolean): LinkRow[] {
  if (rows[index]?.active && !enabled) return rows
  return rows.map((r, i) => (i === index ? { ...r, enabled } : r))
}

/** Removes a row. Removing the active row passes "active" to the first
 *  remaining enabled row (or none, when no row is left). */
export function removeLink(rows: LinkRow[], index: number): LinkRow[] {
  const removed = rows[index]
  if (!removed) return rows
  const next = rows.filter((_, i) => i !== index)
  if (removed.active && next.length > 0) {
    const firstEnabled = next.findIndex((r) => r.enabled)
    return makeActive(next, firstEnabled >= 0 ? firstEnabled : 0)
  }
  return next
}

/** The save payload for the links table. */
export function toLinksInput(rows: LinkRow[], rev: number): SupplierLinksInput {
  const links: SupplierLinkInputRow[] = rows.map((r) => ({
    ...(r.id ? { id: r.id } : {}),
    provider: r.provider,
    upstreamId: r.upstreamId,
    enabled: r.enabled,
    active: r.active,
    priceSync: r.priceSync,
    statusSync: r.statusSync,
  }))
  return { links, rev }
}

/** Problems that would make the server refuse the table (shown before save). */
export function linkIssues(rows: LinkRow[], apiOn: boolean): string[] {
  const out: string[] = []
  if (apiOn && rows.length === 0) out.push('Link at least one supplier product, or untick "Fulfill via supplier API".')
  if (rows.length > 0 && rows.filter((r) => r.active).length !== 1) out.push('Choose exactly one active supplier.')
  return out
}

/** Supplier cost for one block of the product's selling unit:
 *  per-unit products sell `blockUnits` units for the variant price; a fixed
 *  package is one supplier unit. null when the price isn't known. */
export function blockCost(q: LinkQuote | undefined, blockUnits: number): number | null {
  if (!q || q.status !== 'ok' || q.price == null || !Number.isFinite(q.price)) return null
  return q.price * (blockUnits > 0 ? blockUnits : 1)
}

/** Margin of the selling price over the supplier cost, in percent. */
export function marginPct(sell: number, cost: number | null): number | null {
  if (cost == null || !(sell > 0)) return null
  return ((sell - cost) / sell) * 100
}

/** "$1.10" / "-$1,861.19" / "1,234 LBP" — an amount in the supplier's own
 *  currency (a credit account's balance is negative). */
export function fmtCost(v: number, currency?: string): string {
  if (currency && currency !== 'USD') return `${v.toLocaleString('en-US', { maximumFractionDigits: 2 })} ${currency}`
  if (v < 0) return `-${fmtCost(-v, currency)}`
  if (v > 0 && v < 0.01) return `$${v.toPrecision(2)}`
  return `$${v.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`
}
