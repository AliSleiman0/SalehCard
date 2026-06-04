import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Icon, ImageArt, LoadingSpinner, ErrorState, EmptyState } from '@/components'
import { CATEGORIES, type MockCategory } from '@/lib/mock/demo'
import { fmtPrice } from '@/lib/utils'
import { useCurrencyStore } from '@/stores/currency'
import { useLocaleStore } from '@/stores/locale'
import { useProducts } from '../hooks/useProducts'
import { adaptProduct } from '../lib/adaptProduct'
import { ProductCard } from '../components/ProductCard'
import { ProductGrid } from '../components/ProductGrid'

function CategoryTile({ c }: { c: MockCategory }) {
  const navigate = useNavigate()
  const { t } = useTranslation()
  return (
    <div className="cattile hover-pop" onClick={() => navigate('/category/' + c.id)}>
      <ImageArt art={c.art} word={t(c.key)} sub={`${c.count}+`} h={120} radius={0} wordSize={22} />
      <div className="cap">
        <div style={{ fontWeight: 800, fontSize: 15 }}>{t(c.key)}</div>
        <div className="tiny faint">{t(c.tagKey)}</div>
      </div>
    </div>
  )
}

function SectionHead({ title, onMore }: { title: string; onMore?: () => void }) {
  const { t } = useTranslation()
  return (
    <div className="section-head">
      <h2 className="h2">{title}</h2>
      {onMore && (
        <a className="small clickable" style={{ fontWeight: 700, color: 'var(--brand-1)' }} onClick={onMore}>
          {t('view_all')} →
        </a>
      )}
    </div>
  )
}

export default function HomePage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const cur = useCurrencyStore((s) => s.currency)
  const locale = useLocaleStore((s) => s.locale)
  const query = useProducts({ limit: 20 })

  const products = (query.data?.data ?? []).map((p) => adaptProduct(p, locale))
  const firstId = products[0]?.id

  const trust = [
    { icon: 'bolt' as const, c: 'var(--grad)', l: t('trust_instant') },
    { icon: 'shield' as const, c: 'var(--grad-cyan)', l: t('trust_secure') },
    { icon: 'wallet' as const, c: 'linear-gradient(135deg,#ffd76b,#ffae34)', l: t('trust_pay') },
    { icon: 'star' as const, c: 'linear-gradient(135deg,#ff4dd2,#7a2bff)', l: t('trust_rated') },
  ]
  const [l1, l2] = t('hero_title').split('\n')

  return (
    <div className="wrap">
      {/* hero */}
      <section className="hero">
        <div className="hero-card">
          <h1 className="display-l" style={{ marginBottom: 16 }}>
            {l1}
            <br />
            {l2}
          </h1>
          <p style={{ maxWidth: 440, opacity: 0.9, marginBottom: 26, fontSize: 16 }}>{t('hero_sub')}</p>
          <div className="row" style={{ gap: 12 }}>
            <button className="btn btn-cyan btn-lg" onClick={() => navigate('/category/games')}>
              {t('hero_cta')} <Icon name="arrow" size={18} />
            </button>
            <button
              className="btn btn-lg"
              style={{ background: 'rgba(255,255,255,.14)', color: '#fff' }}
              onClick={() => navigate(firstId ? '/product/' + firstId : '/category/games')}
            >
              {t('hero_cta2')}
            </button>
          </div>
        </div>
        <div className="hero-side">
          <div className="promo-tile" style={{ background: 'radial-gradient(120% 120% at 100% 0%, #22e3c8, #0b8f9e)' }}>
            <span className="eyebrow" style={{ color: 'rgba(255,255,255,.85)' }}>
              {t('featured')}
            </span>
            <div className="h3" style={{ color: '#fff', marginTop: 4 }}>
              PUBG MOBILE
            </div>
            <div className="row between" style={{ marginTop: 8 }}>
              <span style={{ color: '#fff', fontWeight: 800 }}>1800 UC · {fmtPrice(24.99, cur)}</span>
              <button
                className="btn btn-sm"
                style={{ background: '#04121a', color: '#fff' }}
                onClick={() => navigate(firstId ? '/product/' + firstId : '/category/games')}
              >
                {t('buy_now')}
              </button>
            </div>
          </div>
          <div
            className="promo-tile"
            style={{ background: 'radial-gradient(120% 120% at 0% 100%, #d633ff, #5b1aa0)' }}
            onClick={() => navigate('/wallet')}
          >
            <span className="eyebrow" style={{ color: 'rgba(255,255,255,.85)' }}>
              {t('nav_wallet')}
            </span>
            <div className="h3" style={{ color: '#fff', marginTop: 4 }}>
              {t('promo_title')}
            </div>
            <p className="small" style={{ color: 'rgba(255,255,255,.85)', marginTop: 6 }}>
              {t('promo_sub')}
            </p>
          </div>
        </div>
      </section>

      {/* trust */}
      <div className="trustrow">
        {trust.map((x, i) => (
          <div className="trust" key={i}>
            <span className="ti" style={{ background: x.c, color: '#fff' }}>
              <Icon name={x.icon} size={18} />
            </span>
            <span className="small" style={{ fontWeight: 700 }}>
              {x.l}
            </span>
          </div>
        ))}
      </div>

      {/* categories */}
      <section className="section">
        <SectionHead title={t('shop_cat')} />
        <div className="catgrid">
          {CATEGORIES.slice(0, 4).map((c) => (
            <CategoryTile key={c.id} c={c} />
          ))}
        </div>
        <div className="catgrid" style={{ marginTop: 16 }}>
          {CATEGORIES.slice(4).map((c) => (
            <CategoryTile key={c.id} c={c} />
          ))}
        </div>
      </section>

      {/* best sellers */}
      <section className="section">
        <SectionHead title={t('best')} onMore={() => navigate('/category/games')} />
        {query.isLoading ? (
          <LoadingSpinner />
        ) : query.isError ? (
          <ErrorState title={t('retry')} onRetry={() => query.refetch()} retryLabel={t('retry')} />
        ) : products.length === 0 ? (
          <EmptyState icon="grid" title={t('no_results')} sub={t('no_results_sub')} />
        ) : (
          <div className="scroller">
            {products.map((p) => (
              <ProductCard key={p.id} p={p} compact />
            ))}
          </div>
        )}
      </section>

      {/* wallet promo banner */}
      <section className="section">
        <div
          className="card"
          style={{
            background:
              'radial-gradient(120% 160% at 100% 0%, rgba(214,51,255,.45), transparent 55%), linear-gradient(135deg,#171042,#2a1466)',
            border: 0,
            color: '#fff',
            padding: 36,
            display: 'flex',
            gap: 24,
            alignItems: 'center',
            flexWrap: 'wrap',
            justifyContent: 'space-between',
          }}
        >
          <div style={{ maxWidth: 520 }}>
            <span className="eyebrow" style={{ color: 'rgba(255,255,255,.8)' }}>
              <Icon name="wallet" size={13} /> {t('nav_wallet')}
            </span>
            <h2 className="h2" style={{ color: '#fff', margin: '10px 0' }}>
              {t('promo_title')}
            </h2>
            <p style={{ opacity: 0.9 }}>{t('promo_sub')}</p>
          </div>
          <button className="btn btn-cyan btn-lg" onClick={() => navigate('/wallet')}>
            {t('topup')} <Icon name="arrow" size={18} />
          </button>
        </div>
      </section>

      {/* featured grid */}
      {products.length > 0 && (
        <section className="section">
          <SectionHead title={t('featured')} onMore={() => navigate('/category/giftcards')} />
          <ProductGrid products={products.slice(0, 4)} />
        </section>
      )}
    </div>
  )
}
