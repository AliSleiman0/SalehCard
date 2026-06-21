import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate, useLocation } from 'react-router-dom'
import { Icon, Logo } from '@/components'
import { ControlsMenu } from './ControlsMenu'
import { useUiStore } from '@/stores/ui'
import { useWallet } from '@/features/wallet/hooks/useWallet'
import { useCurrencyStore } from '@/stores/currency'
import { useCartCount } from '@/stores/cart'
import { useAuthStore } from '@/stores/auth'
import { useLocaleStore } from '@/stores/locale'
import { fmtPrice } from '@/lib/utils'
import { DEMO } from '@/lib/mock/demo'
import { useCategories } from '@/features/catalog/hooks/useCategories'
import { adaptRootCategory } from '@/features/catalog/lib/adaptCategory'

export function Header() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { pathname } = useLocation()
  const { agent } = useUiStore()
  const { currency } = useCurrencyStore()
  const cartCount = useCartCount()
  const { isAuthenticated } = useAuthStore()
  const balance = useWallet(isAuthenticated).data?.balance ?? 0
  const locale = useLocaleStore((s) => s.locale)
  const cats = (useCategories({ depth: 0 }).data?.data ?? [])
    .map((c) => adaptRootCategory(c, locale))
    .sort((a, b) => a.order - b.order)
  const [menu, setMenu] = useState(false)

  useEffect(() => {
    const h = () => setMenu(false)
    window.addEventListener('click', h)
    return () => window.removeEventListener('click', h)
  }, [])

  const onCat = pathname.startsWith('/category/') ? pathname.split('/')[2] : null
  const showCatNav = pathname === '/' || pathname.startsWith('/category')
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

        <div className="searchbar desktop-only" onClick={() => navigate('/category/games')}>
          <Icon name="search" size={18} />
          <input
            placeholder={t('search_ph')}
            onKeyDown={(e) => e.key === 'Enter' && navigate('/category/games')}
          />
        </div>

        <div className="header-right">
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
                setMenu((m) => !m)
              }}
              title="Language & currency"
            >
              <Icon name="globe" size={19} />
            </button>
            {menu && <ControlsMenu />}
          </div>

          <button className="icon-btn" onClick={() => navigate('/cart')}>
            <Icon name="cart" size={19} />
            {cartCount > 0 && <span className="cart-count num">{cartCount}</span>}
          </button>

          <div
            className="avatar desktop-only"
            onClick={() => navigate(isAuthenticated ? '/dashboard' : '/login')}
          >
            {isAuthenticated ? DEMO.user.initials : <Icon name="user" size={18} />}
          </div>
        </div>
      </div>

      {showCatNav && (
        <div className="wrap">
          <nav className="catnav">
            {cats.map((c) => (
              <a
                key={c.key}
                className={onCat === c.key ? 'on' : ''}
                onClick={() => navigate('/category/' + c.key)}
              >
                {c.name}
              </a>
            ))}
          </nav>
        </div>
      )}
    </header>
  )
}
