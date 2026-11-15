import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useParams } from 'react-router-dom'
import { Icon, PageHead, Chip, LoadingSpinner, ErrorState } from '@/components'
import { useGoBack } from '@/lib/goBack'
import { apiErrorText } from '@/lib/apiErrorText'
import { useCan } from '@/stores/auth'
import { useProducts } from '@/features/products/hooks/useProducts'
import { useOffer, useCreateOffer, useUpdateOffer } from '../hooks/useOffers'
import type { OfferInput, DiscountType } from '../api/offers'

// dateInput converts an ISO timestamp to a yyyy-mm-dd value for <input type=date>.
function dateInput(iso?: string): string {
  return iso ? iso.slice(0, 10) : ''
}

// fromPrice is a product's lowest variant price (0 with no variants).
function fromPrice(variants: { price: number }[]): number {
  if (!variants.length) return 0
  return variants.reduce((min, v) => (v.price < min ? v.price : min), variants[0].price)
}

// applyDiscount mirrors the server's OfferPriceFor (clamped to [0, original]).
function applyDiscount(type: DiscountType, value: number, original: number): number {
  if (original <= 0) return original
  const d = type === 'percent' ? (original * value) / 100 : value
  return Math.max(0, original - Math.min(Math.max(d, 0), original))
}

export default function OffersEditPage() {
  const { t } = useTranslation()
  const goBack = useGoBack('/offers')
  const can = useCan()
  const canManage = can('offers.manage')
  const { id } = useParams<{ id: string }>()
  const isNew = !id

  const { data, isLoading, isError, refetch } = useOffer(id)
  // The product picker reads the products domain; an offers-only role would 403.
  const canViewProducts = can('products.view')
  const products = useProducts({ limit: 100, available: true }, { enabled: canViewProducts })
  const create = useCreateOffer()
  const update = useUpdateOffer(id ?? '')

  const [productId, setProductId] = useState('')
  const [discountType, setDiscountType] = useState<DiscountType>('percent')
  const [value, setValue] = useState('25')
  const [startsAt, setStartsAt] = useState('')
  const [endsAt, setEndsAt] = useState('')
  const [sortOrder, setSortOrder] = useState('0')
  const [active, setActive] = useState(true)
  const [error, setError] = useState('')

  // Populate the form once the existing offer loads (edit mode).
  useEffect(() => {
    const o = data?.data
    if (!o) return
    setProductId(o.productId)
    setDiscountType(o.discountType)
    setValue(String(o.discountValue))
    setStartsAt(dateInput(o.startsAt))
    setEndsAt(dateInput(o.endsAt))
    setSortOrder(String(o.sortOrder))
    setActive(o.active)
  }, [data])

  const productList = useMemo(() => products.data?.data ?? [], [products.data])
  const selected = useMemo(
    () => productList.find((p) => p.id === productId),
    [productList, productId],
  )
  // While editing, the joined product on the offer covers a product not in the
  // first page of the picker list (so the preview still works).
  const original = selected ? fromPrice(selected.variants) : (data?.data?.originalFromPrice ?? 0)
  const preview = applyDiscount(discountType, Number(value) || 0, original)

  if (!isNew && isLoading) return <LoadingSpinner />
  if (!isNew && (isError || !data?.data)) {
    return (
      <div className="page">
        <PageHead back="/offers" title={t('nav_offers')} />
        <ErrorState message={t('off_load_one_error')} onRetry={() => refetch()} />
      </div>
    )
  }

  const saving = create.isPending || update.isPending

  const save = () => {
    const v = Number(value)
    if (!productId) {
      setError(t('off_err_product'))
      return
    }
    if (!Number.isFinite(v) || v <= 0) {
      setError(t('off_err_value'))
      return
    }
    if (discountType === 'percent' && v > 100) {
      setError(t('promo_err_percent'))
      return
    }
    if (startsAt && endsAt && endsAt < startsAt) {
      setError(t('promo_err_dates'))
      return
    }
    setError('')
    const input: OfferInput = {
      productId,
      discountType,
      discountValue: v,
      startsAt: startsAt ? new Date(startsAt).toISOString() : null,
      endsAt: endsAt ? new Date(endsAt).toISOString() : null,
      sortOrder: Number(sortOrder) || 0,
      active,
    }
    const onError = (e: unknown) => setError(apiErrorText(e, 'exp_save_error'))
    if (isNew) {
      create.mutate(input, { onSuccess: () => goBack(), onError })
    } else {
      update.mutate(input, { onSuccess: () => goBack(), onError })
    }
  }

  const productName = selected?.title.en || data?.data?.product?.title.en || ''

  return (
    <div className="page">
      <PageHead
        back="/offers"
        crumbs={[t('nav_offers'), isNew ? t('off_new') : productName || t('edit')]}
        title={isNew ? t('off_new') : t('off_edit')}
        sub={isNew ? t('off_new_sub') : productName}
      >
        {canManage && (
          <button className="abtn primary" onClick={save} disabled={saving}>
            <Icon name="check" size={15} /> {t('save')}
          </button>
        )}
      </PageHead>

      {error && (
        <div style={{ color: 'var(--danger)', fontSize: 13, fontWeight: 600, marginBottom: 14 }}>{error}</div>
      )}

      <div className="formgrid">
        <div className="fieldset">
          <div className="acard pad">
            <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 14 }}>{t('off_section')}</h3>
            <label className="alabel">{t('col_product')}</label>
            {!canViewProducts ? (
              <div className="ahint" style={{ color: 'var(--danger)' }}>
                {t('off_need_products')}
              </div>
            ) : (
              <select
                className="select"
                style={{ width: '100%' }}
                value={productId}
                disabled={products.isLoading}
                onChange={(e) => setProductId(e.target.value)}
              >
                <option value="">{products.isLoading ? t('off_loading_products') : t('off_select_product')}</option>
                {productList.map((p) => (
                  <option key={p.id} value={p.id}>
                    {p.title.en} — ${fromPrice(p.variants).toFixed(2)}
                  </option>
                ))}
                {/* Keep the edited offer's product selectable even if it isn't on the first page. */}
                {!isNew && productId && !selected && data?.data?.product && (
                  <option value={productId}>{data.data.product.title.en}</option>
                )}
              </select>
            )}

            <label className="alabel" style={{ marginTop: 16 }}>
              {t('promo_discount_type')}
            </label>
            <div className="g2">
              {(
                [
                  ['percent', t('promo_chip_percent')],
                  ['fixed', t('off_chip_fixed')],
                ] as [DiscountType, string][]
              ).map(([k, l]) => (
                <Chip
                  key={k}
                  on={discountType === k}
                  onClick={() => setDiscountType(k)}
                  style={{ justifyContent: 'center', padding: 12 }}
                >
                  {l}
                </Chip>
              ))}
            </div>
            <div className="g2" style={{ marginTop: 16 }}>
              <div>
                <label className="alabel">
                  {t('promo_col_value')} {discountType === 'percent' ? '(%)' : '($)'}
                </label>
                <input
                  className="afield"
                  type="number"
                  min="0"
                  step="0.01"
                  value={value}
                  onChange={(e) => setValue(e.target.value)}
                />
              </div>
              <div>
                <label className="alabel">{t('cat_sort_order')}</label>
                <input
                  className="afield"
                  type="number"
                  min="0"
                  step="1"
                  value={sortOrder}
                  onChange={(e) => setSortOrder(e.target.value)}
                />
              </div>
            </div>
            <div className="ahint" style={{ marginTop: 8 }}>
              {t('off_sort_hint')}
            </div>
          </div>
        </div>

        <div className="fieldset">
          <div className="acard pad">
            <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 14 }}>{t('promo_validity')}</h3>
            <div className="g2">
              <div>
                <label className="alabel">{t('status')}</label>
                <select
                  className="select"
                  style={{ width: '100%' }}
                  value={active ? 'active' : 'paused'}
                  onChange={(e) => setActive(e.target.value === 'active')}
                >
                  <option value="active">{t('promo_status_active')}</option>
                  <option value="paused">{t('promo_status_paused')}</option>
                </select>
              </div>
              <div />
            </div>
            <label className="alabel" style={{ marginTop: 12 }}>
              {t('promo_start')}
            </label>
            <input className="afield" type="date" value={startsAt} onChange={(e) => setStartsAt(e.target.value)} />
            <label className="alabel" style={{ marginTop: 12 }}>
              {t('promo_end')}
            </label>
            <input className="afield" type="date" value={endsAt} onChange={(e) => setEndsAt(e.target.value)} />
            <div className="ahint" style={{ marginTop: 8 }}>
              {t('promo_no_bound')}
            </div>
          </div>

          <div className="acard pad">
            <div className="alabel">{t('promo_preview')}</div>
            <div
              style={{
                padding: 20,
                background: 'var(--grad-soft)',
                borderRadius: 'var(--ar-md)',
                border: '1.5px dashed var(--ff-code-bd)',
                textAlign: 'center',
              }}
            >
              <div style={{ fontSize: 14, fontWeight: 700, marginBottom: 6 }}>{productName || t('off_select_product_plain')}</div>
              {original > 0 ? (
                <div>
                  <span style={{ textDecoration: 'line-through', color: 'var(--text-dim)', marginInlineEnd: 8 }}>
                    ${original.toFixed(2)}
                  </span>
                  <span className="mono" style={{ fontSize: 22, fontWeight: 800 }}>
                    ${preview.toFixed(2)}
                  </span>
                </div>
              ) : (
                <div className="muted" style={{ fontSize: 13 }}>
                  {t('off_pick_preview')}
                </div>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
