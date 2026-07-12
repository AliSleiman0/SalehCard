import { useMemo, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Icon, ImageArt, Stars, Price, Stepper, LoadingSpinner, ErrorState, useToast } from '@/components'
import { rootMeta } from '@/lib/categoryPresentation'
import { fmtPrice } from '@/lib/utils'
import { priceFor } from '@/lib/pricing'
import { useUiStore } from '@/stores/ui'
import { useCurrencyStore } from '@/stores/currency'
import { useLocaleStore } from '@/stores/locale'
import { useCartStore } from '@/stores/cart'
import { useAuthStore } from '@/stores/auth'
import { useProduct } from '../hooks/useProduct'
import { useProducts } from '../hooks/useProducts'
import { useReviews } from '../hooks/useReviews'
import { adaptProduct } from '../lib/adaptProduct'
import { adaptReview } from '../lib/adaptReview'
import { ProductCard } from '../components/ProductCard'

export default function ProductDetailPage() {
  const { id } = useParams<{ id: string }>()
  const { t } = useTranslation()
  const navigate = useNavigate()
  const toast = useToast()
  const agent = useUiStore((s) => s.agent)
  const cur = useCurrencyStore((s) => s.currency)
  const locale = useLocaleStore((s) => s.locale)
  const addToCart = useCartStore((s) => s.add)
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)
  const savedIds = useAuthStore((s) => s.user?.savedPlayerIds ?? [])

  const query = useProduct(id ?? '')

  const [vi, setVi] = useState(0)
  const [qty, setQty] = useState(1)
  const [pid, setPid] = useState('')
  const [pickId, setPickId] = useState(false)
  const [rName, setRName] = useState('')
  const [country, setCountry] = useState('Türkiye')

  const p = useMemo(
    () => (query.data?.data ? adaptProduct(query.data.data, locale) : null),
    [query.data, locale],
  )

  // approved reviews for this product (real API; aggregate rating stays on p)
  const reviewsQuery = useReviews(id ?? '')
  const reviews = useMemo(
    () => (reviewsQuery.data ?? []).map(adaptReview),
    [reviewsQuery.data],
  )

  // related products from the same category (excluding current)
  const relatedQuery = useProducts({ category: p?.cat, limit: 20 })
  const related = useMemo(
    () =>
      (relatedQuery.data?.data ?? [])
        .map((x) => adaptProduct(x, locale))
        .filter((x) => x.id !== p?.id)
        .slice(0, 6),
    [relatedQuery.data, locale, p?.id],
  )

  if (query.isLoading) return <LoadingSpinner />
  if (query.isError || !p) {
    return (
      <ErrorState
        title={t('no_results')}
        sub={t('no_results_sub')}
        onRetry={() => query.refetch()}
        retryLabel={t('retry')}
      />
    )
  }

  const fulfill = p.fulfill
  const vIndex = Math.min(vi, p.variants.length - 1)
  const v = p.variants[vIndex]
  const unit = priceFor(p, v, agent)
  const total = unit * qty
  const savings = agent ? (v.p - unit) * qty : 0
  const catDomain = p.rootDomain || p.cat
  const catLabel = rootMeta(p.rootDomain).label || p.cat

  const buy = (toCartFlag: boolean) => {
    if (!p.available) return
    if (fulfill === 'transfer' && !rName) {
      toast(t('recipient_name'), 'user')
      return
    }
    if (fulfill !== 'transfer' && p.needsId && !pid) {
      toast(t('id_ph'), 'user')
      return
    }
    addToCart({
      id: p.id,
      variantId: v.id,
      brand: p.brand,
      title: p.title,
      art: p.art,
      image: p.image,
      variant: v.l,
      price: unit,
      qty,
      pid,
      fulfill,
      recipient: fulfill === 'transfer' ? { name: rName, country, detail: '' } : null,
    })
    if (toCartFlag) toast(t('add_cart'), 'cart')
    else navigate('/checkout')
  }

  return (
    <div className="wrap" style={{ paddingBottom: 40 }}>
      <div className="row" style={{ gap: 10, margin: '16px 0' }}>
        <a className="small clickable faint" onClick={() => navigate('/')}>
          {t('nav_home')}
        </a>
        <span className="faint">/</span>
        <a className="small clickable faint" onClick={() => navigate('/category/' + catDomain)}>
          {catLabel}
        </a>
        <span className="faint">/</span>
        <span className="small" style={{ fontWeight: 700 }}>
          {p.brand}
        </span>
      </div>

      <div className="pdp">
        {/* media */}
        <div className="pdp-media">
          <ImageArt art={p.art} src={p.imageFull} word={p.brand} sub={p.title} h={380} wordSize={48} radius={22} />
          <div className="row" style={{ gap: 12, marginTop: 14, flexWrap: 'wrap' }}>
            {!p.available && <span className="badge badge-disc">{t('out_of_stock')}</span>}
            {p.instant && (
              <span className="badge badge-instant">
                <Icon name="bolt" size={12} />
                {t('instant')}
              </span>
            )}
            {fulfill === 'code' && (
              <span className="badge badge-secure">
                <Icon name="shield" size={12} />
                {t('secure_vault')}
              </span>
            )}
            {fulfill === 'credit' && (
              <span className="badge badge-secure">
                <Icon name="user" size={12} />
                {t('credited_to')} ID
              </span>
            )}
            {fulfill === 'transfer' && (
              <span className="badge badge-soft">
                <Icon name="repeat" size={12} />
                {t('st_processing')}
              </span>
            )}
            <span className="badge badge-agent">
              <Icon name="check" size={12} />
              {t('official')}
            </span>
            <span className="badge badge-soft">
              <Icon name="check" size={12} />
              {t('verified')}
            </span>
          </div>
        </div>

        {/* info */}
        <div className="col" style={{ gap: 22 }}>
          <div>
            <div className="col" style={{ gap: 2 }}>
              <h1 className="h1">{p.brand}</h1>
              <span className="muted" style={{ fontSize: 18, fontWeight: 700 }}>
                {p.title}
              </span>
            </div>
            <div className="row wrap-gap" style={{ gap: 12, marginTop: 10 }}>
              {p.reviews > 0 && (
                <>
                  <Stars value={p.rating} />
                  <span className="small num" style={{ fontWeight: 700 }}>
                    {p.rating.toFixed(1)}
                  </span>
                  <span className="small faint num">
                    {p.reviews.toLocaleString()} {t('reviews')}
                  </span>
                </>
              )}
              {p.offer && (
                <span className="badge badge-disc">
                  {p.offer.label}
                  {p.offer.endsAt &&
                    ` · ${t('offer_ends')} ${new Date(p.offer.endsAt).toLocaleDateString()}`}
                </span>
              )}
              {p.sold && (
                <span className="small faint">
                  · {p.sold} {t('sold')}
                </span>
              )}
            </div>
          </div>

          {/* variants */}
          <div>
            <div className="label">{t('choose_amount')}</div>
            <div className="variants">
              {p.variants.map((vv, i) => (
                <div
                  key={i}
                  className={'variant' + (i === vIndex ? ' on' : '')}
                  onClick={() => setVi(i)}
                >
                  <div style={{ fontWeight: 800, fontSize: 15 }}>{vv.l}</div>
                  <div className="row center" style={{ gap: 6, marginTop: 4 }}>
                    <Price usd={priceFor(p, vv, agent)} cur={cur} className="small num" />
                    {agent && <Price usd={vv.p} cur={cur} strike className="tiny" />}
                    {!agent && vv.offerP !== undefined && (
                      <Price usd={vv.p} cur={cur} strike className="tiny" />
                    )}
                  </div>
                </div>
              ))}
            </div>
          </div>

          {/* fulfillment input */}
          {fulfill === 'transfer' ? (
            <div className="col" style={{ gap: 12 }}>
              <div>
                <span className="label">{t('recipient_name')}</span>
                <input
                  className="field"
                  placeholder="Ayşe Yılmaz"
                  value={rName}
                  onChange={(e) => setRName(e.target.value)}
                />
              </div>
              <div>
                <span className="label">{t('country')}</span>
                <select className="field" value={country} onChange={(e) => setCountry(e.target.value)}>
                  <option>Türkiye</option>
                  <option>United States</option>
                  <option>United Arab Emirates</option>
                  <option>Saudi Arabia</option>
                  <option>Egypt</option>
                </select>
              </div>
            </div>
          ) : (
            p.needsId && (
              <div>
                <div className="row between">
                  <span className="label" style={{ whiteSpace: 'nowrap', margin: 0 }}>
                    {p.idLabel || t('player_id')}
                  </span>
                  {isAuthenticated && savedIds.length > 0 && (
                    <a
                      className="tiny clickable"
                      style={{ color: 'var(--brand-1)', fontWeight: 700 }}
                      onClick={() => setPickId((s) => !s)}
                    >
                      {t('saved_ids')} ▾
                    </a>
                  )}
                </div>
                <input
                  className="field"
                  placeholder={t('id_ph')}
                  value={pid}
                  onChange={(e) => setPid(e.target.value)}
                />
                {pickId && (
                  <div className="panel card-pad" style={{ marginTop: 8, padding: 10 }}>
                    {savedIds.map((sv) => (
                      <div
                        key={sv.value}
                        className="lrow clickable"
                        style={{ padding: '10px 6px' }}
                        onClick={() => {
                          setPid(sv.value)
                          setPickId(false)
                        }}
                      >
                        <div className="col" style={{ gap: 1 }}>
                          <span style={{ fontWeight: 700, fontSize: 14 }}>{sv.label}</span>
                          <span className="num tiny faint">{sv.value}</span>
                        </div>
                        <span className="spacer" />
                        <Icon name="chevron" size={16} />
                      </div>
                    ))}
                  </div>
                )}
              </div>
            )
          )}

          {/* qty + price + actions */}
          <div className="panel card-pad" style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
            <div className="row between">
              <span className="label" style={{ margin: 0 }}>
                {t('quantity')}
              </span>
              <Stepper value={qty} set={setQty} />
            </div>
            <hr className="divider" />
            <div className="row between">
              <span className="muted">{t('total')}</span>
              <div className="col" style={{ alignItems: 'flex-end', gap: 2 }}>
                <Price usd={total} cur={cur} className="display-l" />
                {agent && savings > 0 && (
                  <span className="badge badge-agent">
                    <Icon name="shield" size={11} />
                    {t('agent_price')} · {t('you_save')} {fmtPrice(savings, cur)}
                  </span>
                )}
              </div>
            </div>
            <div className="row" style={{ gap: 12 }}>
              <button
                className="btn btn-ghost btn-lg"
                style={{ flex: 1 }}
                disabled={!p.available}
                onClick={() => buy(true)}
              >
                <Icon name="cart" size={18} />
                {t('add_cart')}
              </button>
              <button
                className="btn btn-primary btn-lg"
                style={{ flex: 1.4 }}
                disabled={!p.available}
                onClick={() => buy(false)}
              >
                <Icon name="bolt" size={18} />
                {p.available ? t('buy_now') : t('out_of_stock')}
              </button>
            </div>
            <div className="row center" style={{ gap: 8, color: 'var(--ok)' }}>
              <Icon name={fulfill === 'transfer' ? 'repeat' : 'bolt'} size={15} />
              <span className="small" style={{ fontWeight: 700 }}>
                {fulfill === 'code' && `${t('trust_instant')} · ${t('secure_vault')}`}
                {fulfill === 'credit' &&
                  `${t('credited_to')} ${t('player_id').toLowerCase()} · ${t('trust_instant')}`}
                {fulfill === 'transfer' && t('transfer_note')}
              </span>
            </div>
          </div>
        </div>
      </div>

      {/* how it works */}
      <section className="section">
        <h2 className="h2" style={{ marginBottom: 16 }}>
          {t('how_works')}
        </h2>
        <div className="howsteps">
          {[t('step1'), t('step2'), t('step3')].map((s, i) => (
            <div className="step" key={i}>
              <div className="num-badge">{i + 1}</div>
              <p style={{ fontWeight: 600 }}>{s}</p>
            </div>
          ))}
        </div>
      </section>

      {/* reviews */}
      <section className="section">
        <div className="row between" style={{ marginBottom: 16 }}>
          <h2 className="h2">{t('review_title')}</h2>
          <span className="badge badge-instant">
            <Icon name="check" size={12} /> {p.reviews.toLocaleString()} {t('reviews')}
          </span>
        </div>
        {reviews.length === 0 ? (
          <p className="muted" style={{ padding: '8px 4px' }}>
            {t('no_reviews')}
          </p>
        ) : (
          <div className="prodgrid" style={{ gridTemplateColumns: 'repeat(3,1fr)' }}>
            {reviews.map((r) => (
              <div key={r.id} className="panel card-pad" style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
                <div className="row between">
                  <div className="row" style={{ gap: 10 }}>
                    <span className="avatar" style={{ width: 34, height: 34, fontSize: 13 }}>
                      {r.initials}
                    </span>
                    <span style={{ fontWeight: 700, fontSize: 14 }}>{r.name}</span>
                  </div>
                  <Stars value={r.stars} size={12} />
                </div>
                <p className="small muted">{r.body}</p>
                <span
                  className={'badge ' + (r.verified ? 'badge-instant' : 'badge-soft')}
                  style={{ alignSelf: 'flex-start' }}
                >
                  {r.verified && <Icon name="check" size={11} />}
                  {r.verified ? `${t('verified')} · ${r.date}` : r.date}
                </span>
              </div>
            ))}
          </div>
        )}
      </section>

      {/* related */}
      {related.length > 0 && (
        <section className="section">
          <h2 className="h2" style={{ marginBottom: 16 }}>
            {t('see_more')}
          </h2>
          <div className="scroller">
            {related.map((x) => (
              <ProductCard key={x.id} p={x} compact />
            ))}
          </div>
        </section>
      )}
    </div>
  )
}
