import { useTranslation } from 'react-i18next'
import { useLocation, useNavigate } from 'react-router-dom'
import { Icon } from '@/components'
import { NAV } from '@/app/nav'
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
        {NAV.map((g) => (
          <div className="sb-group" key={g.group}>
            <div className="sb-grouplabel">{t(g.group)}</div>
            {g.items.map((item) => (
              <div
                key={item.to}
                className={cn('sb-link', isActive(pathname, item.to) && 'on')}
                onClick={() => navigate(item.to)}
                title={t(item.label)}
              >
                <Icon name={item.icon} size={19} />
                <span>{t(item.label)}</span>
                {item.badge && (
                  <span className={cn('sb-badge', item.badgeType === 'warn' && 'warn')}>{item.badge}</span>
                )}
              </div>
            ))}
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
