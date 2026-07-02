import { useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Icon, ImageArt, Button, Panel, Badge } from '@/components'
import { useUiStore } from '@/stores/ui'
import { useCurrencyStore } from '@/stores/currency'
import { useLocaleStore } from '@/stores/locale'
import { useAuthStore } from '@/stores/auth'
import { fmtPrice } from '@/lib/utils'
import { fromPrice } from '@/lib/pricing'
import { displayName } from '@/features/auth/userDisplay'
import { useWallet } from '@/features/wallet/hooks/useWallet'
import { useOrders } from '@/features/orders/hooks/useOrders'
import { useProducts } from '@/features/catalog/hooks/useProducts'
import { adaptProduct } from '@/features/catalog/lib/adaptProduct'

export default function ResellerDashboardPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { currency } = useCurrencyStore()
  const locale = useLocaleStore((s) => s.locale)
  const user = useAuthStore((s) => s.user)

  useEffect(() => {
    useUiStore.getState().setAgent(true)
  }, [])

  const balance = useWallet().data?.balance ?? 0
  const orders = useOrders().data ?? []
  const productsQuery = useProducts({ limit: 20 })
  const products = (productsQuery.data?.data ?? []).map((x) => adaptProduct(x, locale))

  return (
    <div className="wrap" style={{ padding: '26px 0 50px' }}>
      {/* agent banner */}
      <div
        className="card card-pad"
        style={{
          background:
            'radial-gradient(120% 160% at 100% 0%, rgba(255,176,46,.4), transparent 55%), linear-gradient(135deg,#2a1e08,#3a2a0c)',
          border: '1px solid rgba(255,176,46,.3)',
          color: '#fff',
          marginBottom: 24,
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          flexWrap: 'wrap',
          gap: 16,
        }}
      >
        <div className="row" style={{ gap: 16 }}>
          <span
            style={{
              width: 54,
              height: 54,
              borderRadius: 16,
              background: 'linear-gradient(135deg,#ffd76b,#ffae34)',
              display: 'grid',
              placeItems: 'center',
              color: '#2a1b00',
            }}
          >
            <Icon name="shield" size={26} />
          </span>
          <div className="col" style={{ gap: 4 }}>
            <span className="row" style={{ gap: 10 }}>
              <h1 className="h2" style={{ color: '#fff' }}>
                {t('agent_dash')}
              </h1>
              <Badge variant="agent" style={{ background: 'rgba(255,255,255,.16)', color: '#ffd76b' }}>
                {user?.resellerTier || t('agent_tier')}
              </Badge>
            </span>
            <span className="small" style={{ opacity: 0.85 }}>
              {displayName(user)} · {t('agent_pricing')}
            </span>
          </div>
        </div>
        <Button
          style={{ background: 'rgba(255,255,255,.16)', color: '#fff' }}
          onClick={() => {
            useUiStore.getState().setAgent(false)
            navigate('/dashboard')
          }}
        >
          <Icon name="user" size={16} />
          {t('switch_customer')}
        </Button>
      </div>

      <div className="statgrid" style={{ gridTemplateColumns: 'repeat(2,1fr)', marginBottom: 24 }}>
        <div className="stat bigbal" style={{ background: 'linear-gradient(135deg,#3a2a0c,#5a3f0e)' }}>
          <span className="eyebrow" style={{ color: 'rgba(255,255,255,.8)' }}>
            {t('current_balance')}
          </span>
          <div className="h1 num" style={{ color: '#fff', margin: '8px 0 12px' }}>
            {fmtPrice(balance, currency)}
          </div>
          <Button variant="gold" size="sm" onClick={() => navigate('/wallet')}>
            {t('topup')}
          </Button>
        </div>
        <div className="stat" style={{ borderTop: '3px solid var(--agent)' }}>
          <span className="eyebrow">{t('orders')}</span>
          <div className="h1 num" style={{ marginTop: 8 }}>
            {orders.length}
          </div>
          <span className="tiny faint">{t('orders_n')}</span>
        </div>
      </div>

      {/* reseller pricing catalog */}
      <Panel>
        <div className="row between" style={{ marginBottom: 16 }}>
          <h3 className="h3">{t('agent_pricing')}</h3>
          <Badge variant="agent">
            <Icon name="shield" size={12} />
            {t('agent_price')}
          </Badge>
        </div>
        <div className="col" style={{ gap: 0 }}>
          {/* header */}
          <div
            className="lrow desktop-only"
            style={{
              fontWeight: 800,
              fontSize: 12,
              color: 'var(--text-faint)',
              textTransform: 'uppercase',
              letterSpacing: '.08em',
            }}
          >
            <span style={{ flex: 1, paddingInlineStart: 86 }}>Product</span>
            <span style={{ width: 100 }}>Retail</span>
            <span style={{ width: 110 }}>{t('agent_price')}</span>
            <span style={{ width: 120, textAlign: 'center' }} />
          </div>
          {products.length === 0 ? (
            <p className="muted" style={{ padding: '12px 4px' }}>
              {t('empty_state')}
            </p>
          ) : (
            products.map((p) => {
              const retail = fromPrice(p, false)
              const ag = fromPrice(p, true)
              return (
                <div className="lrow" key={p.id}>
                  <ImageArt
                    art={p.art}
                    word={p.brand.split(' ')[0]}
                    h={46}
                    wordSize={12}
                    radius={10}
                    style={{ width: 62, flex: 'none' }}
                  />
                  <div className="col" style={{ gap: 1, flex: 1, minWidth: 0 }}>
                    <span style={{ fontWeight: 700, fontSize: 14 }}>{p.brand}</span>
                    <span className="tiny faint">{p.title}</span>
                  </div>
                  <span className="strike small num desktop-only" style={{ width: 100 }}>
                    {fmtPrice(retail, currency)}
                  </span>
                  <span
                    className="num desktop-only"
                    style={{ width: 110, fontWeight: 800, color: 'var(--agent)' }}
                  >
                    {fmtPrice(ag, currency)}
                  </span>
                  <div style={{ width: 120, display: 'flex', justifyContent: 'center' }}>
                    <Button variant="gold" size="sm" onClick={() => navigate('/product/' + p.id)}>
                      <Icon name="cart" size={15} />
                      {t('order_now')}
                    </Button>
                  </div>
                </div>
              )
            })
          )}
        </div>
      </Panel>
    </div>
  )
}
