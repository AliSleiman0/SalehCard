import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { Icon, Panel, LoadingSpinner, ErrorState } from '@/components'
import { AcctSidebar } from '@/features/auth/components/AcctSidebar'
import { OrderRow } from '@/features/auth/components/OrderRow'
import { useLocaleStore } from '@/stores/locale'
import { useOrders } from '../hooks/useOrders'
import { adaptOrder } from '../lib/adaptOrder'

export default function OrdersPage() {
  const { t } = useTranslation()
  const locale = useLocaleStore((s) => s.locale)
  const query = useOrders()
  const orders = useMemo(
    () => (query.data ?? []).map((o) => adaptOrder(o, locale)),
    [query.data, locale],
  )

  return (
    <div className="wrap" style={{ padding: '26px 0 50px' }}>
      <h1 className="h1" style={{ marginBottom: 24 }}>
        {t('order_history')}
      </h1>
      <div className="cols-acct">
        <AcctSidebar active="orders" />
        <Panel>
          <div className="row" style={{ gap: 8, marginBottom: 6 }}>
            <span className="badge badge-secure">
              <Icon name="shield" size={12} />
              {t('secure_vault')}
            </span>
            <span className="small muted">{t('redeem_note')}</span>
          </div>
          {query.isLoading ? (
            <LoadingSpinner />
          ) : query.isError ? (
            <ErrorState
              title={t('no_results')}
              sub={t('no_results_sub')}
              onRetry={() => query.refetch()}
              retryLabel={t('retry')}
            />
          ) : orders.length === 0 ? (
            <p className="muted" style={{ padding: '20px 4px' }}>
              {t('no_orders')}
            </p>
          ) : (
            orders.map((o) => <OrderRow key={o.id} o={o} />)
          )}
        </Panel>
      </div>
    </div>
  )
}
