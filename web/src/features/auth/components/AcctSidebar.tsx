import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { Icon } from '@/components'
import type { IconName } from '@/components'
import { useUiStore } from '@/stores/ui'
import { useAuthStore } from '@/stores/auth'
import { useLogout } from '@/features/auth/hooks/useLogout'
import { displayName, initials } from '@/features/auth/userDisplay'

type SidebarKey = 'dashboard' | 'wallet' | 'orders' | 'savedids' | 'reseller'

const PATHS: Record<Exclude<SidebarKey, 'reseller'>, string> = {
  dashboard: '/dashboard',
  wallet: '/wallet',
  orders: '/orders',
  savedids: '/saved-ids',
}

export function AcctSidebar({ active }: { active: SidebarKey }) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const user = useAuthStore((s) => s.user)
  const logout = useLogout()
  const closeDrawer = () => useUiStore.getState().setAcctDrawer(false)
  const items: { v: Exclude<SidebarKey, 'reseller'>; icon: IconName; l: string }[] = [
    { v: 'dashboard', icon: 'home', l: t('dashboard') },
    { v: 'wallet', icon: 'wallet', l: t('nav_wallet') },
    { v: 'orders', icon: 'shield', l: t('order_history') },
    { v: 'savedids', icon: 'user', l: t('saved_players') },
  ]
  return (
    <aside className="sidenav acct-drawer">
      <button className="icon-btn sidenav-close" onClick={closeDrawer} title="Close">
        <Icon name="close" size={18} />
      </button>
      <div
        className="panel card-pad"
        style={{ marginBottom: 14, display: 'flex', gap: 12, alignItems: 'center' }}
      >
        <span className="avatar" style={{ width: 46, height: 46 }}>
          {initials(user)}
        </span>
        <div className="col" style={{ gap: 2, minWidth: 0 }}>
          <span style={{ fontWeight: 800 }}>{displayName(user)}</span>
          <span className="tiny faint" style={{ overflow: 'hidden', textOverflow: 'ellipsis' }}>
            {user?.email}
          </span>
        </div>
      </div>
      {items.map((it) => (
        <a
          key={it.v}
          className={active === it.v ? 'on' : ''}
          onClick={() => {
            closeDrawer()
            navigate(PATHS[it.v])
          }}
        >
          <Icon name={it.icon} size={18} />
          {it.l}
        </a>
      ))}
      <a
        className={active === 'reseller' ? 'on' : ''}
        onClick={() => {
          useUiStore.getState().setAgent(true)
          navigate('/reseller')
        }}
        style={{ color: 'var(--agent)' }}
      >
        <Icon name="shield" size={18} />
        {t('agent_dash')}
      </a>
      <a
        onClick={() => {
          logout.mutate(undefined, { onSuccess: () => navigate('/') })
        }}
      >
        <Icon name="arrow" size={18} />
        {t('sign_out')}
      </a>
    </aside>
  )
}
