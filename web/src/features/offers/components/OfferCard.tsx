import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { ImageArt, Price } from '@/components'
import { useCurrencyStore } from '@/stores/currency'
import type { ViewOffer } from '../lib/adaptOffer'

function endsInLabel(endsAt: string | undefined, t: (k: string, o?: Record<string, unknown>) => string): string | null {
  if (!endsAt) return null
  const ms = new Date(endsAt).getTime() - Date.now()
  if (ms <= 0) return null
  const days = Math.floor(ms / 86400000)
  const hours = Math.floor((ms % 86400000) / 3600000)
  return days > 0 ? t('offer_ends_d', { d: days, h: hours }) : t('offer_ends_h', { h: hours })
}

export function OfferCard({ o }: { o: ViewOffer }) {
  const navigate = useNavigate()
  const { t } = useTranslation()
  const cur = useCurrencyStore((s) => s.currency)
  const ends = endsInLabel(o.endsAt, t)

  return (
    <div className="prodcard hover-pop" onClick={() => navigate('/product/' + o.productId)}>
      <div style={{ position: 'relative' }}>
        <ImageArt art={o.art} src={o.image} word={o.name} h={150} radius={0} />
        <span className="badge badge-disc" style={{ position: 'absolute', top: 10, insetInlineStart: 10 }}>
          {o.label}
        </span>
        {!o.inStock && (
          <span className="badge badge-soft" style={{ position: 'absolute', top: 10, insetInlineEnd: 10 }}>
            {t('out_of_stock')}
          </span>
        )}
      </div>
      <div className="pc-body">
        <div className="h3" style={{ fontSize: 16, lineHeight: 1.15 }}>
          {o.name}
        </div>
        {ends && <span className="tiny faint">{ends}</span>}
        <div className="spacer" />
        <div className="col" style={{ gap: 0 }}>
          <span className="tiny faint">from</span>
          <span className="row" style={{ gap: 7 }}>
            <Price usd={o.now} cur={cur} className="h3" />
            <Price usd={o.was} cur={cur} strike className="small" />
          </span>
        </div>
      </div>
    </div>
  )
}
