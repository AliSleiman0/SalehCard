import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { Icon, Button } from '@/components'
import { AcctSidebar } from '@/features/auth/components/AcctSidebar'
import { OrderRow } from '@/features/auth/components/OrderRow'
import { SavedIdCard } from '@/features/auth/components/SavedIdCard'
import { useCurrencyStore } from '@/stores/currency'
import { useLocaleStore } from '@/stores/locale'
import { useAuthStore } from '@/stores/auth'
import { useWallet } from '@/features/wallet/hooks/useWallet'
import { useOrders } from '@/features/orders/hooks/useOrders'
import { adaptOrder } from '@/features/orders/lib/adaptOrder'
import { displayName } from '@/features/auth/userDisplay'
import { fmtPrice } from '@/lib/utils'
import { DEMO } from '@/lib/mock/demo'

export default function DashboardPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const cur = useCurrencyStore((s) => s.currency)
  const locale = useLocaleStore((s) => s.locale)
  const user = useAuthStore((s) => s.user)
  const balance = useWallet().data?.balance ?? 0
  const ordersQuery = useOrders()
  const recentOrders = useMemo(
    () => (ordersQuery.data ?? []).slice(0, 3).map((o) => adaptOrder(o, locale)),
    [ordersQuery.data, locale],
  )
  // cashback and saved IDs remain mock until the loyalty/profile work lands.
  const u = DEMO.user

  return (
    <div className="wrap" style={{ padding: '26px 0 50px' }}>
      <h1 className="h1" style={{ marginBottom: 6 }}>
        {t('hi')}, {displayName(user).split(' ')[0] || u.name.split(' ')[0]} 👋
      </h1>
      <p className="muted" style={{ marginBottom: 24 }}>
        {t('overview')}
      </p>
      <div className="cols-acct">
        <AcctSidebar active="dashboard" />
        <div className="col" style={{ gap: 24 }}>
          <div className="statgrid">
            <div className="stat bigbal" style={{ gridColumn: 'span 1' }}>
              <span className="eyebrow" style={{ color: 'rgba(255,255,255,.8)' }}>
                <Icon name="wallet" size={13} /> {t('current_balance')}
              </span>
              <div className="display-l num" style={{ color: '#fff', margin: '8px 0 14px' }}>
                {fmtPrice(balance, cur)}
              </div>
              <Button variant="cyan" size="sm" onClick={() => navigate('/wallet')}>
                {t('topup')}
              </Button>
            </div>
            <div className="stat">
              <span className="eyebrow">{t('cashback')}</span>
              <div className="h1 num" style={{ marginTop: 8 }}>
                {fmtPrice(u.cashback, cur)}
              </div>
              <span className="tiny faint">{t('this_month')}</span>
            </div>
            <div className="stat">
              <span className="eyebrow">{t('loyalty')}</span>
              <div className="h1 num" style={{ marginTop: 8 }}>
                {(user?.loyaltyPoints ?? u.loyalty).toLocaleString()}
              </div>
              <span className="tiny faint">pts</span>
            </div>
          </div>

          <div className="panel card-pad">
            <div className="row between" style={{ marginBottom: 10 }}>
              <h3 className="h3">{t('recent_orders')}</h3>
              <a
                className="small clickable"
                style={{ fontWeight: 700, color: 'var(--brand-1)' }}
                onClick={() => navigate('/orders')}
              >
                {t('view_all')} →
              </a>
            </div>
            {recentOrders.length === 0 ? (
              <p className="muted" style={{ padding: '12px 4px' }}>
                {t('no_orders')}
              </p>
            ) : (
              recentOrders.map((o) => <OrderRow key={o.id} o={o} />)
            )}
          </div>

          <div className="panel card-pad">
            <div className="row between" style={{ marginBottom: 14 }}>
              <h3 className="h3">{t('saved_players')}</h3>
              <a
                className="small clickable"
                style={{ fontWeight: 700, color: 'var(--brand-1)' }}
                onClick={() => navigate('/saved-ids')}
              >
                {t('manage')} →
              </a>
            </div>
            <div className="grid" style={{ gridTemplateColumns: 'repeat(3,1fr)', gap: 12 }}>
              {DEMO.savedIds.map((s) => (
                <SavedIdCard key={s.id} s={s} compact />
              ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
