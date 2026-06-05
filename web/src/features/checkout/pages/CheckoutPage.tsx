import { useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { Icon, Price, Button, Panel, Badge, Input, useToast } from '@/components'
import { LineItem } from '@/features/checkout/components/LineItem'
import { PromoField } from '@/features/checkout/components/OrderSummary'
import { fmtPrice } from '@/lib/utils'
import { useCartStore } from '@/stores/cart'
import { useCurrencyStore } from '@/stores/currency'
import { useUiStore } from '@/stores/ui'
import { useLocaleStore } from '@/stores/locale'
import { useWallet } from '@/features/wallet/hooks/useWallet'
import { usePlaceOrder } from '@/features/orders/hooks/usePlaceOrder'
import { adaptOrder } from '@/features/orders/lib/adaptOrder'
import type { PaymentMethod, PlaceOrderInput } from '@/types'

type Method = 'wallet' | 'visa' | 'usdt'

// methodToPayment maps the UI's method labels to the API's payment methods
// ('visa' is the card path).
const methodToPayment: Record<Method, PaymentMethod> = {
  wallet: 'wallet',
  visa: 'card',
  usdt: 'usdt',
}

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

  const [method, setMethod] = useState<Method>('wallet')
  const insufficient = method === 'wallet' && balance < total
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
      paymentMethod: methodToPayment[method],
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
          toast(err.message || t('failed_title'), 'user')
        },
      },
    )
  }

  const methods: {
    k: Method
    icon: 'wallet' | null
    c: string
    l: string
    tag?: string
    badge?: string
    sub: string
  }[] = [
    {
      k: 'wallet',
      icon: 'wallet',
      c: 'var(--grad)',
      l: t('pay_wallet'),
      tag: t('recommended'),
      sub: `${t('balance')}: ${fmtPrice(balance, currency)}`,
    },
    { k: 'visa', icon: null, c: '#1a1f71', l: t('pay_visa'), badge: 'VISA', sub: '•••• 4242' },
    { k: 'usdt', icon: null, c: '#26a17b', l: t('pay_usdt'), badge: '₮', sub: 'TRC-20 / ERC-20' },
  ]

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
              {methods.map((m) => (
                <div
                  key={m.k}
                  className={'method' + (method === m.k ? ' on' : '')}
                  onClick={() => setMethod(m.k)}
                >
                  <span className="mi" style={{ background: m.c }}>
                    {m.icon ? <Icon name={m.icon} size={18} /> : m.badge}
                  </span>
                  <div className="col" style={{ gap: 2, flex: 1 }}>
                    <span className="row" style={{ gap: 8 }}>
                      <span style={{ fontWeight: 800 }}>{m.l}</span>
                      {m.tag && (
                        <Badge variant="instant" style={{ fontSize: 10 }}>
                          {m.tag}
                        </Badge>
                      )}
                    </span>
                    <span className="tiny faint num">{m.sub}</span>
                  </div>
                  <span
                    className="icon-btn"
                    style={{
                      width: 26,
                      height: 26,
                      background: method === m.k ? 'var(--grad)' : 'var(--surface-2)',
                      color: '#fff',
                      border: 0,
                    }}
                  >
                    {method === m.k && <Icon name="check" size={14} />}
                  </span>
                </div>
              ))}
            </div>

            {/* method-specific panels */}
            {method === 'wallet' && insufficient && (
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
            {method === 'visa' && (
              <div
                className="grid"
                style={{ marginTop: 16, gridTemplateColumns: '1fr 1fr', gap: 12 }}
              >
                <div style={{ gridColumn: '1/-1' }}>
                  <Input
                    label={t('card_num')}
                    className="num"
                    placeholder="4242 4242 4242 4242"
                  />
                </div>
                <Input label={t('expiry')} placeholder="12 / 28" />
                <Input label={t('cvc')} placeholder="•••" />
                <div style={{ gridColumn: '1/-1' }}>
                  <Input label={t('name_card')} placeholder="Y. Demir" />
                </div>
              </div>
            )}
            {method === 'usdt' && (
              <Panel style={{ marginTop: 16 }}>
                <p className="small muted" style={{ marginBottom: 12 }}>
                  {t('usdt_note')}
                </p>
                <div className="vault" style={{ fontSize: 13 }}>
                  <span
                    className="code"
                    style={{ filter: 'none', overflow: 'hidden', textOverflow: 'ellipsis' }}
                  >
                    TQn9Y2khE...r5v8f29ab
                  </span>
                  <Badge variant="secure">TRC-20</Badge>
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
            {placing
              ? t('processing')
              : method === 'wallet'
                ? t('place_order')
                : `${t('pay')} ${fmtPrice(total, currency)}`}
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
