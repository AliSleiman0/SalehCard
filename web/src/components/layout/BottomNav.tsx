import { useTranslation } from 'react-i18next'
import { useNavigate, useLocation } from 'react-router-dom'
import { Icon } from '@/components'
import type { IconName } from '@/components'
import { useCartCount } from '@/stores/cart'
import { useAuthStore } from '@/stores/auth'

export function BottomNav() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { pathname } = useLocation()
  const cartCount = useCartCount()
  const { isAuthenticated } = useAuthStore()

  const items: { path: string; icon: IconName; label: string; badge?: number }[] = [
    { path: '/', icon: 'home', label: t('nav_home') },
    { path: '/category/games', icon: 'grid', label: t('nav_cats') },
    { path: '/wallet', icon: 'wallet', label: t('nav_wallet') },
    { path: '/cart', icon: 'cart', label: t('nav_cart'), badge: cartCount },
    {
      path: isAuthenticated ? '/dashboard' : '/login',
      icon: 'user',
      label: t('nav_account'),
    },
  ]

  const isActive = (path: string) =>
    path === '/' ? pathname === '/' : pathname.startsWith(path.split('/').slice(0, 2).join('/'))

  return (
    <nav className="bottomnav mobile-only">
      {items.map((it) => (
        <a
          key={it.path + it.label}
          className={isActive(it.path) ? 'on' : ''}
          onClick={() => navigate(it.path)}
        >
          <span className="ni" style={{ position: 'relative' }}>
            <Icon name={it.icon} size={20} />
            {it.badge !== undefined && it.badge > 0 && (
              <span className="cart-count num">{it.badge}</span>
            )}
          </span>
          {it.label}
        </a>
      ))}
    </nav>
  )
}
