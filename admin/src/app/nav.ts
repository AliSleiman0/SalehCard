import type { IconName } from '@/components'

export interface NavItem {
  to: string
  icon: IconName
  label: string // i18n key
  badge?: string
  badgeType?: 'warn' | 'danger'
  // RBAC domain gating this item: shown only when the admin holds
  // "<domain>.view" (or the super-admin wildcard). `superAdmin: true` items
  // (role management) show only for super admins.
  domain?: string
  superAdmin?: boolean
}

export interface NavGroup {
  group: string // i18n key
  items: NavItem[]
}

/**
 * Whether the logged-in admin may see a nav item / access its route: a
 * superAdmin-only item needs the wildcard, a domain item needs "<domain>.view",
 * and an item with neither is always allowed. The single predicate shared by the
 * sidebar filter, the route guard (RequireDomain), and firstAllowedRoute — keep
 * them in lockstep so a visible link never lands on a page the guard rejects.
 */
export function itemAllowed(
  item: { domain?: string; superAdmin?: boolean },
  can: (perm: string) => boolean,
): boolean {
  if (item.superAdmin) return can('*')
  return item.domain ? can(item.domain + '.view') : true
}

/**
 * First route the logged-in admin may view, in sidebar order — the landing
 * target after login and the redirect for a route their role can't access.
 * Falls back to a dead route (→ NotFound) only in the pathological
 * no-permissions case (a role was deleted out from under the admin).
 */
export function firstAllowedRoute(can: (perm: string) => boolean): string {
  for (const g of NAV) {
    for (const item of g.items) {
      if (itemAllowed(item, can)) return item.to
    }
  }
  return '/no-access'
}

// Sidebar structure mirrors the prototype's NAV.
export const NAV: NavGroup[] = [
  { group: 'grp_overview', items: [{ to: '/', icon: 'grid', label: 'nav_dashboard', domain: 'dashboard' }] },
  {
    group: 'grp_catalog',
    items: [
      { to: '/products', icon: 'box', label: 'nav_products', domain: 'products' },
      { to: '/categories', icon: 'layers', label: 'nav_categories', domain: 'categories' },
      { to: '/inventory', icon: 'layers', label: 'nav_inventory', domain: 'inventory' },
      { to: '/offers', icon: 'tag', label: 'nav_offers', domain: 'offers' },
    ],
  },
  {
    group: 'grp_operations',
    items: [
      { to: '/orders', icon: 'bag', label: 'nav_orders', domain: 'orders' },
      { to: '/bridge', icon: 'server', label: 'nav_bridge', domain: 'bridge' },
      { to: '/suppliers', icon: 'box', label: 'nav_suppliers', domain: 'suppliers' },
      { to: '/users', icon: 'users', label: 'nav_users', domain: 'users' },
      { to: '/resellers', icon: 'handshake', label: 'nav_resellers', domain: 'resellers' },
      { to: '/kyc', icon: 'id', label: 'nav_kyc', domain: 'kyc' },
    ],
  },
  {
    group: 'grp_finance',
    items: [
      { to: '/finance', icon: 'wallet', label: 'nav_wallet', domain: 'finance' },
      { to: '/topups', icon: 'coins', label: 'nav_topups', domain: 'topups' },
      { to: '/payments', icon: 'card', label: 'nav_payments', domain: 'payments' },
      { to: '/promos', icon: 'tag', label: 'nav_promos', domain: 'promos' },
      { to: '/expenses', icon: 'coins', label: 'nav_expenses', domain: 'expenses' },
    ],
  },
  { group: 'grp_content', items: [{ to: '/reviews', icon: 'star', label: 'nav_reviews', domain: 'reviews' }] },
  {
    group: 'grp_system',
    items: [
      { to: '/roles', icon: 'shield', label: 'nav_roles', superAdmin: true },
      { to: '/audit', icon: 'shield', label: 'nav_audit', domain: 'audit' },
      { to: '/settings', icon: 'settings', label: 'nav_settings', domain: 'settings' },
    ],
  },
]
