import { userName, joinedLabel } from '@/features/users/lib/adaptUser'
import type { Variant } from '@/types'
import type { AdminReseller } from '../api/resellers'

/** Effective reseller price for a variant: the lowest of retail, the
 *  tier-margin price, the global per-variant override, and this reseller's
 *  custom price. Mirrors the backend rule (product.ResellerUnitPrice) — keep
 *  the two in lockstep so the table always predicts what checkout charges. */
export function effectivePrice(v: Variant, margin: number, custom?: number): number {
  let p = v.price
  if (margin > 0 && margin < 100) p = Math.min(p, v.price * (1 - margin / 100))
  if (v.resellerPrice != null) p = Math.min(p, v.resellerPrice)
  if (custom != null) p = Math.min(p, custom)
  return p
}

/** Brand colors for the three reseller tiers (ported from the design prototype). */
export const TIER_COLORS: Record<string, string> = {
  Gold: '#ffb02e',
  Silver: '#9aa3b5',
  Bronze: '#cd7f4d',
}

/** Resolve a tier name to its accent color, falling back to the brand color. */
export function tierColor(tier: string): string {
  return TIER_COLORS[tier] ?? 'var(--brand-1)'
}

/** The flat shape the reseller list/detail render. */
export interface ResellerView {
  id: string
  name: string
  email: string
  phone: string
  tier: string
  status: string
  balance: number
  cur: 'USD' | 'TRY'
  margin: number
  orders: number
  vol: number
  joined: string
  raw: AdminReseller
}

/** Map an enriched API reseller to the flat view the admin pages render. The
 *  User model carries no currency, so the admin views are USD. */
export function adaptReseller(r: AdminReseller): ResellerView {
  return {
    id: r.id,
    name: userName(r),
    email: r.email,
    phone: r.phone ?? '',
    tier: r.resellerTier ?? '',
    status: r.status || 'active',
    balance: r.walletBalance,
    cur: 'USD',
    margin: r.margin,
    orders: r.orders,
    vol: r.volume,
    joined: joinedLabel(r.createdAt),
    raw: r,
  }
}
