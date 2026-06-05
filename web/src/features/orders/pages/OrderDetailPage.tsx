import { useTranslation } from 'react-i18next'
import { useNavigate, useParams } from 'react-router-dom'
import { Icon, Card, Button, CodeVault, LoadingSpinner, ErrorState } from '@/components'
import { OrderHead, CreditConfirm, TransferDetail } from '@/features/orders/components/OrderParts'
import { useLocaleStore } from '@/stores/locale'
import { useOrder } from '../hooks/useOrder'
import { adaptOrder } from '../lib/adaptOrder'

export default function OrderDetailPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { id } = useParams()
  const locale = useLocaleStore((s) => s.locale)
  const query = useOrder(id ?? '')

  if (query.isLoading) return <LoadingSpinner />
  if (query.isError || !query.data) {
    return (
      <ErrorState
        title={t('no_results')}
        sub={t('no_results_sub')}
        onRetry={() => query.refetch()}
        retryLabel={t('retry')}
      />
    )
  }

  const raw = query.data
  const o = adaptOrder(raw, locale)
  const reorderId = raw.items[0]?.productId

  return (
    <div className="wrap" style={{ padding: '26px 0 50px', maxWidth: 720 }}>
      <a
        className="small clickable faint"
        onClick={() => navigate('/orders')}
        style={{ display: 'inline-block', marginBottom: 16 }}
      >
        ← {t('order_history')}
      </a>
      <Card style={{ display: 'flex', flexDirection: 'column', gap: 18 }}>
        <OrderHead o={o} />
        <hr className="divider" />
        {(!o.fulfill || o.fulfill === 'code') &&
          (o.code ? (
            <CodeVault label={t('your_code')} code={o.code} pin={o.pin} />
          ) : (
            <div className="slot" style={{ padding: 24 }}>
              {t('processing')}…
            </div>
          ))}
        {o.fulfill === 'credit' && <CreditConfirm o={o} />}
        {o.fulfill === 'transfer' && <TransferDetail o={o} />}
        <div className="row" style={{ gap: 12 }}>
          <Button
            variant="ghost"
            block
            onClick={() => navigate(reorderId ? '/product/' + reorderId : '/')}
          >
            <Icon name="repeat" size={16} />
            {t('reorder')}
          </Button>
          <Button variant="primary" block onClick={() => navigate('/')}>
            {t('continue_shop')}
          </Button>
        </div>
      </Card>
    </div>
  )
}
