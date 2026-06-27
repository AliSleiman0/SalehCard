import type { UserRole } from '@/types'
import type { AdminUser } from '../api/users'

/** The flat shape the user list/detail render. */
export interface UserView {
  id: string
  name: string
  email: string
  phone: string
  role: UserRole
  status: string
  balance: number
  cur: 'USD' | 'TRY'
  orders: number
  spent: number
  loyalty: number
  joined: string
  raw: AdminUser
}

/** Display name for a user — the User model has no name field, so we derive one
 *  from the email local-part, falling back to phone. */
export function userName(u: Pick<AdminUser, 'email' | 'phone'>): string {
  if (u.email) return u.email.split('@')[0]
  if (u.phone) return u.phone
  return 'Unknown'
}

/** Format a join date as a compact "Mon YYYY" (matches the prior mock display). */
export function joinedLabel(iso: string): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return '—'
  return d.toLocaleDateString('en-US', { month: 'short', year: 'numeric' })
}

/** Map an enriched API user to the flat view the admin pages render. The User
 *  model carries no currency, so the admin views are USD. */
export function adaptUser(u: AdminUser): UserView {
  return {
    id: u.id,
    name: userName(u),
    email: u.email,
    phone: u.phone ?? '',
    role: u.role,
    status: u.status || 'active',
    balance: u.walletBalance,
    cur: 'USD',
    orders: u.orders,
    spent: u.spent,
    loyalty: u.loyaltyPoints,
    joined: joinedLabel(u.createdAt),
    raw: u,
  }
}
