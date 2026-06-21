import { useMemo, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Icon, LoadingSpinner, ErrorState, EmptyState, Segmented } from '@/components'
import { rootMeta } from '@/lib/categoryPresentation'
import { fmtPrice } from '@/lib/utils'
import { fromPrice } from '@/lib/pricing'
import { useUiStore } from '@/stores/ui'
import { useLocaleStore } from '@/stores/locale'
import { useProducts } from '../hooks/useProducts'
import { useCategories } from '../hooks/useCategories'
import { adaptProduct } from '../lib/adaptProduct'
import { adaptRootCategory } from '../lib/adaptCategory'
import { ProductGrid } from '../components/ProductGrid'

type Sort = 'pop' | 'low' | 'high' | 'rating'

export default function CategoryPage() {
  const { slug } = useParams<{ slug: string }>()
  // The :slug is a rootDomain (games, app_topups, …); the storefront browses a
  // whole top-level domain at once.
  const cat = slug ?? 'games'
  const { t } = useTranslation()
  const navigate = useNavigate()
  const agent = useUiStore((s) => s.agent)
  const locale = useLocaleStore((s) => s.locale)

  const [sort, setSort] = useState<Sort>('pop')
  const [maxP, setMaxP] = useState(110)

  const query = useProducts({ rootDomain: cat, limit: 50 })
  const view = (useCategories({ depth: 0 }).data?.data ?? [])
    .map((x) => adaptRootCategory(x, locale))
    .find((v) => v.key === cat)
  const title = view?.name || rootMeta(cat).label || cat

  const all = useMemo(
    () => (query.data?.data ?? []).map((p) => adaptProduct(p, locale)),
    [query.data, locale],
  )

  const items = useMemo(() => {
    let r = all.filter((p) => fromPrice(p, agent) <= maxP)
    if (sort === 'low') r = [...r].sort((a, b) => fromPrice(a, agent) - fromPrice(b, agent))
    if (sort === 'high') r = [...r].sort((a, b) => fromPrice(b, agent) - fromPrice(a, agent))
    if (sort === 'rating') r = [...r].sort((a, b) => b.rating - a.rating)
    return r
  }, [all, sort, maxP, agent])

  const brands = [...new Set(all.map((p) => p.brand))].slice(0, 6)

  return (
    <div className="wrap" style={{ paddingTop: 18, paddingBottom: 40 }}>
      <div className="row" style={{ gap: 10, marginBottom: 16 }}>
        <a className="small clickable faint" onClick={() => navigate('/')}>
          {t('nav_home')}
        </a>
        <span className="faint">/</span>
        <span className="small" style={{ fontWeight: 700 }}>
          {title}
        </span>
      </div>
      <div className="row between wrap-gap" style={{ marginBottom: 22 }}>
        <div>
          <h1 className="h1">{title}</h1>
          <p className="muted small">
            {items.length} {t('results')}
            {view?.tag ? ` · ${view.tag}` : ''}
          </p>
        </div>
        <Segmented<Sort>
          plain
          value={sort}
          onChange={setSort}
          options={[
            { value: 'pop', label: t('sort_pop') },
            { value: 'low', label: t('sort_low') },
            { value: 'high', label: t('sort_high') },
            { value: 'rating', label: t('sort_rating') },
          ]}
        />
      </div>

      <div className="cols-acct">
        {/* filters */}
        <aside className="panel card-pad sidenav" style={{ gap: 18, display: 'block' }}>
          <div className="row between" style={{ marginBottom: 14 }}>
            <span className="row" style={{ gap: 8, fontWeight: 800 }}>
              <Icon name="filter" size={16} />
              {t('filters')}
            </span>
            <a
              className="tiny clickable"
              style={{ color: 'var(--brand-1)', fontWeight: 700 }}
              onClick={() => {
                setMaxP(110)
                setSort('pop')
              }}
            >
              {t('clear')}
            </a>
          </div>
          <div className="label">
            {t('price_range')}: ≤ {fmtPrice(maxP, 'USD')}
          </div>
          <input
            type="range"
            min="1"
            max="110"
            value={maxP}
            onChange={(e) => setMaxP(+e.target.value)}
            style={{ width: '100%', accentColor: 'var(--brand-1)' }}
          />
          <hr className="divider" style={{ margin: '18px 0' }} />
          <div className="label">{t('platform')}</div>
          <div className="row wrap-gap" style={{ gap: 8 }}>
            {brands.map((b) => (
              <span key={b} className="badge badge-soft clickable">
                {b}
              </span>
            ))}
          </div>
          <hr className="divider" style={{ margin: '18px 0' }} />
          <div className="row" style={{ gap: 8 }}>
            <span className="badge badge-instant">
              <Icon name="bolt" size={11} />
              {t('instant')}
            </span>
            <span className="badge badge-secure">
              <Icon name="shield" size={11} />
              {t('secure_vault')}
            </span>
          </div>
        </aside>

        {query.isLoading ? (
          <LoadingSpinner />
        ) : query.isError ? (
          <ErrorState title={t('retry')} onRetry={() => query.refetch()} retryLabel={t('retry')} />
        ) : items.length === 0 ? (
          <EmptyState icon="search" title={t('no_results')} sub={t('no_results_sub')} />
        ) : (
          <ProductGrid products={items} />
        )}
      </div>
    </div>
  )
}
