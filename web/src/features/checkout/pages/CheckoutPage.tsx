import { useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { Icon, Price, Button, Panel, Badge, useToast } from '@/components'
import { LineItem } from '@/features/checkout/components/LineItem'
import { PromoField } from '@/features/checkout/components/OrderSummary'
import { fmtPrice } from '@/lib/utils'
import { useCartStore } from '@/stores/cart'
import { useCurrencyStore } from '@/stores/currency'
import { useUiStore } from '@/stores/ui'
import { useLocaleStore } from '@/stores/locale'
import { useWallet } from '@/features/wallet/hooks/useWallet'
import { usePlaceOrder, OrderError } from '@/features/orders/hooks/usePlaceOrder'
import { adaptOrder } from '@/features/orders/lib/adaptOrder'
import type { PlaceOrderInput } from '@/types'

// Wallet is the only live payment method (card/usdt were mock-approved and
// are disabled until a real gateway exists — mirrors the API's validation).

export default function CheckoutPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const toast = useToast()
  const cartItems = useCartStore((s) => s.items)
  const currency = useCurrencyStore((s) => s.currency)
  const locale = useLocaleStore((s) => s.locale)
  const agent = useUiStore((s) => s.agent)
  const walletQuery = useWallet()
  const placeOrder = usePlaceOrder()

  const balance = walletQuery.data?.balance ?? 0
  const sub = cartItems.reduce((s, x) => s + x.price * x.qty, 0)
  const total = sub

  const insufficient = balance < total
  // One idempotency key per checkout attempt; held in a ref so React re-renders
  // while the request is in flight don't regenerate it (React-Query retries reuse it).
  const keyRef = useRef<string | null>(null)

  const empty = cartItems.length === 0

  const pay = (): void => {
    if (empty || placeOrder.isPending) return

    const input: PlaceOrderInput = {
      items: cartItems.map((it) => ({
        productId: it.id,
        variantId: it.variantId,
        qty: it.qty,
        playerId: it.pid || undefined,
        recipient: it.recipient ?? undefined,
      })),
      currency: 'USD', // prices are USD; display currency is applied at render time
      paymentMethod: 'wallet',
    }

    // Fresh key per click (a corrective re-submit after an error is a new attempt).
    keyRef.current = crypto.randomUUID()

    placeOrder.mutate(
      { input, idempotencyKey: keyRef.current },
      {
        onSuccess: (order) => {
          const view = adaptOrder(order, locale)
          useCartStore.getState().clear()
          navigate('/order-success/' + order.id, { state: { order: view } })
        },
        onError: (err) => {
          if (err instanceof OrderError && err.code === 'KYC_REQUIRED') {
            toast('Identity verification is required — please verify in the SalehCard app.', 'user')
            return
          }
          toast(err.message || t('failed_title'), 'user')
        },
      },
    )
  }


  if (empty) {
    return (
      <div className="wrap" style={{ padding: '26px 0 50px' }}>
        <h1 className="h1" style={{ marginBottom: 22 }}>
          {t('checkout')}
        </h1>
        <Panel style={{ textAlign: 'center', padding: 40 }}>
          <p className="muted" style={{ marginBottom: 16 }}>
            {t('cart_empty')}
          </p>
          <Button variant="primary" onClick={() => navigate('/')}>
            {t('continue_shop')}
          </Button>
        </Panel>
      </div>
    )
  }

  const placing = placeOrder.isPending

  return (
    <div className="wrap" style={{ padding: '26px 0 50px' }}>
      <div className="row" style={{ gap: 10, marginBottom: 16 }}>
        <a className="small clickable faint" onClick={() => navigate('/cart')}>
          {t('cart_title')}
        </a>
        <span className="faint">/</span>
        <span className="small" style={{ fontWeight: 700 }}>
          {t('checkout')}
        </span>
      </div>
      <h1 className="h1" style={{ marginBottom: 22 }}>
        {t('checkout')}
      </h1>
      <div className="cols2">
        <div className="col" style={{ gap: 22 }}>
          {/* methods */}
          <Panel>
            <div className="label" style={{ marginBottom: 14 }}>
              {t('pay_method')}
            </div>
            <div className="col" style={{ gap: 12 }}>
              <div className="method on">
                <span className="mi" style={{ background: 'var(--grad)' }}>
                  <Icon name="wallet" size={18} />
                </span>
                <div className="col" style={{ gap: 2, flex: 1 }}>
                  <span className="row" style={{ gap: 8 }}>
                    <span style={{ fontWeight: 800 }}>{t('pay_wallet')}</span>
                    <Badge variant="instant" style={{ fontSize: 10 }}>
                      {t('recommended')}
                    </Badge>
                  </span>
                  <span className="tiny faint num">
                    {t('balance')}: {fmtPrice(balance, currency)}
                  </span>
                </div>
                <span
                  className="icon-btn"
                  style={{
                    width: 26,
                    height: 26,
                    background: 'var(--grad)',
                    color: '#fff',
                    border: 0,
                  }}
                >
                  <Icon name="check" size={14} />
                </span>
              </div>
            </div>

            {insufficient && (
              <Panel
                style={{
                  marginTop: 14,
                  background: 'var(--grad-soft)',
                  border: '1px solid var(--border-strong)',
                }}
              >
                <div className="row between">
                  <span className="small" style={{ fontWeight: 700 }}>
                    {t('insufficient')}
                  </span>
                  <Button variant="cyan" size="sm" onClick={() => navigate('/wallet')}>
                    {t('topup')}
                  </Button>
                </div>
              </Panel>
            )}
          </Panel>

          {/* items recap */}
          <Panel>
            {cartItems.map((it) => (
              <LineItem key={it.key} it={it} />
            ))}
          </Panel>
        </div>

        {/* summary */}
        <Panel
          style={{
            display: 'flex',
            flexDirection: 'column',
            gap: 14,
            position: 'sticky',
            top: 90,
          }}
        >
          <h3 className="h3">{t('order_summary')}</h3>
          <PromoField />
          <hr className="divider" />
          <div className="row between">
            <span className="muted">{t('subtotal')}</span>
            <Price usd={sub} cur={currency} className="num" />
          </div>
          {agent && (
            <div className="row between" style={{ color: 'var(--agent)' }}>
              <span className="small">{t('agent_pricing')}</span>
              <Icon name="shield" size={14} />
            </div>
          )}
          <hr className="divider" />
          <div className="row between">
            <span style={{ fontWeight: 800, fontSize: 18 }}>{t('total')}</span>
            <Price usd={total} cur={currency} className="display-l" />
          </div>
          <Button
            variant="primary"
            size="lg"
            block
            onClick={pay}
            disabled={insufficient || placing}
          >
            <Icon name="shield" size={18} />
            {placing ? t('processing') : t('place_order')}
          </Button>
          <div className="row center" style={{ gap: 8, color: 'var(--ok)' }}>
            <Icon name="bolt" size={14} />
            <span className="tiny" style={{ fontWeight: 700 }}>
              {t('trust_instant')} · {t('secure_vault')}
            </span>
          </div>
        </Panel>
      </div>
    </div>
  )
}
