import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate, useParams } from 'react-router-dom'
import { Icon, PageHead, Toggle, StatusBadge, LoadingSpinner, ErrorState, ffKey, type FfKey } from '@/components'
import { CATS } from '@/lib/mock/demo'
import { useProduct, useCreateProduct, useUpdateProduct } from '../hooks/useProducts'
import type { FulfillmentType, Locale } from '@/types'

interface VariantRow {
  denomination: string
  price: string
  resellerPrice: string
}

const FF_OPTIONS: [FfKey, string, string, string, string][] = [
  ['code', 'ff-code', 'var(--ff-code)', 'Code / PIN', 'Gift cards, keys, vouchers — delivered instantly from inventory.'],
  ['credit', 'ff-credit', 'var(--ff-credit)', 'Account credit', 'Game / app top-ups credited to a player ID — no code.'],
  ['transfer', 'ff-transfer', 'var(--ff-transfer)', 'Money transfer', 'Service flow with a status timeline & manual steps.'],
]

function toFulfillment(ff: FfKey): FulfillmentType {
  return ff === 'credit' ? 'account_credit' : ff === 'transfer' ? 'transfer' : 'code'
}

export default function ProductEditPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  const isNew = !id
  const { data, isLoading, isError, refetch } = useProduct(id)
  const create = useCreateProduct()
  const update = useUpdateProduct(id ?? '')

  const [langTab, setLangTab] = useState<Locale>('en')
  const [title, setTitle] = useState({ en: '', ar: '', tr: '' })
  const [category, setCategory] = useState(CATS[0])
  const [ff, setFf] = useState<FfKey>('code')
  const [active, setActive] = useState(false)
  const [stock, setStock] = useState(0)
  const [variants, setVariants] = useState<VariantRow[]>([
    { denomination: '$10', price: '10.00', resellerPrice: '8.80' },
  ])

  // Populate from the loaded product when editing.
  useEffect(() => {
    const p = data?.data
    if (!p) return
    setTitle({ en: p.title.en, ar: p.title.ar, tr: p.title.tr })
    setCategory(p.category)
    setFf(ffKey(p.fulfillmentType))
    setActive(p.available)
    setStock(p.stock)
    setVariants(
      p.variants.length
        ? p.variants.map((v) => ({
            denomination: v.denomination,
            price: String(v.price),
            resellerPrice: v.resellerPrice != null ? String(v.resellerPrice) : '',
          }))
        : [{ denomination: '', price: '', resellerPrice: '' }]
    )
  }, [data])

  const pending = create.isPending || update.isPending
  const error = create.error || update.error

  const buildInput = () => ({
    title,
    category,
    images: data?.data?.images ?? [],
    fulfillmentType: toFulfillment(ff),
    available: active,
    stock,
    variants: variants
      .filter((v) => v.denomination.trim() && v.price.trim())
      .map((v) => ({
        denomination: v.denomination,
        price: parseFloat(v.price) || 0,
        ...(v.resellerPrice.trim() ? { resellerPrice: parseFloat(v.resellerPrice) || 0 } : {}),
      })),
  })

  const onSave = () => {
    const input = buildInput()
    const onSuccess = () => navigate('/products')
    if (isNew) create.mutate(input, { onSuccess })
    else update.mutate(input, { onSuccess })
  }

  const margin = (v: VariantRow) => {
    const c = parseFloat(v.price)
    const r = parseFloat(v.resellerPrice)
    if (!c || !r) return 0
    return Math.round((1 - r / c) * 100)
  }

  const headTitle = useMemo(() => (isNew ? t('new_product') : data?.data?.title.en || 'Edit product'), [isNew, data, t])

  if (!isNew && isLoading) return <div className="page"><LoadingSpinner /></div>
  if (!isNew && isError)
    return (
      <div className="page">
        <ErrorState message="Could not load product" onRetry={() => refetch()} />
      </div>
    )

  return (
    <div className="page">
      <PageHead
        crumbs={[t('nav_products'), isNew ? t('new_product') : headTitle]}
        title={isNew ? t('new_product') : 'Edit product'}
        sub={isNew ? 'Create a new product listing' : id}
      >
        <button className="abtn" onClick={() => navigate('/products')}>
          <Icon name="chevleft" size={15} /> {t('back')}
        </button>
        <button className="abtn primary" onClick={onSave} disabled={pending}>
          <Icon name="check" size={15} /> {pending ? 'Saving…' : t('save')}
        </button>
      </PageHead>

      {error && (
        <div
          style={{
            marginBottom: 16,
            padding: '10px 14px',
            borderRadius: 'var(--ar-sm)',
            background: 'rgba(255,77,109,.12)',
            color: 'var(--danger)',
            fontSize: 13,
            fontWeight: 700,
          }}
        >
          {(error as Error).message}
        </div>
      )}

      <div className="formgrid">
        <div className="fieldset">
          {/* Basics */}
          <div className="acard pad">
            <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 16 }}>Basics</h3>
            <div className="langtab">
              {(['en', 'ar', 'tr'] as Locale[]).map((l) => (
                <button key={l} className={langTab === l ? 'on' : ''} onClick={() => setLangTab(l)}>
                  {l.toUpperCase()}
                </button>
              ))}
            </div>
            <label className="alabel">Title ({langTab.toUpperCase()})</label>
            <input
              className="afield"
              value={title[langTab]}
              onChange={(e) => setTitle({ ...title, [langTab]: e.target.value })}
              placeholder={langTab === 'ar' ? 'اسم المنتج بالعربية' : langTab === 'tr' ? 'Ürün adı' : 'Product title'}
              dir={langTab === 'ar' ? 'rtl' : 'ltr'}
            />
            <div className="ahint">Localized titles shown to customers in each language.</div>
            <div style={{ marginTop: 16 }}>
              <label className="alabel">Description ({langTab.toUpperCase()})</label>
              <textarea className="afield" placeholder="Describe the product, redemption steps, region…" />
              <div className="ahint">Description is design-only for now (not yet persisted by the API).</div>
            </div>
            <div className="g2" style={{ marginTop: 16 }}>
              <div>
                <label className="alabel">Category</label>
                <select className="select" style={{ width: '100%' }} value={category} onChange={(e) => setCategory(e.target.value)}>
                  {CATS.map((c) => (
                    <option key={c}>{c}</option>
                  ))}
                </select>
              </div>
              <div>
                <label className="alabel">Region</label>
                <select className="select" style={{ width: '100%' }}>
                  <option>United States</option>
                  <option>Türkiye</option>
                  <option>Global</option>
                  <option>MENA</option>
                </select>
              </div>
            </div>
          </div>

          {/* Fulfillment selector */}
          <div className="acard pad">
            <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 4 }}>Fulfillment type</h3>
            <div className="ahint" style={{ marginTop: 0, marginBottom: 14 }}>
              Determines how the product is delivered — this changes the fields below.
            </div>
            <div className="g3">
              {FF_OPTIONS.map(([k, , color, label, desc]) => (
                <div
                  key={k}
                  onClick={() => setFf(k)}
                  style={{
                    cursor: 'pointer',
                    padding: 15,
                    borderRadius: 'var(--ar-md)',
                    border: '1.5px solid ' + (ff === k ? color : 'var(--border)'),
                    background: ff === k ? 'var(--grad-soft)' : 'var(--surface-2)',
                    transition: '.15s',
                  }}
                >
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 8 }}>
                    <span style={{ width: 10, height: 10, borderRadius: 99, background: color }} />
                    <b style={{ fontSize: 13.5 }}>{label}</b>
                    {ff === k && <Icon name="checkc" size={16} />}
                  </div>
                  <div style={{ fontSize: 12, color: 'var(--text-dim)', lineHeight: 1.4 }}>{desc}</div>
                </div>
              ))}
            </div>
          </div>

          {/* Variants builder */}
          <div className="acard">
            <div className="panelhead">
              <Icon name="layers" size={17} />
              <h3>Variants &amp; pricing</h3>
              <div className="ph-act">
                <button
                  className="abtn xs"
                  onClick={() => setVariants([...variants, { denomination: '', price: '', resellerPrice: '' }])}
                >
                  <Icon name="plus" size={13} /> Add variant
                </button>
              </div>
            </div>
            <div className="tablewrap">
              <table className="tbl">
                <thead>
                  <tr>
                    <th>{ff === 'credit' ? 'Denomination' : 'Face value'}</th>
                    <th>Customer price</th>
                    <th>Reseller price</th>
                    <th>Margin</th>
                    <th></th>
                  </tr>
                </thead>
                <tbody>
                  {variants.map((v, i) => (
                    <tr key={i}>
                      <td>
                        <input
                          className="afield"
                          style={{ padding: '7px 10px', width: 90 }}
                          value={v.denomination}
                          onChange={(e) => {
                            const next = [...variants]
                            next[i] = { ...v, denomination: e.target.value }
                            setVariants(next)
                          }}
                        />
                      </td>
                      <td>
                        <input
                          className="afield"
                          style={{ padding: '7px 10px', width: 110 }}
                          value={v.price}
                          onChange={(e) => {
                            const next = [...variants]
                            next[i] = { ...v, price: e.target.value }
                            setVariants(next)
                          }}
                        />
                      </td>
                      <td>
                        <input
                          className="afield"
                          style={{ padding: '7px 10px', width: 110 }}
                          value={v.resellerPrice}
                          onChange={(e) => {
                            const next = [...variants]
                            next[i] = { ...v, resellerPrice: e.target.value }
                            setVariants(next)
                          }}
                        />
                      </td>
                      <td>
                        <span className="st st-ok" style={{ fontSize: 11 }}>
                          {margin(v)}%
                        </span>
                      </td>
                      <td>
                        <span
                          className="iact danger"
                          onClick={() => setVariants(variants.filter((_, j) => j !== i))}
                        >
                          <Icon name="trash" size={15} />
                        </span>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>

          {/* Fulfillment-specific block */}
          {ff === 'code' && (
            <div className="acard pad" style={{ borderColor: 'var(--ff-code-bd)' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 4 }}>
                <span className="bdg ff-code">
                  <i className="d" />
                  Code inventory
                </span>
                <h3 style={{ fontSize: 15, fontWeight: 800 }}>{stock} codes available</h3>
                <div style={{ marginInlineStart: 'auto' }}>
                  <button
                    className="abtn sm primary"
                    disabled={isNew}
                    title={isNew ? 'Save the product first' : undefined}
                    onClick={() => id && navigate(`/inventory?upload=${id}`)}
                  >
                    <Icon name="upload" size={14} /> Upload codes
                  </button>
                </div>
              </div>
              <div className="ahint" style={{ marginTop: 0 }}>
                Manage the code pool that fulfills this product in Inventory. Low-stock alert fires below the threshold.
              </div>
            </div>
          )}
          {ff === 'credit' && (
            <div className="acard pad" style={{ borderColor: 'var(--ff-credit-bd)' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 10 }}>
                <span className="bdg ff-credit">
                  <i className="d" />
                  Account credit
                </span>
                <h3 style={{ fontSize: 15, fontWeight: 800 }}>Player ID field</h3>
              </div>
              <label className="alabel">ID label shown at checkout</label>
              <input className="afield" defaultValue="Player ID" />
              <div className="ahint">Customers enter this ID; top-ups are credited automatically via the provider API.</div>
            </div>
          )}
          {ff === 'transfer' && (
            <div className="acard pad" style={{ borderColor: 'var(--ff-transfer-bd)' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 10 }}>
                <span className="bdg ff-transfer">
                  <i className="d" />
                  Money transfer
                </span>
                <h3 style={{ fontSize: 15, fontWeight: 800 }}>Service settings</h3>
              </div>
              <div className="ahint" style={{ marginTop: 0 }}>
                Transfer orders use a manual status timeline (Submitted → Processing → Completed). No inventory or code
                pool is attached.
              </div>
            </div>
          )}
        </div>

        {/* SIDE */}
        <div className="fieldset">
          <div className="acard pad">
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 14 }}>
              <h3 style={{ fontSize: 15, fontWeight: 800 }}>Status</h3>
              <Toggle on={active} onClick={() => setActive(!active)} />
            </div>
            <StatusBadge s={active ? 'active' : 'draft'} />
            <div className="ahint">
              {active ? 'Visible in the storefront and purchasable.' : 'Hidden from customers until published.'}
            </div>
          </div>

          {ff === 'code' && (
            <div className="acard pad">
              <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 12 }}>Stock</h3>
              <label className="alabel">Codes available (stock)</label>
              <input
                className="afield"
                type="number"
                value={stock}
                onChange={(e) => setStock(parseInt(e.target.value, 10) || 0)}
              />
              <div className="ahint">Bulk-upload codes in Inventory to grow the pool.</div>
            </div>
          )}

          <div className="acard pad">
            <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 12 }}>Images</h3>
            <div className="imgslot" style={{ height: 150, marginBottom: 10 }}>
              box art · 1:1 · drop image
            </div>
            <div className="g3">
              {[1, 2, 3].map((i) => (
                <div key={i} className="imgslot" style={{ height: 56 }}>
                  {i}
                </div>
              ))}
            </div>
            <div className="ahint">Image upload is design-only for now (gradient box-art is generated).</div>
          </div>

          <div className="acard pad">
            <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 12 }}>Organize</h3>
            <label className="alabel">Tags</label>
            <input className="afield" defaultValue="instant, popular" />
            <div style={{ marginTop: 14 }}>
              <label className="alabel">Badges</label>
              <div className="chiprow">
                <div className="chip on">Instant</div>
                <div className="chip">Best seller</div>
                <div className="chip">New</div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
