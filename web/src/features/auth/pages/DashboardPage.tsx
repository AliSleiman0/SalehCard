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
import { fmtPrice } from '@/lib/utils'

export default function DashboardPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const cur = useCurrencyStore((s) => s.currency)
  const locale = useLocaleStore((s) => s.locale)
  const user = useAuthStore((s) => s.user)
  const ids = user?.savedPlayerIds ?? []
  const balance = useWallet().data?.balance ?? 0
  const ordersQuery = useOrders()
  const recentOrders = useMemo(
    () => (ordersQuery.data ?? []).slice(0, 3).map((o) => adaptOrder(o, locale)),
    [ordersQuery.data, locale],
  )
  return (
    <div className="wrap" style={{ padding: '26px 0 50px' }}>
      <div className="cols-acct">
        <AcctSidebar active="dashboard" />
        <div className="col" style={{ gap: 24 }}>
          <div className="statgrid">
            <div className="stat bigbal">
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
              <span className="eyebrow">{t('loyalty')}</span>
              <div className="h1 num" style={{ marginTop: 8 }}>
                {(user?.loyaltyPoints ?? 0).toLocaleString()}
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
            {ids.length === 0 ? (
              <p className="muted" style={{ padding: '4px 2px' }}>
                {t('no_saved_ids')}
              </p>
            ) : (
              <div className="savedgrid">
                {ids.map((id) => (
                  <SavedIdCard key={id.value} id={id} />
                ))}
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
