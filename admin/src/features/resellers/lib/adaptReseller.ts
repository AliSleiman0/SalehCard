import { userName, joinedLabel } from '@/features/users/lib/adaptUser'
import type { AdminReseller } from '../api/resellers'

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
