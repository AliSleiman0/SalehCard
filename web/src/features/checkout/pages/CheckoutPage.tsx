import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { Icon, Price, Button, Panel, Badge, Input } from '@/components'
import { LineItem } from '@/features/checkout/components/LineItem'
import { PromoField } from '@/features/checkout/components/OrderSummary'
import { fmtPrice } from '@/lib/utils'
import { useCartStore } from '@/stores/cart'
import type { CartItem } from '@/stores/cart'
import { useWalletStore } from '@/stores/wallet'
import { useCurrencyStore } from '@/stores/currency'
import { useUiStore } from '@/stores/ui'
import type { OrderView } from '@/features/orders/types'

type Method = 'wallet' | 'visa' | 'usdt'

function genCode(brand: string): string {
  const seg = (): string => Math.random().toString(36).slice(2, 6).toUpperCase()
  const pre = (brand.split(' ')[0] || 'SC').slice(0, 4).toUpperCase()
  return `${pre}-${seg()}-${seg()}-${seg()}`
}

const DEMO_ITEM: CartItem = {
  key: 'demo',
  id: 'pubg-uc',
  brand: 'PUBG MOBILE',
  title: 'UC Top-up',
  art: 'battle',
  variant: '1800 UC',
  price: 24.99,
  qty: 1,
  pid: '5129384761',
  fulfill: 'credit',
}

export default function CheckoutPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const cartItems = useCartStore((s) => s.items)
  const balance = useWalletStore((s) => s.balance)
  const currency = useCurrencyStore((s) => s.currency)
  const agent = useUiStore((s) => s.agent)

  const items = cartItems.length ? cartItems : [DEMO_ITEM]
  const sub = items.reduce((s, x) => s + x.price * x.qty, 0)
  const total = sub

  const [method, setMethod] = useState<Method>('wallet')
  const insufficient = method === 'wallet' && balance < total

  const pay = (): void => {
    if (method === 'wallet' && !insufficient) useWalletStore.getState().charge(total)
    const it0 = items[0]
    const fulfill: OrderView['fulfill'] = it0.fulfill || 'code'
    const now = new Date()
    const stamp =
      now.toLocaleDateString('en-US', { month: 'short', day: 'numeric' }) +
      ' · ' +
      now.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit' })

    const order: OrderView = {
      id: 'SC-' + Math.floor(90000 + Math.random() * 9999),
      product: `${it0.brand} — ${it0.variant}${items.length > 1 ? ` +${items.length - 1}` : ''}`,
      art: it0.art,
      total,
      method: method === 'wallet' ? 'Wallet' : method === 'visa' ? 'Visa' : 'USDT',
      fulfill,
    }

    if (fulfill === 'code') {
      order.code = genCode(it0.brand)
      order.pin = it0.brand.includes('PUBG') ? '4471' : ''
      order.status = 'delivered'
    } else if (fulfill === 'credit') {
      order.account = it0.pid || '—'
      order.amount = it0.variant
      order.ts = stamp
      order.status = 'delivered'
    } else {
      order.ref =
        'MT-' +
        Math.floor(1000 + Math.random() * 8999) +
        '-' +
        Math.floor(1000 + Math.random() * 8999)
      order.recipient = it0.recipient || { name: 'Ayşe Yılmaz', country: 'Türkiye', detail: '' }
      order.status = 'processing'
      order.steps = [
        { k: 'submitted', ts: stamp },
        { k: 'processing', ts: stamp },
        { k: 'completed', ts: '' },
      ]
    }

    useCartStore.getState().clear()
    navigate('/order-success/' + order.id, { state: { order } })
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
            {items.map((it) => (
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
          <Button variant="primary" size="lg" block onClick={pay} disabled={insufficient}>
            <Icon name="shield" size={18} />
            {method === 'wallet' ? t('place_order') : `${t('pay')} ${fmtPrice(total, currency)}`}
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
