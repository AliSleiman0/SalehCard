import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate, useLocation } from 'react-router-dom'
import { Icon, Logo } from '@/components'
import { ControlsMenu } from './ControlsMenu'
import { AccountMenu } from './AccountMenu'
import { useUiStore } from '@/stores/ui'
import { useWallet } from '@/features/wallet/hooks/useWallet'
import { useCurrencyStore } from '@/stores/currency'
import { useCartCount } from '@/stores/cart'
import { useAuthStore } from '@/stores/auth'
import { useLocaleStore } from '@/stores/locale'
import { fmtPrice } from '@/lib/utils'
import { initials } from '@/features/auth/userDisplay'
import { useCategories } from '@/features/catalog/hooks/useCategories'
import { adaptRootCategory } from '@/features/catalog/lib/adaptCategory'
import { useUnreadCount } from '@/features/notifications/hooks/useUnreadCount'
import { SearchSuggest } from '@/features/catalog/components/SearchSuggest'

export function Header() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { pathname } = useLocation()
  const { agent } = useUiStore()
  const { currency } = useCurrencyStore()
  const cartCount = useCartCount()
  const { isAuthenticated, user } = useAuthStore()
  const balance = useWallet(isAuthenticated).data?.balance ?? 0
  const locale = useLocaleStore((s) => s.locale)
  const cats = (useCategories({ depth: 0 }).data?.data ?? [])
    .map((c) => adaptRootCategory(c, locale))
    .sort((a, b) => a.order - b.order)
  // At most one header popover is open: language/currency controls or the
  // avatar's account menu. Any outside click or Escape closes it.
  const [menu, setMenu] = useState<null | 'controls' | 'account'>(null)
  const toggle = (m: 'controls' | 'account') => setMenu((cur) => (cur === m ? null : m))
  const unread = useUnreadCount().data ?? 0

  useEffect(() => {
    const close = () => setMenu(null)
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') close()
    }
    window.addEventListener('click', close)
    window.addEventListener('keydown', onKey)
    return () => {
      window.removeEventListener('click', close)
      window.removeEventListener('keydown', onKey)
    }
  }, [])

  // Navigating anywhere closes an open popover.
  useEffect(() => setMenu(null), [pathname])

  const onCat = pathname.startsWith('/category/') ? pathname.split('/')[2] : null
  // The category bar is a persistent browse affordance on every page except the
  // focused auth screens, so the store is always reachable (incl. the dashboard).
  const showCatNav = pathname !== '/login' && pathname !== '/register'
  const onAcctPage = ['/dashboard', '/wallet', '/orders', '/saved-ids'].some((p) =>
    pathname.startsWith(p),
  )

  return (
    <header className="appheader">
      <div className="wrap">
        <div className="header-left">
          {isAuthenticated && onAcctPage && (
            <button
              className="icon-btn acct-menu-btn"
              onClick={(e) => {
                e.stopPropagation()
                useUiStore.getState().toggleAcctDrawer()
              }}
              title="Account menu"
            >
              <Icon name="user" size={19} />
            </button>
          )}
          <div className="clickable" onClick={() => navigate('/')}>
            <Logo size={30} agent={agent} />
          </div>
        </div>

        <SearchSuggest />

        <div className="header-right">
          <button
            className="icon-btn mobile-only"
            onClick={() => navigate('/search')}
            title={t('search_ph')}
          >
            <Icon name="search" size={19} />
          </button>

          {isAuthenticated && (
            <div className="wallet-pill desktop-only" onClick={() => navigate('/wallet')}>
              <span className="dot-grad">
                <Icon name="wallet" size={15} />
              </span>
              <span className="num">{fmtPrice(balance, currency)}</span>
            </div>
          )}

          <div style={{ position: 'relative' }}>
            <button
              className="icon-btn"
              onClick={(e) => {
                e.stopPropagation()
                toggle('controls')
              }}
              title="Language & currency"
              aria-haspopup="menu"
              aria-expanded={menu === 'controls'}
            >
              <Icon name="globe" size={19} />
            </button>
            {menu === 'controls' && <ControlsMenu />}
          </div>

          {isAuthenticated && (
            <button className="icon-btn" onClick={() => navigate('/notifications')} title={t('notifications')}>
              <Icon name="bell" size={19} />
              {unread > 0 && <span className="cart-count num">{unread > 9 ? '9+' : unread}</span>}
            </button>
          )}

          <button className="icon-btn" onClick={() => navigate('/cart')}>
            <Icon name="cart" size={19} />
            {cartCount > 0 && <span className="cart-count num">{cartCount}</span>}
          </button>

          {isAuthenticated ? (
            <div className="desktop-only" style={{ position: 'relative' }}>
              <button
                type="button"
                className="avatar"
                style={{ border: 0 }}
                onClick={(e) => {
                  e.stopPropagation()
                  toggle('account')
                }}
                title={t('account_menu')}
                aria-haspopup="menu"
                aria-expanded={menu === 'account'}
              >
                {initials(user)}
              </button>
              {menu === 'account' && <AccountMenu onClose={() => setMenu(null)} />}
            </div>
          ) : (
            <div className="avatar desktop-only" onClick={() => navigate('/login')}>
              <Icon name="user" size={18} />
            </div>
          )}
        </div>
      </div>

      {showCatNav && (
        <div className="wrap">
          <nav className="catnav">
            <a
              className={pathname.startsWith('/categories') ? 'on' : ''}
              onClick={() => navigate('/categories')}
            >
              {t('nav_shop')}
            </a>
            {cats.map((c) => (
              <a
                key={c.key}
                className={onCat === c.key ? 'on' : ''}
                onClick={() => navigate('/category/' + c.key)}
              >
                {c.name}
              </a>
            ))}
            {isAuthenticated && (
              <a
                className={pathname === '/offers' ? 'on' : ''}
                style={{ color: 'var(--danger)', fontWeight: 800 }}
                onClick={() => navigate('/offers')}
              >
                {t('offers_nav')}
              </a>
            )}
          </nav>
        </div>
      )}
    </header>
  )
}
