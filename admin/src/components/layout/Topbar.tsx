import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { Icon } from '@/components'
import { useThemeStore } from '@/stores/theme'
import { useLocaleStore } from '@/stores/locale'
import { useAuthStore } from '@/stores/auth'
import { LOCALE_NAMES, LOCALE_FLAGS } from '@/lib/utils'
import type { Locale, Theme } from '@/types'

type Menu = 'theme' | 'lang' | 'admin' | null

export function Topbar() {
  const { t } = useTranslation()
  const { theme, setTheme } = useThemeStore()
  const { locale, setLocale } = useLocaleStore()
  const { user, logout } = useAuthStore()
  const navigate = useNavigate()
  const [menu, setMenu] = useState<Menu>(null)
  const [search, setSearch] = useState('')

  const close = () => setMenu(null)
  useEffect(() => {
    if (!menu) return
    const h = () => close()
    window.addEventListener('click', h)
    return () => window.removeEventListener('click', h)
  }, [menu])
  const stop = (e: React.MouseEvent) => e.stopPropagation()

  const initials = (user?.name ?? 'Admin')
    .split(' ')
    .map((w) => w[0])
    .slice(0, 2)
    .join('')

  // Global search: routes a query into Orders for now (the design's intent is a
  // unified search across orders/users/products/codes).
  // TODO: build a real cross-entity search endpoint + results popover.
  const runSearch = () => {
    if (search.trim()) navigate(`/orders?q=${encodeURIComponent(search.trim())}`)
  }

  return (
    <header className="topbar">
      <div
        className="tb-search"
        onClick={(e) => (e.currentTarget.querySelector('input') as HTMLInputElement)?.focus()}
      >
        <Icon name="search" size={17} />
        <input
          placeholder={t('search_ph')}
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && runSearch()}
        />
        <kbd>⌘K</kbd>
      </div>
      <div className="tb-actions">
        {/* theme */}
        <div style={{ position: 'relative' }} onClick={stop}>
          <button className="iconbtn" onClick={() => setMenu(menu === 'theme' ? null : 'theme')} title={t('theme')}>
            <Icon name={theme === 'dark' ? 'moon' : 'sun'} size={18} />
          </button>
          {menu === 'theme' && (
            <div className="menu">
              <div className="menu-label">{t('theme')}</div>
              {(['dark', 'light'] as Theme[]).map((v) => (
                <button
                  key={v}
                  className={theme === v ? 'on' : ''}
                  onClick={() => {
                    setTheme(v)
                    close()
                  }}
                >
                  <Icon name={v === 'dark' ? 'moon' : 'sun'} size={16} /> {t(v)}{' '}
                  <Icon name="check" size={15} className="ck" />
                </button>
              ))}
            </div>
          )}
        </div>
        {/* language */}
        <div style={{ position: 'relative' }} onClick={stop}>
          <button className="iconbtn" onClick={() => setMenu(menu === 'lang' ? null : 'lang')} title={t('language')}>
            <Icon name="globe" size={18} />
          </button>
          {menu === 'lang' && (
            <div className="menu">
              <div className="menu-label">{t('language')}</div>
              {(['en', 'ar', 'tr'] as Locale[]).map((v) => (
                <button
                  key={v}
                  className={locale === v ? 'on' : ''}
                  onClick={() => {
                    setLocale(v)
                    close()
                  }}
                >
                  <span style={{ width: 22, fontWeight: 800, fontSize: 11.5, color: 'var(--text-faint)' }}>
                    {LOCALE_FLAGS[v]}
                  </span>
                  {LOCALE_NAMES[v]} <Icon name="check" size={15} className="ck" />
                </button>
              ))}
            </div>
          )}
        </div>
        <button className="iconbtn" title="Notifications">
          <Icon name="bell" size={18} />
          <span className="ping">5</span>
        </button>
        {/* admin */}
        <div style={{ position: 'relative' }} onClick={stop}>
          <div className="tb-admin" onClick={() => setMenu(menu === 'admin' ? null : 'admin')}>
            <div className="tb-ava">{initials}</div>
            <div className="desktop-only" style={{ lineHeight: 1.15 }}>
              <div style={{ fontWeight: 800, fontSize: 13 }}>{user?.name ?? 'Admin'}</div>
              <div style={{ fontSize: 11, color: 'var(--text-faint)' }}>Super admin</div>
            </div>
            <Icon name="chevdown" size={15} />
          </div>
          {menu === 'admin' && (
            <div className="menu" style={{ minWidth: 200 }}>
              <div className="menu-label">Signed in as</div>
              <div style={{ padding: '4px 10px 10px', fontSize: 13 }}>
                <div style={{ fontWeight: 800 }}>{user?.email ?? '—'}</div>
              </div>
              <button onClick={() => navigate('/settings')}>
                <Icon name="settings" size={16} /> Account settings
              </button>
              <button>
                <Icon name="shield" size={16} /> Activity log
              </button>
              <button
                style={{ color: 'var(--danger)' }}
                onClick={() => {
                  logout()
                  navigate('/login')
                }}
              >
                <Icon name="logout" size={16} /> {t('sign_out')}
              </button>
            </div>
          )}
        </div>
      </div>
    </header>
  )
}
