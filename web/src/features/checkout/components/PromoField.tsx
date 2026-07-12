import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Icon, Button } from '@/components'
import { fmtPrice } from '@/lib/utils'
import { useCurrencyStore } from '@/stores/currency'
import { useValidatePromo } from '../hooks/useValidatePromo'
import type { PromoResult } from '../api/promos'

// PromoField previews a promo code against the order total and reports the
// applied result upward (the order endpoint re-applies it authoritatively).
export function PromoField({
  orderTotal,
  applied,
  onApplied,
}: {
  orderTotal: number
  applied: PromoResult | null
  onApplied: (r: PromoResult | null) => void
}) {
  const { t } = useTranslation()
  const cur = useCurrencyStore((s) => s.currency)
  const validate = useValidatePromo()
  const [code, setCode] = useState('')
  const [error, setError] = useState<string | null>(null)

  const apply = () => {
    const c = code.trim().toUpperCase()
    if (!c) return
    setError(null)
    validate.mutate(
      { code: c, orderTotal },
      {
        onSuccess: (r) => {
          if (r.valid && r.discount > 0) onApplied(r)
          else setError(t('promo_invalid'))
        },
        onError: (err) => setError(err.message || t('promo_invalid')),
      },
    )
  }

  if (applied) {
    return (
      <div className="row between" style={{ color: 'var(--ok)' }}>
        <span className="row" style={{ gap: 6 }}>
          <Icon name="check" size={14} />
          <span className="small" style={{ fontWeight: 700 }}>
            {t('promo_applied')}: {applied.code} (−{fmtPrice(applied.discount, cur)})
          </span>
        </span>
        <a
          className="tiny clickable"
          style={{ fontWeight: 700, color: 'var(--brand-1)' }}
          onClick={() => {
            onApplied(null)
            setCode('')
          }}
        >
          {t('remove')}
        </a>
      </div>
    )
  }

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
        <Button variant="ghost" onClick={apply} loading={validate.isPending}>
          {t('apply')}
        </Button>
      </div>
      {error && (
        <span className="tiny" style={{ color: 'var(--danger)', fontWeight: 600, marginTop: 6, display: 'block' }}>
          {error}
        </span>
      )}
    </div>
  )
}
