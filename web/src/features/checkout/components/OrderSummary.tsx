import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Icon, Price, Button, useToast } from '@/components'
import { fmtPrice } from '@/lib/utils'
import { useCurrencyStore } from '@/stores/currency'

export function PromoField() {
  const { t } = useTranslation()
  const toast = useToast()
  const [code, setCode] = useState('')
  const [ok, setOk] = useState(false)
  return (
    <div>
      <div className="row" style={{ gap: 8 }}>
        <input
          className="field"
          placeholder={t('promo_code')}
          value={code}
          onChange={(e) => setCode(e.target.value.toUpperCase())}
          style={{ padding: '11px 14px' }}
        />
        <Button
          variant="ghost"
          onClick={() => {
            if (code) {
              setOk(true)
              toast(t('promo_ok'), 'check')
            }
          }}
        >
          {t('apply')}
        </Button>
      </div>
      {ok && (
        <div className="row" style={{ gap: 6, marginTop: 8, color: 'var(--ok)' }}>
          <Icon name="check" size={14} />
          <span className="tiny" style={{ fontWeight: 700 }}>
            {t('promo_ok')}: {code}
          </span>
        </div>
      )}
    </div>
  )
}

export function OrderSummary({
  sub,
  cta,
  onCta,
  disc = 0,
}: {
  sub: number
  cta: string
  onCta: () => void
  disc?: number
}) {
  const { t } = useTranslation()
  const cur = useCurrencyStore((s) => s.currency)
  const total = sub - disc
  return (
    <div
      className="panel card-pad"
      style={{ display: 'flex', flexDirection: 'column', gap: 14, position: 'sticky', top: 90 }}
    >
      <h3 className="h3">{t('order_summary')}</h3>
      <PromoField />
      <hr className="divider" />
      <div className="row between">
        <span className="muted">{t('subtotal')}</span>
        <Price usd={sub} cur={cur} className="num" />
      </div>
      {disc > 0 && (
        <div className="row between" style={{ color: 'var(--ok)' }}>
          <span>{t('discount')}</span>
          <span className="num">−{fmtPrice(disc, cur)}</span>
        </div>
      )}
      <hr className="divider" />
      <div className="row between">
        <span style={{ fontWeight: 800, fontSize: 18 }}>{t('total')}</span>
        <Price usd={total} cur={cur} className="display-l" />
      </div>
      <Button variant="primary" size="lg" block onClick={onCta}>
        {cta} <Icon name="arrow" size={18} />
      </Button>
      <div className="row center" style={{ gap: 8, color: 'var(--text-dim)' }}>
        <Icon name="shield" size={14} />
        <span className="tiny">256-bit secure · {t('trust_instant')}</span>
      </div>
    </div>
  )
}
