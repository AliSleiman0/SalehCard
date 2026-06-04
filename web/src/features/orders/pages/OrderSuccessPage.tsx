import { useTranslation } from 'react-i18next'
import { useNavigate, useLocation } from 'react-router-dom'
import { Icon, Card, Button, Badge, CodeVault } from '@/components'
import { OrderHead, CreditConfirm, TransferDetail } from '@/features/orders/components/OrderParts'
import type { OrderView } from '@/features/orders/types'
import { useWalletStore } from '@/stores/wallet'
import { useCurrencyStore } from '@/stores/currency'
import { fmtPrice } from '@/lib/utils'

const FALLBACK_ORDER: OrderView = {
  id: 'SC-90421',
  product: 'PUBG MOBILE — 1800 UC',
  art: 'battle',
  total: 24.99,
  method: 'Wallet',
  fulfill: 'credit',
  account: '5129384761',
  amount: '1800 UC',
  ts: 'Jun 4, 2026 · 14:22',
  status: 'delivered',
}

export default function OrderSuccessPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const loc = useLocation()
  const balance = useWalletStore((s) => s.balance)
  const currency = useCurrencyStore((s) => s.currency)

  const order = (loc.state as { order?: OrderView } | null)?.order ?? FALLBACK_ORDER
  const f = order.fulfill

  const title =
    f === 'credit' ? t('credited_title') : f === 'transfer' ? t('transfer_title') : t('success_title')
  const sub =
    f === 'credit' ? t('credited_sub') : f === 'transfer' ? t('transfer_sub') : t('success_sub')

  return (
    <div className="wrap" style={{ padding: '40px 0', maxWidth: 640 }}>
      <div className="col center" style={{ textAlign: 'center', gap: 14, marginBottom: 30 }}>
        <div
          style={{
            width: 78,
            height: 78,
            borderRadius: 24,
            background: 'var(--grad)',
            display: 'grid',
            placeItems: 'center',
            color: '#fff',
            boxShadow: '0 18px 44px -14px rgba(138,59,255,.8)',
          }}
        >
          <Icon name={f === 'transfer' ? 'repeat' : 'check'} size={38} stroke={3} />
        </div>
        <h1 className="h1">{title}</h1>
        <p className="muted" style={{ maxWidth: 440 }}>
          {sub}
        </p>
      </div>

      <Card style={{ display: 'flex', flexDirection: 'column', gap: 18 }}>
        <OrderHead o={order} />
        <hr className="divider" />
        {f === 'code' && (
          <>
            <CodeVault label={t('your_code')} code={order.code ?? ''} pin={order.pin} />
            <p className="tiny faint">{t('redeem_note')}</p>
          </>
        )}
        {f === 'credit' && <CreditConfirm o={order} />}
        {f === 'transfer' && <TransferDetail o={order} />}
        <div className="row" style={{ gap: 12 }}>
          <Button variant="ghost" block onClick={() => navigate('/orders')}>
            <Icon name="shield" size={17} />
            {t('view_order')}
          </Button>
          <Button variant="primary" block onClick={() => navigate('/')}>
            {t('back_home')}
          </Button>
        </div>
      </Card>

      <div className="row center" style={{ gap: 10, marginTop: 20 }}>
        {f === 'code' && (
          <Badge variant="secure">
            <Icon name="shield" size={12} />
            {t('secure_vault')}
          </Badge>
        )}
        {f === 'credit' && (
          <Badge variant="instant">
            <Icon name="bolt" size={12} />
            {t('trust_instant')}
          </Badge>
        )}
        {f === 'transfer' && (
          <Badge variant="soft">
            <Icon name="repeat" size={12} />
            {t('st_processing')}
          </Badge>
        )}
        <span className="small muted">
          {t('wallet_bal')}: <span className="num">{fmtPrice(balance, currency)}</span>
        </span>
      </div>
    </div>
  )
}
