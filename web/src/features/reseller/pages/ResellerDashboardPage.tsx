import { useEffect, useState } from 'react'
import type { ReactNode } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Icon, ImageArt, Stepper, Button, Panel, Badge, useToast } from '@/components'
import { useUiStore } from '@/stores/ui'
import { useCurrencyStore } from '@/stores/currency'
import { fmtPrice } from '@/lib/utils'
import { fromPrice } from '@/lib/pricing'
import { DEMO, BEST } from '@/lib/mock/demo'

function Stat({ l, v, sub }: { l: string; v: ReactNode; sub: string }) {
  return (
    <div className="stat" style={{ borderTop: '3px solid var(--agent)' }}>
      <span className="eyebrow">{l}</span>
      <div className="h1 num" style={{ marginTop: 8 }}>
        {v}
      </div>
      <span className="tiny faint">{sub}</span>
    </div>
  )
}

export default function ResellerDashboardPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const toast = useToast()
  const { currency } = useCurrencyStore()

  useEffect(() => {
    useUiStore.getState().setAgent(true)
  }, [])

  const bulk = BEST.slice(0, 6)
  const [cart, setCart] = useState<Record<string, number>>({})
  const totalUnits = Object.values(cart).reduce((a, b) => a + b, 0)
  const totalCost = bulk.reduce((s, p) => s + (cart[p.id] || 0) * fromPrice(p, true), 0)

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
              <Badge
                variant="agent"
                style={{ background: 'rgba(255,255,255,.16)', color: '#ffd76b' }}
              >
                {t('agent_tier')}
              </Badge>
            </span>
            <span className="small" style={{ opacity: 0.85 }}>
              {DEMO.user.name} · {t('agent_pricing')}
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

      <div className="statgrid" style={{ gridTemplateColumns: 'repeat(4,1fr)', marginBottom: 24 }}>
        <div
          className="stat bigbal"
          style={{ background: 'linear-gradient(135deg,#3a2a0c,#5a3f0e)' }}
        >
          <span className="eyebrow" style={{ color: 'rgba(255,255,255,.8)' }}>
            {t('agent_balance')}
          </span>
          <div className="h1 num" style={{ color: '#fff', margin: '8px 0 12px' }}>
            {fmtPrice(DEMO.agentBalance, currency)}
          </div>
          <Button variant="gold" size="sm">
            {t('topup')}
          </Button>
        </div>
        <Stat l={t('margin')} v="14.2%" sub={t('agent_pricing')} />
        <Stat l={t('this_month')} v="2,481" sub={t('orders_n')} />
        <Stat l={t('limits')} v={fmtPrice(10000, currency)} sub="68% used" />
      </div>

      {/* bulk order */}
      <Panel>
        <div className="row between" style={{ marginBottom: 16 }}>
          <h3 className="h3">{t('bulk_order')}</h3>
          <Badge variant="agent">
            <Icon name="shield" size={12} />
            {t('agent_pricing')}
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
            <span style={{ width: 140, textAlign: 'center' }}>{t('quantity')}</span>
          </div>
          {bulk.map((p) => {
            const retail = fromPrice(p, false)
            const ag = fromPrice(p, true)
            const q = cart[p.id] || 0
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
                <div style={{ width: 140, display: 'flex', justifyContent: 'center' }}>
                  <Stepper value={q} set={(v) => setCart((c) => ({ ...c, [p.id]: v }))} min={0} />
                </div>
              </div>
            )
          })}
        </div>
        <hr className="divider" style={{ margin: '16px 0' }} />
        <div className="row between wrap-gap" style={{ gap: 14 }}>
          <span className="muted">
            {totalUnits} {t('orders_n')} ·{' '}
            <b className="num" style={{ color: 'var(--text)' }}>
              {fmtPrice(totalCost, currency)}
            </b>
          </span>
          <Button
            variant="gold"
            size="lg"
            disabled={!totalUnits}
            style={!totalUnits ? { opacity: 0.5 } : {}}
            onClick={() => {
              toast(`${totalUnits} ${t('orders_n')}`, 'check')
            }}
          >
            <Icon name="bolt" size={18} />
            {t('bulk_order')} · {fmtPrice(totalCost, currency)}
          </Button>
        </div>
      </Panel>
    </div>
  )
}
