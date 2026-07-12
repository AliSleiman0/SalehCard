import { useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { useQueries } from '@tanstack/react-query'
import { Icon, Price, Button, Panel, Badge, useToast } from '@/components'
import { LineItem } from '@/features/checkout/components/LineItem'
import { PromoField } from '@/features/checkout/components/OrderSummary'
import { DynamicField } from '@/features/checkout/components/DynamicField'
import { nonQuantityFields, buildOrderLine, isValidLebaneseMobile } from '@/features/checkout/lib/orderFields'
import { fetchProduct } from '@/features/catalog/api/products'
import { fmtPrice } from '@/lib/utils'
import { useCartStore } from '@/stores/cart'
import { useCurrencyStore } from '@/stores/currency'
import { useUiStore } from '@/stores/ui'
import { useLocaleStore } from '@/stores/locale'
import { useWallet } from '@/features/wallet/hooks/useWallet'
import { usePlaceOrder, OrderError } from '@/features/orders/hooks/usePlaceOrder'
import { adaptOrder } from '@/features/orders/lib/adaptOrder'
import type { PlaceOrderInput, Product } from '@/types'

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

  // Full product specs per cart line (reuses the PDP's ['products', id] cache) so
  // we can render the per-product dynamic delivery fields and rebuild order lines.
  const productQueries = useQueries({
    queries: cartItems.map((it) => ({
      queryKey: ['products', it.id],
      queryFn: () => fetchProduct(it.id),
      enabled: !!it.id,
    })),
  })
  const productFor = (id: string): Product | undefined =>
    productQueries.map((q) => q.data?.data).find((p) => p?.id === id)

  // Collected delivery-field values, keyed `${item.key}|${field.key}`.
  const [fieldValues, setFieldValues] = useState<Record<string, string>>({})
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({})

  // The first field is prefilled from the cart line's pid (chosen on the PDP).
  const valueFor = (itemKey: string, fieldKey: string, index: number, pid?: string): string => {
    const k = `${itemKey}|${fieldKey}`
    if (fieldValues[k] !== undefined) return fieldValues[k]
    return index === 0 ? (pid ?? '') : ''
  }
  const setValue = (itemKey: string, fieldKey: string, v: string) =>
    setFieldValues((prev) => ({ ...prev, [`${itemKey}|${fieldKey}`]: v }))

  // Cart lines that need delivery-field input (non-code product with schema fields).
  const needing = cartItems.filter((it) => {
    const p = productFor(it.id)
    return p && p.fulfillmentType !== 'code' && nonQuantityFields(p).length > 0
  })

  const pay = (): void => {
    if (empty || placeOrder.isPending) return

    // Validate every dynamic field, then build order lines from the collected values.
    const errors: Record<string, string> = {}
    for (const it of needing) {
      const p = productFor(it.id)!
      nonQuantityFields(p).forEach((f, i) => {
        const val = valueFor(it.key, f.key, i, it.pid).trim()
        if (!val) errors[`${it.key}|${f.key}`] = t('field_required')
        else if (f.key === 'phone' && !isValidLebaneseMobile(val))
          errors[`${it.key}|${f.key}`] = t('invalid_phone')
      })
    }
    if (Object.keys(errors).length > 0) {
      setFieldErrors(errors)
      toast(t('field_required'), 'user')
      return
    }
    setFieldErrors({})

    const items = cartItems.map((it) => {
      const p = productFor(it.id)
      const itemValues: Record<string, string> = {}
      nonQuantityFields(p).forEach((f, i) => {
        itemValues[f.key] = valueFor(it.key, f.key, i, it.pid)
      })
      return buildOrderLine(it, p, itemValues)
    })

    const input: PlaceOrderInput = {
      items,
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

          {/* delivery details (per-product dynamic fields) */}
          {needing.length > 0 && (
            <Panel>
              <div className="label" style={{ marginBottom: 14 }}>
                {t('delivery_details')}
              </div>
              <div className="col" style={{ gap: 18 }}>
                {needing.map((it) => {
                  const p = productFor(it.id)!
                  const fields = nonQuantityFields(p)
                  return (
                    <div key={it.key} className="col" style={{ gap: 12 }}>
                      {needing.length > 1 && (
                        <span className="small" style={{ fontWeight: 700, color: 'var(--text-dim)' }}>
                          {it.brand} · {it.variant}
                        </span>
                      )}
                      {fields.map((f, i) => (
                        <DynamicField
                          key={f.key}
                          field={f}
                          value={valueFor(it.key, f.key, i, it.pid)}
                          onChange={(v) => setValue(it.key, f.key, v)}
                          error={fieldErrors[`${it.key}|${f.key}`]}
                        />
                      ))}
                    </div>
                  )
                })}
              </div>
            </Panel>
          )}

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
