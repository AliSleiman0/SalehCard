import { useTranslation } from 'react-i18next'
import { useNavigate, useLocation, useParams } from 'react-router-dom'
import { Icon, Card, Button, Badge, CodeVault, LoadingSpinner, ErrorState } from '@/components'
import { OrderHead, CreditConfirm, TransferDetail } from '@/features/orders/components/OrderParts'
import type { OrderView } from '@/features/orders/types'
import { useWallet } from '@/features/wallet/hooks/useWallet'
import { useCurrencyStore } from '@/stores/currency'
import { useLocaleStore } from '@/stores/locale'
import { useOrder } from '@/features/orders/hooks/useOrder'
import { adaptOrder } from '@/features/orders/lib/adaptOrder'
import { fmtPrice } from '@/lib/utils'

export default function OrderSuccessPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const loc = useLocation()
  const { id } = useParams()
  const locale = useLocaleStore((s) => s.locale)
  const balance = useWallet().data?.balance ?? 0
  const currency = useCurrencyStore((s) => s.currency)

  // Prefer the order handed over via router state (fresh from checkout); fall
  // back to fetching it by id (e.g. on a hard reload of this page).
  const stateOrder = (loc.state as { order?: OrderView } | null)?.order
  const query = useOrder(stateOrder ? '' : (id ?? ''))

  if (!stateOrder && query.isLoading) return <LoadingSpinner />
  if (!stateOrder && (query.isError || !query.data)) {
    return (
      <ErrorState
        title={t('no_results')}
        sub={t('no_results_sub')}
        onRetry={() => query.refetch()}
        retryLabel={t('retry')}
      />
    )
  }

  const order = stateOrder ?? adaptOrder(query.data!, locale)
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
