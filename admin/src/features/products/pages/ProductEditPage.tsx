import { useEffect, useMemo, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate, useParams } from 'react-router-dom'
import { Icon, PageHead, Toggle, StatusBadge, LoadingSpinner, ErrorState, ffKey, type FfKey } from '@/components'
import { useProductCategories } from '../hooks/useCategories'
import { categoryLabel } from '../api/categories'
import { useProduct, useCreateProduct, useUpdateProduct, useUploadProductImage } from '../hooks/useProducts'
import { reconcileBridgePhoneField } from '../lib/bridgeFields'
import { useCan } from '@/stores/auth'
import type { FulfillmentType, Locale, InputField } from '@/types'

const MAX_IMAGE_BYTES = 10 << 20 // 10 MB — server is authoritative; this is UX only.
const IMAGE_ACCEPT = 'image/jpeg,image/png,image/webp'

interface VariantRow {
  // Carried through so an edit preserves the variant's identity: the backend
  // mints a fresh _id for any variant submitted without one, which would orphan
  // the customer's cart line and reseller per-variant price overrides.
  id?: string
  denomination: string
  price: string
  resellerPrice: string
  faceValue: string
}

const FF_OPTIONS: [FfKey, string, string, string, string][] = [
  ['code', 'ff-code', 'var(--ff-code)', 'Code / PIN', 'Gift cards, keys, vouchers — delivered instantly from inventory.'],
  ['credit', 'ff-credit', 'var(--ff-credit)', 'Account credit', 'Game / app top-ups credited to a player ID — no code.'],
  ['transfer', 'ff-transfer', 'var(--ff-transfer)', 'Money transfer', 'Service flow with a status timeline & manual steps.'],
]

function toFulfillment(ff: FfKey): FulfillmentType {
  return ff === 'credit' ? 'account_credit' : ff === 'transfer' ? 'transfer' : 'code'
}

// The verification-provider id stored on a product. One provider today (RapidAPI
// ID Game Checker); the active adapter is chosen globally by the API's
// IDCHECK_PROVIDER, so this is just the stored label.
const VERIFY_PROVIDER_ID = 1

// Known ID Game Checker game slugs — a datalist so ops pick a verified slug
// without typos, while still allowing any slug the provider supports. Only
// single-parameter (id-only) games belong here: multi-param games like Mobile
// Legends (id+zone) / Genshin (id+server) can't be verified by the single-id
// lookup, so they stay collect-only (no verification).
const KNOWN_GAME_SLUGS = ['pubgm-global', 'dfm-garena', 'free-fire']

export default function ProductEditPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const can = useCan()
  const canManage = can('products.manage')
  const { id } = useParams<{ id: string }>()
  const isNew = !id
  const { data, isLoading, isError, refetch } = useProduct(id)
  const create = useCreateProduct()
  const update = useUpdateProduct(id ?? '')
  const uploadImage = useUploadProductImage()

  const { data: catsRes } = useProductCategories()
  const cats = useMemo(() => catsRes?.data ?? [], [catsRes])

  const [langTab, setLangTab] = useState<Locale>('en')
  const [title, setTitle] = useState({ en: '', ar: '', tr: '' })
  const [description, setDescription] = useState({ en: '', ar: '', tr: '' })
  const [category, setCategory] = useState('')
  const [ff, setFf] = useState<FfKey>('code')
  const [active, setActive] = useState(false)
  const [stock, setStock] = useState(0)
  const [variants, setVariants] = useState<VariantRow[]>([
    { denomination: '', price: '', resellerPrice: '', faceValue: '' },
  ])
  // Mobile-recharge (bridge) delivery — a sub-mode of account_credit. When on,
  // the product fulfills via the Android bridge device instead of the manual queue.
  const [bridgeOn, setBridgeOn] = useState(false)
  const [bridgeProvider, setBridgeProvider] = useState<'touch' | 'alfa'>('touch')
  const [bridgeMethod, setBridgeMethod] = useState<'transfer_credit' | 'recharge_line'>(
    'transfer_credit',
  )
  // Input-field specs are edited labels-only; everything else is preserved as
  // loaded so a save never mangles legacy fulfillment config.
  const [inputFields, setInputFields] = useState<InputField[]>([])
  // Purchase-time ID verification (check_name): toggle + provider game slug.
  const [verifyEnabled, setVerifyEnabled] = useState(false)
  const [verifyApp, setVerifyApp] = useState('')
  const [images, setImages] = useState<string[]>([])
  const [thumbnail, setThumbnail] = useState('')
  const [imageError, setImageError] = useState('')
  const imageFileRef = useRef<HTMLInputElement>(null)

  // Populate from the loaded product when editing.
  useEffect(() => {
    const p = data?.data
    if (!p) return
    setTitle({ en: p.title.en, ar: p.title.ar, tr: p.title.tr })
    setDescription({ en: p.description?.en ?? '', ar: p.description?.ar ?? '', tr: p.description?.tr ?? '' })
    setCategory(p.category)
    setFf(ffKey(p.fulfillmentType))
    setActive(p.available)
    setStock(p.stock)
    setVariants(
      p.variants.length
        ? p.variants.map((v) => ({
            id: v.id,
            denomination: v.denomination,
            price: String(v.price),
            resellerPrice: v.resellerPrice != null ? String(v.resellerPrice) : '',
            faceValue: v.faceValue != null ? String(v.faceValue) : '',
          }))
        : [{ denomination: '', price: '', resellerPrice: '', faceValue: '' }]
    )
    const isBridge = p.fulfillmentMode === 'bridge_device'
    setBridgeOn(isBridge)
    if (p.bridge) {
      setBridgeProvider(p.bridge.provider)
      setBridgeMethod(p.bridge.method)
    }
    setInputFields(p.inputFields ?? [])
    setVerifyEnabled(!!p.verification)
    setVerifyApp(p.verification?.app ?? '')
    setImages(p.images ?? [])
    setThumbnail(p.thumbnail ?? '')
  }, [data])

  // Default a brand-new product to the first real category once the list loads.
  useEffect(() => {
    if (isNew && !category && cats.length > 0) setCategory(cats[0].value)
  }, [isNew, category, cats])

  // Options = the real categories, always including the product's current value
  // (so editing a product whose category has no siblings still shows it).
  const catOptions = useMemo(() => {
    const values = cats.map((c) => c.value)
    if (category && !values.includes(category)) values.unshift(category)
    return values
  }, [cats, category])

  const pending = create.isPending || update.isPending || uploadImage.isPending
  const error = create.error || update.error

  const buildInput = () => {
    // Verification: send the config when enabled with a slug; send the clear
    // sentinel ({provider:0, app:''}) when disabling a product that had it; omit
    // otherwise so the backend's nil-guard leaves stored data untouched.
    const hadVerification = !!data?.data?.verification
    const verification =
      verifyEnabled && verifyApp.trim()
        ? { provider: VERIFY_PROVIDER_ID, app: verifyApp.trim() }
        : hadVerification
          ? { provider: 0, app: '' }
          : undefined

    // Bridge (mobile recharge) is a sub-mode of account_credit. When enabled we
    // send fulfillmentMode=bridge_device + the spec, reconcile the input fields to
    // a single first-positioned `phone` field, and carry each variant's transfer
    // face value. When it's off but the product used to be a bridge, we send the
    // clear sentinels.
    const isBridge = ff === 'credit' && bridgeOn
    const wasBridge = data?.data?.fulfillmentMode === 'bridge_device'
    const fields = isBridge ? reconcileBridgePhoneField(inputFields) : inputFields

    return {
      title,
      description,
      category,
      images,
      thumbnail,
      fulfillmentType: toFulfillment(ff),
      ...(isBridge
        ? { fulfillmentMode: 'bridge_device' as const, bridge: { provider: bridgeProvider, method: bridgeMethod } }
        : wasBridge
          ? { fulfillmentMode: 'manual_operator' as const, bridge: { provider: '' as const, method: '' as const } }
          : {}),
      available: active,
      stock,
      variants: variants
        .filter((v) => v.denomination.trim() && v.price.trim())
        .map((v) => ({
          // Echo the existing variant id so the backend keeps it; omit for new
          // rows (never send an empty string — the ObjectID JSON decode rejects it).
          ...(v.id ? { id: v.id } : {}),
          denomination: v.denomination,
          price: parseFloat(v.price) || 0,
          ...(v.resellerPrice.trim() ? { resellerPrice: parseFloat(v.resellerPrice) || 0 } : {}),
          ...(isBridge && bridgeMethod === 'transfer_credit' && v.faceValue.trim()
            ? { faceValue: parseFloat(v.faceValue) || 0 }
            : {}),
        })),
      // Send the full array whenever the product has fields now or had them
      // before (so removing them all clears the stored config); omit only when
      // there were never any, leaving stored data untouched via the nil-guard.
      ...(fields.length || data?.data?.inputFields?.length ? { inputFields: fields } : {}),
      ...(verification ? { verification } : {}),
    }
  }

  const onImageFile = (file: File) => {
    setImageError('')
    if (file.size > MAX_IMAGE_BYTES) {
      setImageError('Image must be 10 MB or smaller.')
      return
    }
    if (!['image/jpeg', 'image/png', 'image/webp'].includes(file.type)) {
      setImageError('Image must be JPEG, PNG, or WebP.')
      return
    }
    uploadImage.mutate(file, {
      onSuccess: (res) => {
        if (!res.data) return
        setImages([res.data.imageUrl])
        setThumbnail(res.data.thumbnailUrl)
      },
      onError: (err) => setImageError((err as Error).message),
    })
  }

  const onRemoveImage = () => {
    setImages([])
    setThumbnail('')
    setImageError('')
  }

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
        {canManage && (
          <button className="abtn primary" onClick={onSave} disabled={pending}>
            <Icon name="check" size={15} /> {pending ? 'Saving…' : t('save')}
          </button>
        )}
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
              <textarea
                className="afield"
                value={description[langTab]}
                onChange={(e) => setDescription({ ...description, [langTab]: e.target.value })}
                placeholder="Describe the product, redemption steps, region…"
                dir={langTab === 'ar' ? 'rtl' : 'ltr'}
              />
              <div className="ahint">Localized description shown to customers in each language.</div>
            </div>
            <div style={{ marginTop: 16 }}>
              <label className="alabel">Category</label>
              <select className="select" style={{ width: '100%' }} value={category} onChange={(e) => setCategory(e.target.value)}>
                {catOptions.map((c) => (
                  <option key={c} value={c}>
                    {categoryLabel(c)}
                  </option>
                ))}
              </select>
            </div>
          </div>

          {/* Fulfillment selector */}
          <div className="acard pad">
            <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 4 }}>Fulfillment type</h3>
            <div className="ahint" style={{ marginTop: 0, marginBottom: 14 }}>
              Determines how the product is delivered — this changes the fields below.
            </div>
            <div className="g3">
              {FF_OPTIONS.map(([k, , color, label, desc]) => {
                // A mobile-recharge bridge product must stay account_credit — the
                // bridge spec is a sub-mode of credit. Lock the other types while
                // the bridge is on so a stray click can't flip it to Code/PIN and
                // silently wipe the bridge config on save.
                const locked = bridgeOn && k !== 'credit'
                return (
                  <div
                    key={k}
                    onClick={() => {
                      if (!locked) setFf(k)
                    }}
                    title={
                      locked
                        ? 'Turn off the mobile-recharge bridge below to change the fulfillment type.'
                        : undefined
                    }
                    style={{
                      cursor: locked ? 'not-allowed' : 'pointer',
                      opacity: locked ? 0.45 : 1,
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
                )
              })}
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
                  onClick={() => setVariants([...variants, { denomination: '', price: '', resellerPrice: '', faceValue: '' }])}
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
                    {ff === 'credit' && bridgeOn && bridgeMethod === 'transfer_credit' && (
                      <th>Transfer amount</th>
                    )}
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
                      {ff === 'credit' && bridgeOn && bridgeMethod === 'transfer_credit' && (
                        <td>
                          <input
                            className="afield"
                            style={{ padding: '7px 10px', width: 100 }}
                            value={v.faceValue}
                            placeholder="5"
                            onChange={(e) => {
                              const next = [...variants]
                              next[i] = { ...v, faceValue: e.target.value }
                              setVariants(next)
                            }}
                          />
                        </td>
                      )}
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

          {/* Input fields — define what the customer enters at checkout */}
          <div className="acard">
            <div className="panelhead">
              <Icon name="layers" size={17} />
              <h3>Input fields</h3>
              <div className="ph-act">
                <button
                  className="abtn xs"
                  onClick={() =>
                    setInputFields([
                      ...inputFields,
                      { key: '', label: { en: '', ar: '' }, type: 'text', sensitive: false },
                    ])
                  }
                >
                  <Icon name="plus" size={13} /> Add field
                </button>
              </div>
            </div>
            <div className="ahint" style={{ margin: '0 0 10px' }}>
              Fields the customer fills at checkout (e.g. Account ID, Zone ID, Email). Values are
              attached to the order for fulfillment. Keys must be unique. <b>Sensitive</b> fields
              are masked and never stored on the order. <code>select</code> options aren't editable
              here yet — use <code>text</code> for now.
            </div>
            {inputFields.length === 0 ? (
              <div className="ahint" style={{ margin: 0 }}>
                No input fields — the customer enters nothing at checkout.
              </div>
            ) : (
              <div className="tablewrap">
                <table className="tbl">
                  <thead>
                    <tr>
                      <th>Key</th>
                      <th>Type</th>
                      <th>Label (EN)</th>
                      <th>Label (AR)</th>
                      <th>Sensitive</th>
                      <th></th>
                    </tr>
                  </thead>
                  <tbody>
                    {inputFields.map((f, i) => {
                      const patch = (u: Partial<InputField>) => {
                        const next = [...inputFields]
                        next[i] = { ...f, ...u }
                        setInputFields(next)
                      }
                      return (
                        <tr key={i}>
                          <td>
                            <input
                              className="afield"
                              style={{ padding: '7px 10px', width: 120 }}
                              value={f.key}
                              placeholder="accountId"
                              dir="ltr"
                              onChange={(e) => patch({ key: e.target.value })}
                            />
                          </td>
                          <td>
                            <select
                              className="select"
                              value={f.type}
                              onChange={(e) => patch({ type: e.target.value as InputField['type'] })}
                            >
                              {(['text', 'select', 'amount', 'quantity'] as const).map((t) => (
                                <option key={t} value={t}>
                                  {t}
                                </option>
                              ))}
                            </select>
                          </td>
                          <td>
                            <input
                              className="afield"
                              style={{ padding: '7px 10px', minWidth: 150 }}
                              value={f.label.en}
                              dir="ltr"
                              onChange={(e) => patch({ label: { ...f.label, en: e.target.value } })}
                            />
                          </td>
                          <td>
                            <input
                              className="afield"
                              style={{ padding: '7px 10px', minWidth: 150 }}
                              value={f.label.ar}
                              dir="rtl"
                              onChange={(e) => patch({ label: { ...f.label, ar: e.target.value } })}
                            />
                          </td>
                          <td style={{ textAlign: 'center' }}>
                            <input
                              type="checkbox"
                              checked={!!f.sensitive}
                              onChange={(e) => patch({ sensitive: e.target.checked })}
                            />
                          </td>
                          <td>
                            <span
                              className="iact danger"
                              onClick={() => setInputFields(inputFields.filter((_, j) => j !== i))}
                            >
                              <Icon name="trash" size={15} />
                            </span>
                          </td>
                        </tr>
                      )
                    })}
                  </tbody>
                </table>
              </div>
            )}
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
              <div className="ahint" style={{ marginTop: 0 }}>
                Customers enter their player/account ID at checkout; top-ups are credited
                automatically via the provider API.
              </div>

              {/* Mobile-recharge (bridge) delivery */}
              <div
                style={{
                  marginTop: 14,
                  paddingTop: 14,
                  borderTop: '1px solid var(--border)',
                }}
              >
                <label style={{ display: 'flex', alignItems: 'center', gap: 10, cursor: 'pointer' }}>
                  <input
                    type="checkbox"
                    checked={bridgeOn}
                    onChange={(e) => setBridgeOn(e.target.checked)}
                  />
                  <b style={{ fontSize: 13.5 }}>Deliver via mobile-recharge bridge (Lebanon)</b>
                </label>
                <div className="ahint" style={{ margin: '6px 0 0' }}>
                  Fulfill this top-up automatically on a dual-SIM bridge phone (MTC Touch / Alfa)
                  instead of the manual queue. Adds a <code>phone</code> field at checkout. Manage
                  devices under <b>Bridge</b>.
                </div>
                {bridgeOn && (
                  <div style={{ display: 'flex', gap: 16, flexWrap: 'wrap', marginTop: 12 }}>
                    <div>
                      <label className="alabel">Operator</label>
                      <select
                        className="select"
                        value={bridgeProvider}
                        onChange={(e) => setBridgeProvider(e.target.value as 'touch' | 'alfa')}
                      >
                        <option value="touch">MTC Touch</option>
                        <option value="alfa">Alfa</option>
                      </select>
                    </div>
                    <div>
                      <label className="alabel">Method</label>
                      <select
                        className="select"
                        value={bridgeMethod}
                        onChange={(e) =>
                          setBridgeMethod(e.target.value as 'transfer_credit' | 'recharge_line')
                        }
                      >
                        <option value="transfer_credit">Credit transfer (by amount)</option>
                        <option value="recharge_line">Scratch card (from inventory)</option>
                      </select>
                    </div>
                  </div>
                )}
                {bridgeOn && bridgeMethod === 'transfer_credit' && (
                  <div className="ahint" style={{ margin: '10px 0 0' }}>
                    Set each variant's <b>Transfer amount</b> above — the credit sent to the
                    customer's line (distinct from the price they pay).
                  </div>
                )}
                {bridgeOn && bridgeMethod === 'recharge_line' && (
                  <div className="ahint" style={{ margin: '10px 0 0' }}>
                    Scratch-card codes are claimed from this product's code <b>Inventory</b> — upload
                    codes there, one applied per order.
                  </div>
                )}
              </div>
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

          {/* Purchase-time ID verification (check_name) */}
          <div className="acard pad">
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 4 }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                <Icon name="shield" size={17} />
                <h3 style={{ fontSize: 15, fontWeight: 800 }}>Purchase-time ID verification</h3>
              </div>
              <Toggle on={verifyEnabled} onClick={() => setVerifyEnabled(!verifyEnabled)} />
            </div>
            <div className="ahint" style={{ marginTop: 0 }}>
              When on, customers enter their game ID on the product page and must see the resolved
              nickname before they can buy — catching wrong IDs before payment. Best for
              account-credit top-ups (PUBG, Delta Force, …).
            </div>
            {verifyEnabled && (
              <div style={{ marginTop: 14 }}>
                <label className="alabel">Game (provider slug)</label>
                <input
                  className="afield"
                  list="game-slugs"
                  value={verifyApp}
                  onChange={(e) => setVerifyApp(e.target.value)}
                  placeholder="e.g. pubgm-global"
                />
                <datalist id="game-slugs">
                  {KNOWN_GAME_SLUGS.map((s) => (
                    <option key={s} value={s} />
                  ))}
                </datalist>
                <div className="ahint">
                  Must exactly match the ID Game Checker slug for this game (e.g. <b>pubgm-global</b>,{' '}
                  <b>dfm-garena</b>). Add a text input field for the player ID too, so it shows at checkout.
                </div>
                {!verifyApp.trim() && (
                  <div style={{ color: 'var(--danger)', fontSize: 12.5, fontWeight: 600, marginTop: 6 }}>
                    Enter a game slug, or turn verification off.
                  </div>
                )}
              </div>
            )}
          </div>
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

          <div className="acard pad">
            <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 12 }}>Image</h3>
            <input
              ref={imageFileRef}
              type="file"
              accept={IMAGE_ACCEPT}
              style={{ display: 'none' }}
              onChange={(e) => {
                const file = e.target.files?.[0]
                e.target.value = ''
                if (file) onImageFile(file)
              }}
            />
            {(() => {
              const preview = thumbnail || images[0]
              if (preview) {
                return (
                  <>
                    <img
                      src={preview}
                      alt=""
                      className="imgslot"
                      style={{ width: '100%', height: 140, objectFit: 'cover' }}
                    />
                    <div style={{ display: 'flex', gap: 8, marginTop: 10 }}>
                      <button
                        className="abtn sm"
                        disabled={uploadImage.isPending || !canManage}
                        onClick={() => imageFileRef.current?.click()}
                      >
                        <Icon name="upload" size={13} /> Replace
                      </button>
                      <button className="abtn sm danger" disabled={uploadImage.isPending || !canManage} onClick={onRemoveImage}>
                        <Icon name="x" size={13} /> Remove
                      </button>
                    </div>
                  </>
                )
              }
              return (
                <button
                  className="abtn sm"
                  style={{ width: '100%', height: 140 }}
                  disabled={uploadImage.isPending || !canManage}
                  onClick={() => imageFileRef.current?.click()}
                >
                  <Icon name="upload" size={13} /> {uploadImage.isPending ? 'Uploading…' : 'Choose image'}
                </button>
              )
            })()}
            <div className="ahint">JPEG, PNG, or WebP · max 10 MB · compressed to 1024px + 256px thumbnail.</div>
            {imageError && (
              <div style={{ color: 'var(--danger)', fontSize: 12.5, fontWeight: 600, marginTop: 6 }}>{imageError}</div>
            )}
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

        </div>
      </div>
    </div>
  )
}
