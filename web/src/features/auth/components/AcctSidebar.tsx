import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { Icon } from '@/components'
import type { IconName } from '@/components'
import { useUiStore } from '@/stores/ui'
import { DEMO } from '@/lib/mock/demo'

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
  const items: { v: Exclude<SidebarKey, 'reseller'>; icon: IconName; l: string }[] = [
    { v: 'dashboard', icon: 'home', l: t('dashboard') },
    { v: 'wallet', icon: 'wallet', l: t('nav_wallet') },
    { v: 'orders', icon: 'shield', l: t('order_history') },
    { v: 'savedids', icon: 'user', l: t('saved_players') },
  ]
  return (
    <aside className="sidenav">
      <div
        className="panel card-pad"
        style={{ marginBottom: 14, display: 'flex', gap: 12, alignItems: 'center' }}
      >
        <span className="avatar" style={{ width: 46, height: 46 }}>
          {DEMO.user.initials}
        </span>
        <div className="col" style={{ gap: 2, minWidth: 0 }}>
          <span style={{ fontWeight: 800 }}>{DEMO.user.name}</span>
          <span className="tiny faint" style={{ overflow: 'hidden', textOverflow: 'ellipsis' }}>
            {DEMO.user.email}
          </span>
        </div>
      </div>
      {items.map((it) => (
        <a
          key={it.v}
          className={active === it.v ? 'on' : ''}
          onClick={() => navigate(PATHS[it.v])}
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
    </aside>
  )
}
