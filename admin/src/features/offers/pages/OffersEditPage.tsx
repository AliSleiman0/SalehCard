import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate, useParams } from 'react-router-dom'
import { Icon, PageHead, Chip, LoadingSpinner, ErrorState } from '@/components'
import { ApiError } from '@/lib/api-client'
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
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  const isNew = !id

  const { data, isLoading, isError, refetch } = useOffer(id)
  const products = useProducts({ limit: 100, available: true })
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
        <ErrorState message="Couldn't load this offer." onRetry={() => refetch()} />
      </div>
    )
  }

  const saving = create.isPending || update.isPending

  const save = () => {
    const v = Number(value)
    if (!productId) {
      setError('Pick a product.')
      return
    }
    if (!Number.isFinite(v) || v <= 0) {
      setError('Discount value must be greater than zero.')
      return
    }
    if (discountType === 'percent' && v > 100) {
      setError('Percent value must be between 0 and 100.')
      return
    }
    if (startsAt && endsAt && endsAt < startsAt) {
      setError('End date must be after the start date.')
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
    const onError = (e: unknown) => setError(e instanceof ApiError ? e.message : 'Save failed.')
    if (isNew) {
      create.mutate(input, { onSuccess: () => navigate('/offers'), onError })
    } else {
      update.mutate(input, { onSuccess: () => navigate('/offers'), onError })
    }
  }

  const productName = selected?.title.en || data?.data?.product?.title.en || ''

  return (
    <div className="page">
      <PageHead
        crumbs={[t('nav_offers'), isNew ? 'New offer' : productName || 'Edit']}
        title={isNew ? 'New offer' : 'Edit offer'}
        sub={isNew ? 'Put a product on a time-boxed sale price' : productName}
      >
        <button className="abtn" onClick={() => navigate('/offers')}>
          <Icon name="chevleft" size={15} /> {t('back')}
        </button>
        <button className="abtn primary" onClick={save} disabled={saving}>
          <Icon name="check" size={15} /> {t('save')}
        </button>
      </PageHead>

      {error && (
        <div style={{ color: 'var(--danger)', fontSize: 13, fontWeight: 600, marginBottom: 14 }}>{error}</div>
      )}

      <div className="formgrid">
        <div className="fieldset">
          <div className="acard pad">
            <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 14 }}>Product &amp; discount</h3>
            <label className="alabel">Product</label>
            <select
              className="select"
              style={{ width: '100%' }}
              value={productId}
              disabled={products.isLoading}
              onChange={(e) => setProductId(e.target.value)}
            >
              <option value="">{products.isLoading ? 'Loading products…' : 'Select a product…'}</option>
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

            <label className="alabel" style={{ marginTop: 16 }}>
              Discount type
            </label>
            <div className="g2">
              {(
                [
                  ['percent', 'Percent %'],
                  ['fixed', 'Fixed $ off'],
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
                <label className="alabel">Value {discountType === 'percent' ? '(%)' : '($)'}</label>
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
                <label className="alabel">Sort order</label>
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
              Lower sort order shows first and wins when a product has more than one live offer.
            </div>
          </div>
        </div>

        <div className="fieldset">
          <div className="acard pad">
            <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 14 }}>Validity</h3>
            <div className="g2">
              <div>
                <label className="alabel">Status</label>
                <select
                  className="select"
                  style={{ width: '100%' }}
                  value={active ? 'active' : 'paused'}
                  onChange={(e) => setActive(e.target.value === 'active')}
                >
                  <option value="active">Active</option>
                  <option value="paused">Paused</option>
                </select>
              </div>
              <div />
            </div>
            <label className="alabel" style={{ marginTop: 12 }}>
              Start date
            </label>
            <input className="afield" type="date" value={startsAt} onChange={(e) => setStartsAt(e.target.value)} />
            <label className="alabel" style={{ marginTop: 12 }}>
              End date
            </label>
            <input className="afield" type="date" value={endsAt} onChange={(e) => setEndsAt(e.target.value)} />
            <div className="ahint" style={{ marginTop: 8 }}>
              Leave a date empty for no bound.
            </div>
          </div>

          <div className="acard pad">
            <div className="alabel">Customer preview</div>
            <div
              style={{
                padding: 20,
                background: 'var(--grad-soft)',
                borderRadius: 'var(--ar-md)',
                border: '1.5px dashed var(--ff-code-bd)',
                textAlign: 'center',
              }}
            >
              <div style={{ fontSize: 14, fontWeight: 700, marginBottom: 6 }}>{productName || 'Select a product'}</div>
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
                  Pick a product to preview the sale price.
                </div>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
