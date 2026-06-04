import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { Button, Panel } from '@/components'
import { LineItem } from '@/features/checkout/components/LineItem'
import { OrderSummary } from '@/features/checkout/components/OrderSummary'
import { useCartStore } from '@/stores/cart'

export default function CartPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const items = useCartStore((s) => s.items)
  const remove = useCartStore((s) => s.remove)
  const setQty = useCartStore((s) => s.setQty)

  const subtotal = items.reduce((s, x) => s + x.price * x.qty, 0)

  if (!items.length)
    return (
      <div className="wrap" style={{ padding: '60px 0', textAlign: 'center' }}>
        <div
          className="slot"
          style={{ width: 120, height: 120, margin: '0 auto 22px', borderRadius: 28 }}
        >
          cart.empty
        </div>
        <h2 className="h2">{t('cart_empty')}</h2>
        <Button
          variant="primary"
          size="lg"
          style={{ marginTop: 20 }}
          onClick={() => navigate('/')}
        >
          {t('continue_shop')}
        </Button>
      </div>
    )

  return (
    <div className="wrap" style={{ padding: '26px 0 50px' }}>
      <h1 className="h1" style={{ marginBottom: 22 }}>
        {t('cart_title')}
      </h1>
      <div className="cols2">
        <Panel>
          {items.map((it) => (
            <LineItem key={it.key} it={it} onQty={setQty} onRemove={remove} />
          ))}
        </Panel>
        <OrderSummary
          sub={subtotal}
          cta={t('checkout')}
          onCta={() => navigate('/checkout')}
        />
      </div>
    </div>
  )
}
