import { useTranslation } from 'react-i18next'
import { useLocation, useNavigate } from 'react-router-dom'
import { Icon } from '@/components'
import { NAV, itemAllowed } from '@/app/nav'
import { useDashboardStats } from '@/features/dashboard/hooks/useDashboard'
import { useCan } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'
import { cn } from '@/lib/utils'

function isActive(pathname: string, to: string): boolean {
  if (to === '/') return pathname === '/'
  return pathname === to || pathname.startsWith(to + '/')
}

export function Sidebar() {
  const { t } = useTranslation()
  const { collapsed, toggleCollapsed } = useUiStore()
  const navigate = useNavigate()
  const { pathname } = useLocation()
  const can = useCan()

  // RBAC: drop nav items the admin's role cannot view (super-admin-only items
  // need the wildcard), then drop groups left empty.
  const visibleNav = NAV.map((g) => ({
    ...g,
    items: g.items.filter((item) => itemAllowed(item, can)),
  })).filter((g) => g.items.length > 0)

  // Live work-queue badges: shared cache with the dashboard stats query (no extra
  // request); refreshes on window focus / navigation. Skipped without
  // dashboard.view — the stats endpoint would 403.
  const { data: statsRes } = useDashboardStats({ enabled: can('dashboard.view') })
  const pendingByRoute: Record<string, number | undefined> = {
    '/topups': statsRes?.data?.pendingTopups,
    '/kyc': statsRes?.data?.pendingKyc,
  }

  return (
    <aside className="sidebar">
      <div className="sb-head">
        <div className="sb-mark">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none">
            <path d="M16 5H10a3 3 0 0 0 0 6h4a3 3 0 0 1 0 6H7" stroke="#fff" strokeWidth="2.6" strokeLinecap="round" />
            <circle cx="18.5" cy="6" r="1.7" fill="#22e3c8" />
          </svg>
        </div>
        <div className="sb-word">
          Saleh<span className="grad-text">Card</span>
        </div>
      </div>
      <nav className="sb-scroll">
        {visibleNav.map((g) => (
          <div className="sb-group" key={g.group}>
            <div className="sb-grouplabel">{t(g.group)}</div>
            {g.items.map((item) => {
              const pending = pendingByRoute[item.to]
              const badge = item.badge ?? (pending && pending > 0 ? String(pending) : undefined)
              const badgeType = item.badge ? item.badgeType : 'warn'
              return (
                <div
                  key={item.to}
                  className={cn('sb-link', isActive(pathname, item.to) && 'on')}
                  onClick={() => navigate(item.to)}
                  title={t(item.label)}
                >
                  <Icon name={item.icon} size={19} />
                  <span>{t(item.label)}</span>
                  {badge && (
                    <span className={cn('sb-badge', badgeType === 'warn' && 'warn')}>{badge}</span>
                  )}
                </div>
              )
            })}
          </div>
        ))}
      </nav>
      <div className="sb-foot">
        <button className="sb-collapse" onClick={toggleCollapsed} title={t('collapse')}>
          <Icon name={collapsed ? 'chevright' : 'chevleft'} size={18} />
          <span className="sf-txt">{t('collapse')}</span>
        </button>
      </div>
    </aside>
  )
}
