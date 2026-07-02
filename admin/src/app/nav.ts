import type { IconName } from '@/components'

export interface NavItem {
  to: string
  icon: IconName
  label: string // i18n key
  badge?: string
  badgeType?: 'warn' | 'danger'
}

export interface NavGroup {
  group: string // i18n key
  items: NavItem[]
}

// Sidebar structure mirrors the prototype's NAV (badge counts are demo values
// from the design until those modules are wired).
export const NAV: NavGroup[] = [
  { group: 'grp_overview', items: [{ to: '/', icon: 'grid', label: 'nav_dashboard' }] },
  {
    group: 'grp_catalog',
    items: [
      { to: '/products', icon: 'box', label: 'nav_products' },
      { to: '/inventory', icon: 'layers', label: 'nav_inventory', badge: '8', badgeType: 'warn' },
      { to: '/offers', icon: 'tag', label: 'nav_offers' },
    ],
  },
  {
    group: 'grp_operations',
    items: [
      { to: '/orders', icon: 'bag', label: 'nav_orders', badge: '3', badgeType: 'danger' },
      { to: '/users', icon: 'users', label: 'nav_users' },
      { to: '/resellers', icon: 'handshake', label: 'nav_resellers' },
      { to: '/kyc', icon: 'id', label: 'nav_kyc' },
    ],
  },
  {
    group: 'grp_finance',
    items: [
      { to: '/finance', icon: 'wallet', label: 'nav_wallet' },
      { to: '/topups', icon: 'coins', label: 'nav_topups' },
      { to: '/promos', icon: 'tag', label: 'nav_promos' },
      { to: '/expenses', icon: 'coins', label: 'nav_expenses' },
    ],
  },
  { group: 'grp_content', items: [{ to: '/reviews', icon: 'star', label: 'nav_reviews' }] },
  { group: 'grp_system', items: [{ to: '/settings', icon: 'settings', label: 'nav_settings' }] },
]
