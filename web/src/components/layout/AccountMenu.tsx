import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { Icon } from '@/components'
import type { IconName } from '@/components'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'
import { useLogout } from '@/features/auth/hooks/useLogout'
import { displayName } from '@/features/auth/userDisplay'

// AccountMenu is the dropdown under the header avatar (client msg #49/#58):
// the account destinations from the sidebar + sign out, reachable from any page.
// It reuses the header's `.pop` popover styling; the parent owns open/close.
export function AccountMenu({ onClose }: { onClose: () => void }) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const user = useAuthStore((s) => s.user)
  const logout = useLogout()

  const go = (path: string) => {
    onClose()
    navigate(path)
  }
  const items: { path: string; icon: IconName; label: string }[] = [
    { path: '/', icon: 'grid', label: t('browse_store') },
    { path: '/dashboard', icon: 'home', label: t('dashboard') },
    { path: '/wallet', icon: 'wallet', label: t('nav_wallet') },
    { path: '/orders', icon: 'shield', label: t('order_history') },
    { path: '/saved-ids', icon: 'user', label: t('saved_players') },
    { path: '/complaints', icon: 'chat', label: t('complaints') },
  ]

  return (
    <div
      className="pop acct-pop"
      role="menu"
      aria-label={t('account_menu')}
      onClick={(e) => e.stopPropagation()}
      style={{ minWidth: 230 }}
    >
      <div className="col" style={{ gap: 2, padding: '8px 12px 10px', minWidth: 0 }}>
        <span style={{ fontWeight: 800 }}>{displayName(user)}</span>
        <span className="tiny faint" style={{ overflow: 'hidden', textOverflow: 'ellipsis' }}>
          {user?.email || user?.phone}
        </span>
      </div>
      <hr className="divider" style={{ margin: '0 0 6px' }} />
      {items.map((it) => (
        <button key={it.path} role="menuitem" onClick={() => go(it.path)}>
          <Icon name={it.icon} size={16} /> {it.label}
        </button>
      ))}
      {user?.role === 'reseller' && (
        <button
          role="menuitem"
          style={{ color: 'var(--agent)' }}
          onClick={() => {
            useUiStore.getState().setAgent(true)
            go('/reseller')
          }}
        >
          <Icon name="shield" size={16} /> {t('agent_dash')}
        </button>
      )}
      <hr className="divider" style={{ margin: '6px 0' }} />
      <button
        role="menuitem"
        disabled={logout.isPending}
        onClick={() =>
          logout.mutate(undefined, {
            onSuccess: () => {
              onClose()
              navigate('/')
            },
          })
        }
      >
        <Icon name="arrow" size={16} /> {t('sign_out')}
      </button>
    </div>
  )
}
