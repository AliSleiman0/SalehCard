import { useTranslation } from 'react-i18next'
import { Icon, Panel } from '@/components'
import { AcctSidebar } from '@/features/auth/components/AcctSidebar'
import { OrderRow } from '@/features/auth/components/OrderRow'
import { DEMO } from '@/lib/mock/demo'

// TODO: wire to order module API
export default function OrdersPage() {
  const { t } = useTranslation()
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
          {DEMO.orders.map((o) => (
            <OrderRow key={o.id} o={o} />
          ))}
        </Panel>
      </div>
    </div>
  )
}
