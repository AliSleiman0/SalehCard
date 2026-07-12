import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Icon, ImageArt, Stars, Price } from '@/components'
import { useUiStore } from '@/stores/ui'
import { useCurrencyStore } from '@/stores/currency'
import { fromPrice } from '@/lib/pricing'
import type { ViewProduct } from '../lib/adaptProduct'

export function ProductCard({ p, compact = false }: { p: ViewProduct; compact?: boolean }) {
  const navigate = useNavigate()
  const { t } = useTranslation()
  const agent = useUiStore((s) => s.agent)
  const cur = useCurrencyStore((s) => s.currency)
  const base = fromPrice(p, false)
  const eff = fromPrice(p, agent)

  return (
    <div className="prodcard hover-pop" onClick={() => navigate('/product/' + p.id)}>
      <div style={{ position: 'relative' }}>
        <ImageArt art={p.art} src={p.image} word={p.brand} sub={p.title} h={compact ? 124 : 150} radius={0} />
        <div className="row" style={{ position: 'absolute', top: 10, insetInlineStart: 10, gap: 6 }}>
          {p.offer && <span className="badge badge-disc">{p.offer.label}</span>}
          {!p.available ? (
            <span className="badge badge-soft">{t('out_of_stock')}</span>
          ) : (
            p.instant && (
              <span className="badge badge-instant">
                <Icon name="bolt" size={11} />
                {t('instant')}
              </span>
            )
          )}
        </div>
        {agent && (
          <span className="badge badge-agent" style={{ position: 'absolute', top: 10, insetInlineEnd: 10 }}>
            <Icon name="shield" size={11} />
            {t('agent_price')}
          </span>
        )}
      </div>
      <div className="pc-body">
        <div className="row between" style={{ alignItems: 'flex-start', gap: 8 }}>
          <div className="h3" style={{ fontSize: 17, lineHeight: 1.1 }}>
            {p.brand}
            <br />
            <span className="small muted" style={{ fontFamily: 'var(--font-body)' }}>
              {p.title}
            </span>
          </div>
        </div>
        {p.reviews > 0 && (
          <div className="row" style={{ gap: 6 }}>
            <Stars value={p.rating} size={13} />
            <span className="tiny faint num">{p.reviews.toLocaleString()}</span>
          </div>
        )}
        <div className="spacer" />
        <div className="row between" style={{ marginTop: 4 }}>
          <div className="col" style={{ gap: 0 }}>
            <span className="tiny faint">from</span>
            <span className="row" style={{ gap: 7 }}>
              <Price usd={eff} cur={cur} className="h3" />
              {agent && <Price usd={base} cur={cur} strike className="small" />}
              {!agent && p.offer && <Price usd={p.offer.wasFrom} cur={cur} strike className="small" />}
            </span>
          </div>
          <span className="icon-btn" style={{ width: 36, height: 36, background: 'var(--grad)', color: '#fff', border: 0 }}>
            <Icon name="arrow" size={16} />
          </span>
        </div>
      </div>
    </div>
  )
}
