import { useEffect, useMemo, useRef, useState } from 'react'
import { Trans, useTranslation } from 'react-i18next'
import { useNavigate, useParams } from 'react-router-dom'
import { Icon, PageHead, Toggle, StatusBadge, LoadingSpinner, ErrorState, ffKey, type FfKey } from '@/components'
import { useGoBack } from '@/lib/goBack'
import { useProductCategories } from '../hooks/useCategories'
import { categoryLabel } from '../api/categories'
import { useCategoryTree } from '@/features/categories/hooks/useCategories'
import { orderedTree, pathLabel } from '@/features/categories/api/categories'
import {
  useProduct,
  useCreateProduct,
  useUpdateProduct,
  useUploadProductImage,
  useFulfillmentProviders,
  useRestoreProduct,
} from '../hooks/useProducts'
import { reconcileBridgePhoneField } from '../lib/bridgeFields'
import { toRows, fromRows, rowIssues, type InputFieldRow } from '../lib/fieldRows'
import {
  toForm as toUnitForm,
  fromForm as fromUnitForm,
  issues as unitIssues,
  amountRows,
  amountIgnoredByPrice,
  driverRow,
  enableRows,
  disableRows,
  droppedPackages,
  fixedDenomination,
  preview as unitPreview,
  unitDenomination,
  type UnitPricingForm,
  type UnitRowsStash,
} from '../lib/unitPricing'
import { useCan } from '@/stores/auth'
import { toast } from '@/stores/toast'
import { SupplierLinksTable } from '../components/SupplierLinksTable'
import { linkIssues, toLinkRows, toLinksInput, type LinkRow } from '../lib/supplierLinks'
import { quoted } from '@/lib/quoted'
import { apiErrorText } from '@/lib/apiErrorText'
import type { FulfillmentType, Locale, InputFieldType } from '@/types'

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

// [type, class, colour, label key, description key] — labels are looked up at render.
const FF_OPTIONS: [FfKey, string, string, string, string][] = [
  ['code', 'ff-code', 'var(--ff-code)', 'ff_code', 'prod_ff_code_desc'],
  ['credit', 'ff-credit', 'var(--ff-credit)', 'ff_credit', 'prod_ff_credit_desc'],
  ['transfer', 'ff-transfer', 'var(--ff-transfer)', 'ff_transfer', 'prod_ff_transfer_desc'],
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
  const goBack = useGoBack('/products')
  const can = useCan()
  const canManage = can('products.manage')
  const { id } = useParams<{ id: string }>()
  const isNew = !id
  const { data, isLoading, isError, refetch } = useProduct(id)
  const create = useCreateProduct()
  const update = useUpdateProduct(id ?? '')
  const uploadImage = useUploadProductImage()
  const restore = useRestoreProduct()
  // A trashed product stays fully editable (so it can be organised into the right
  // category before it goes back on sale) — the banner just makes its state clear.
  const trashed = !isNew && !!data?.data?.deletedAt

  const { data: catsRes } = useProductCategories()
  const cats = useMemo(() => catsRes?.data ?? [], [catsRes])

  // Managed taxonomy tree — the product is assigned to a node (any tier). The
  // legacy free-text facet select below is the fallback when no taxonomy exists.
  const { data: catTreeRes } = useCategoryTree()
  const catTree = useMemo(() => catTreeRes?.data ?? [], [catTreeRes])
  const catTreeById = useMemo(() => new Map(catTree.map((n) => [n.id, n])), [catTree])
  const catTreeOptions = useMemo(() => orderedTree(catTree), [catTree])
  const hasTree = catTreeOptions.length > 0

  const [langTab, setLangTab] = useState<Locale>('en')
  const [title, setTitle] = useState({ en: '', ar: '', tr: '' })
  const [description, setDescription] = useState({ en: '', ar: '', tr: '' })
  const [category, setCategory] = useState('')
  const [categoryId, setCategoryId] = useState('')
  const [ff, setFf] = useState<FfKey>('code')
  const [active, setActive] = useState(false)
  const [stock, setStock] = useState(0)
  // Admin-set unit cost. String-backed so an empty field stays empty (not "0").
  const [cost, setCost] = useState('')
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
  // Upstream supplier (api-mode) delivery: orders are placed automatically at a
  // wholesale panel supplier. Mutually exclusive with the bridge checkbox —
  // both imply fulfillmentMode. The supplier links table lists every supplier
  // product it can be fulfilled by; it is sent only when edited (rev-guarded).
  const [apiOn, setApiOn] = useState(false)
  const [linkRows, setLinkRows] = useState<LinkRow[]>([])
  const [linksRev, setLinksRev] = useState(0)
  const [linksDirty, setLinksDirty] = useState(false)
  // Automatic fallback to the next supplier link (on by default).
  const [fallbackOn, setFallbackOn] = useState(true)
  const { data: provRes } = useFulfillmentProviders()
  const providerOptions = provRes?.data ?? []
  // Input-field specs: key, type, labels, and constraints (min/max, select
  // options) are all editable; legacyName rides along untouched. Numeric
  // bounds live as strings while editing (fieldRows.ts).
  const [inputFields, setInputFields] = useState<InputFieldRow[]>([])
  const [fieldIssues, setFieldIssues] = useState<string[]>([])
  // Purchase-time ID verification (check_name): toggle + provider game slug.
  const [verifyEnabled, setVerifyEnabled] = useState(false)
  const [verifyApp, setVerifyApp] = useState('')
  // Money-transfer buy/sell rate board (transfer products only). String-backed so
  // partial typing (currencies, "89." …) stays literal; parsed on save.
  const [mt, setMt] = useState({ base: '', quote: '', buy: '', sell: '', min: '', max: '' })
  const [mtError, setMtError] = useState('')
  // Per-unit ("customer enters amount") pricing: the single variant prices a
  // block of N units and an amount-type input field drives the line price.
  // String-backed (unitPricing.ts); honored on account-credit products (manual
  // or supplier-fulfilled) and money transfers.
  const [unit, setUnit] = useState<UnitPricingForm>(() => toUnitForm())
  const [unitNotice, setUnitNotice] = useState('')
  // What switching to per-unit changed in the input fields (removed quantity
  // rows, auto-added amount row), kept until save so switching back restores it.
  const [unitStash, setUnitStash] = useState<UnitRowsStash | null>(null)
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
    setCategoryId(p.categoryId ?? '')
    setFf(ffKey(p.fulfillmentType))
    setActive(p.available)
    setStock(p.stock)
    setCost(p.pricing?.cost != null ? String(p.pricing.cost) : '')
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
    setApiOn(p.fulfillmentMode === 'api')
    setLinkRows(toLinkRows(p.supplierLinks, p.activeSupplierLinkId))
    setLinksRev(p.supplierLinksRev ?? 0)
    setLinksDirty(false)
    setFallbackOn(p.supplierFallback ?? true)
    setInputFields(toRows(p.inputFields ?? []))
    setVerifyEnabled(!!p.verification)
    setVerifyApp(p.verification?.app ?? '')
    const m = p.moneyTransfer
    setMt({
      base: m?.baseCurrency ?? '',
      quote: m?.quoteCurrency ?? '',
      buy: m?.buyRate != null ? String(m.buyRate) : '',
      sell: m?.sellRate != null ? String(m.sellRate) : '',
      min: m?.minAmount != null ? String(m.minAmount) : '',
      max: m?.maxAmount != null ? String(m.maxAmount) : '',
    })
    setUnit(toUnitForm(p))
    setUnitNotice('')
    setUnitStash(null)
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

  // Per-unit pricing is offered on account-credit products fulfilled by hand or
  // by a supplier (the supplier is sent the typed amount as its quantity); the
  // bridge has its own qty semantics, so the server rejects it there and any
  // other configuration is forced back to fixed packages.
  // Money transfers can be sold by amount too (send 1,000 → pay 1,000 × rate);
  // bridge/API dispatch only exist on credit products, so they never block it.
  const unitEligible = (ff === 'credit' && !bridgeOn) || ff === 'transfer'
  const perUnit = unit.enabled && unitEligible
  const unitForm: UnitPricingForm = { ...unit, enabled: perUnit }
  // A fixed-price credit product that still collects an amount charges the
  // package price whatever the customer types — warn while editing.
  const ignoredAmounts = amountIgnoredByPrice({
    fulfillmentType: ff === 'credit' ? 'account_credit' : ff,
    unitPricing: perUnit ? { blockUnits: 1 } : null,
    inputFields,
  })

  // Switching to per-unit preps the input fields: quantity fields are removed
  // (pack count contradicts typed amount) and an amount field is added when
  // there is none, for the admin to bound.
  const setPricingMode = (toPerUnit: boolean) => {
    if (toPerUnit === perUnit) return
    setUnitNotice('')
    if (!toPerUnit) {
      // Undo what per-unit changed before it was saved, and drop the auto
      // "per N" label from the remaining package (an admin-typed label stays).
      setInputFields((rows) => disableRows(rows, unitStash))
      setUnitStash(null)
      setVariants((vs) =>
        vs.length ? [{ ...vs[0], denomination: fixedDenomination(vs[0].denomination, unit) }, ...vs.slice(1)] : vs,
      )
      setUnit((u) => ({ ...u, enabled: false }))
      return
    }
    const { rows, removed, stash } = enableRows(inputFields)
    setInputFields(rows)
    setUnitStash(stash)
    const driver = driverRow(unit, rows)
    setUnit((u) => ({ ...u, enabled: true, fieldKey: driver?.key ?? '' }))
    if (removed.length)
      setUnitNotice(t('prod_unit_removed_qty', { count: removed.length, fields: quoted(removed) }))
  }
  // The amount range customers may type on a per-unit supplier product — the
  // picker checks it against the supplier's accepted quantities.
  const supplierAmountRange = useMemo(() => {
    if (!perUnit) return undefined
    const d = driverRow(unit, inputFields)
    if (!d) return undefined
    const n = (s: string) => (s.trim() && Number.isFinite(Number(s)) ? Number(s) : undefined)
    return { min: n(d.min), max: n(d.max) }
  }, [perUnit, unit, inputFields])
  // Per-unit keeps only the first price row: warn about the packages a save deletes.
  const droppedPkgs = perUnit ? droppedPackages(variants) : []

  const pending = create.isPending || update.isPending || uploadImage.isPending
  const error = create.error || update.error

  // Supplier-API delivery — shared by the code and credit fulfillment cards
  // (panels deliver both top-ups and codes). Checking it unchecks the bridge
  // (both imply fulfillmentMode).
  const supplierSection = (
    <div style={{ marginTop: 14, paddingTop: 14, borderTop: '1px solid var(--border)' }}>
      <label style={{ display: 'flex', alignItems: 'center', gap: 10, cursor: 'pointer' }}>
        <input
          type="checkbox"
          checked={apiOn}
          onChange={(e) => {
            setApiOn(e.target.checked)
            if (e.target.checked) setBridgeOn(false)
          }}
        />
        <b style={{ fontSize: 13.5 }}>{t('prod_api_toggle')}</b>
      </label>
      <div className="ahint" style={{ margin: '6px 0 0' }}>
        {t('prod_api_hint')}
      </div>
      {apiOn && providerOptions.length > 0 && (
        <div style={{ marginTop: 12 }}>
          <SupplierLinksTable
            rows={linkRows}
            onChange={(rows) => {
              setLinkRows(rows)
              setLinksDirty(true)
            }}
            providers={providerOptions}
            sellPrice={parseFloat(variants[0]?.price ?? '') || 0}
            blockUnits={perUnit ? Number(unit.blockUnits) || 1 : 1}
            amountRange={supplierAmountRange}
            searchHint={title.en}
          />
          <label style={{ display: 'flex', alignItems: 'center', gap: 10, cursor: 'pointer', marginTop: 12 }}>
            <input
              type="checkbox"
              checked={fallbackOn}
              onChange={(e) => setFallbackOn(e.target.checked)}
            />
            <b style={{ fontSize: 13.5 }}>{t('prod_fallback_toggle')}</b>
          </label>
          <div className="ahint" style={{ margin: '6px 0 0' }}>
            {t('prod_fallback_hint')}
          </div>
        </div>
      )}
      {apiOn && ff === 'credit' && !perUnit && ignoredAmounts.length > 0 && (
        <div className="ahint" style={{ margin: '10px 0 0', color: 'var(--warn)' }}>
          {t('prod_api_amount_warn')}
        </div>
      )}
      {apiOn && providerOptions.length === 0 && (
        <div className="ahint" style={{ margin: '10px 0 0' }}>
          <Trans i18nKey="prod_no_suppliers" components={{ c: <code /> }} />
        </div>
      )}
    </div>
  )

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
    // Supplier API delivery applies to credit AND code products (panels deliver
    // top-ups and codes alike); bridge wins when both are somehow on. Turning it
    // off sends the mode the backend would otherwise derive — a code product
    // must fall back to `inventory`, not `manual_operator`, or its code-pool
    // dispatch would break. The supplier links stay (not sent unless edited).
    const isApi = (ff === 'credit' || ff === 'code') && apiOn && !isBridge
    const wasApi = data?.data?.fulfillmentMode === 'api'
    // Convert rows to the API shape first so reconcileBridgePhoneField keeps
    // operating on plain InputField[].
    const apiFields = fromRows(inputFields)
    const fields = isBridge ? reconcileBridgePhoneField(apiFields) : apiFields

    // Money-transfer rate board (transfer products only). Send the spec when the
    // admin filled currencies/rates; send the clear sentinel ({baseCurrency:'',
    // quoteCurrency:''}) when a product that had rates now has none (incl. a
    // switch away from transfer); omit otherwise so the backend leaves it alone.
    const hadMt = !!data?.data?.moneyTransfer
    const mtHasContent =
      ff === 'transfer' &&
      !!(mt.base.trim() || mt.quote.trim() || mt.buy.trim() || mt.sell.trim())
    const moneyTransfer = mtHasContent
      ? {
          baseCurrency: mt.base.trim().toUpperCase(),
          quoteCurrency: mt.quote.trim().toUpperCase(),
          ...(mt.buy.trim() ? { buyRate: parseFloat(mt.buy) || 0 } : {}),
          ...(mt.sell.trim() ? { sellRate: parseFloat(mt.sell) || 0 } : {}),
          ...(mt.min.trim() ? { minAmount: parseFloat(mt.min) || 0 } : {}),
          ...(mt.max.trim() ? { maxAmount: parseFloat(mt.max) || 0 } : {}),
        }
      : hadMt
        ? { baseCurrency: '' as const, quoteCurrency: '' as const }
        : undefined

    // Per-unit pricing: send the spec when on; the clear sentinel when a product
    // that had it is back on fixed packages (incl. a switch to bridge/API/another
    // type); omit otherwise. Per-unit keeps only the first variant row (its id
    // preserved) and stamps its denomination ("per 1,000").
    const hadUnit = !!data?.data?.unitPricing
    const unitPricing = perUnit
      ? fromUnitForm(unitForm, inputFields)
      : hadUnit
        ? { fieldKey: '', blockUnits: 0 }
        : undefined
    const variantRows = perUnit
      ? [{ ...(variants[0] ?? { denomination: '', price: '', resellerPrice: '', faceValue: '' }), denomination: unitDenomination(unitForm) }]
      : variants

    return {
      title,
      description,
      category,
      // When a taxonomy node is picked, the server derives category + rootDomain
      // from it; the flat `category` above is the fallback (no taxonomy / "None").
      ...(categoryId ? { categoryId } : {}),
      images,
      thumbnail,
      fulfillmentType: toFulfillment(ff),
      ...(isBridge
        ? {
            fulfillmentMode: 'bridge_device' as const,
            bridge: { provider: bridgeProvider, method: bridgeMethod },
          }
        : isApi
          ? {
              fulfillmentMode: 'api' as const,
              ...(wasBridge ? { bridge: { provider: '' as const, method: '' as const } } : {}),
            }
          : wasBridge || wasApi
            ? {
                fulfillmentMode:
                  !wasBridge && ff === 'code' ? ('inventory' as const) : ('manual_operator' as const),
                ...(wasBridge ? { bridge: { provider: '' as const, method: '' as const } } : {}),
              }
            : {}),
      available: active,
      // The supplier links table is kept when supplier delivery is turned off
      // (orders then just don't go to a supplier); sent only when edited.
      ...(linksDirty ? { supplierLinks: toLinksInput(linkRows, linksRev) } : {}),
      // Sent only when changed (absent = on for a new product).
      ...(fallbackOn !== (data?.data?.supplierFallback ?? true) ? { supplierFallback: fallbackOn } : {}),
      stock,
      // Unit cost: send only when filled so a blank preserves the stored value.
      ...(cost.trim() !== '' ? { cost: Math.max(0, parseFloat(cost) || 0) } : {}),
      variants: variantRows
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
      ...(moneyTransfer ? { moneyTransfer } : {}),
      ...(unitPricing ? { unitPricing } : {}),
    }
  }

  const onImageFile = (file: File) => {
    setImageError('')
    if (file.size > MAX_IMAGE_BYTES) {
      setImageError(t('prod_img_too_big'))
      return
    }
    if (!['image/jpeg', 'image/png', 'image/webp'].includes(file.type)) {
      setImageError(t('prod_img_bad_type'))
      return
    }
    uploadImage.mutate(file, {
      onSuccess: (res) => {
        if (!res.data) return
        setImages([res.data.imageUrl])
        setThumbnail(res.data.thumbnailUrl)
      },
      onError: (err) => setImageError(apiErrorText(err, 'prod_img_upload_failed')),
    })
  }

  const onRemoveImage = () => {
    setImages([])
    setThumbnail('')
    setImageError('')
  }

  // Client-side money-transfer validation mirroring the server: currencies
  // required together, at least one positive rate, non-negative rates, min ≤ max.
  // Empty (no rates at all) is fine — it clears / omits. Returns '' when valid.
  const mtIssue = (): string => {
    if (ff !== 'transfer') return ''
    const hasContent = !!(
      mt.base.trim() || mt.quote.trim() || mt.buy.trim() || mt.sell.trim() || mt.min.trim() || mt.max.trim()
    )
    if (!hasContent) return ''
    if (!mt.base.trim() || !mt.quote.trim()) return t('prod_mt_need_currencies')
    const buy = parseFloat(mt.buy)
    const sell = parseFloat(mt.sell)
    if ((mt.buy.trim() && buy < 0) || (mt.sell.trim() && sell < 0)) return t('prod_mt_negative')
    const hasBuy = mt.buy.trim() !== '' && buy > 0
    const hasSell = mt.sell.trim() !== '' && sell > 0
    if (!hasBuy && !hasSell) return t('prod_mt_need_rate')
    const min = parseFloat(mt.min)
    const max = parseFloat(mt.max)
    if (mt.min.trim() && mt.max.trim() && min > 0 && max > 0 && min > max) return t('prod_mt_min_max')
    return ''
  }

  const onSave = () => {
    // Block the save on input-field problems the admin can fix in the table
    // (duplicate keys, min>max, optionless select …) — mirrors the server's
    // validation so a submit never bounces on a 400 the UI could have caught.
    const apiDelivery =
      (ff === 'credit' || ff === 'code') && apiOn && !(ff === 'credit' && bridgeOn) && providerOptions.length > 0
    const issues = [
      ...rowIssues(inputFields),
      ...unitIssues(unitForm, inputFields, variants),
      ...(linksDirty || (apiDelivery && linkRows.length === 0) ? linkIssues(linkRows, apiDelivery) : []),
    ]
    setFieldIssues(issues)
    if (issues.length) return
    const mtErr = mtIssue()
    setMtError(mtErr)
    if (mtErr) return
    const input = buildInput()
    const onSuccess = () => goBack()
    if (isNew) create.mutate(input, { onSuccess })
    else update.mutate(input, { onSuccess })
  }

  const margin = (v: VariantRow) => {
    const c = parseFloat(v.price)
    const r = parseFloat(v.resellerPrice)
    if (!c || !r) return 0
    return Math.round((1 - r / c) * 100)
  }

  const headTitle = useMemo(() => (isNew ? t('new_product') : data?.data?.title.en || t('prod_edit_title')), [isNew, data, t])

  if (!isNew && isLoading) return <div className="page"><LoadingSpinner /></div>
  if (!isNew && isError)
    return (
      <div className="page">
        <PageHead back="/products" title={t('nav_products')} />
        <ErrorState message={t('prod_load_error')} onRetry={() => refetch()} />
      </div>
    )

  return (
    <div className="page">
      <PageHead
        back="/products"
        crumbs={[t('nav_products'), isNew ? t('new_product') : headTitle]}
        title={isNew ? t('new_product') : t('prod_edit_title')}
        sub={isNew ? t('prod_new_sub') : id}
      >
        {canManage && (
          <button className="abtn primary" onClick={onSave} disabled={pending}>
            <Icon name="check" size={15} /> {pending ? t('prod_saving') : t('save')}
          </button>
        )}
      </PageHead>

      {trashed && (
        <div
          role="status"
          style={{
            marginBottom: 16,
            padding: '10px 14px',
            borderRadius: 'var(--ar-sm)',
            background: 'rgba(255,77,109,.12)',
            color: 'var(--danger)',
            fontSize: 13,
            fontWeight: 700,
            display: 'flex',
            alignItems: 'center',
            gap: 12,
          }}
        >
          <Icon name="trash" size={15} />
          <span style={{ flex: 1 }}>
            {t('prod_trash_banner')}
          </span>
          {canManage && (
            <button
              className="abtn xs ok"
              disabled={restore.isPending}
              onClick={() =>
                restore.mutate(id!, {
                  onSuccess: () => toast.success(t('prod_restored')),
                  onError: () => toast.error(t('prod_restore_failed')),
                })
              }
            >
              <Icon name="refresh" size={13} /> {t('prod_restore')}
            </button>
          )}
        </div>
      )}

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
          {apiErrorText(error, 'prod_save_failed')}
        </div>
      )}

      {fieldIssues.length > 0 && (
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
          {fieldIssues.map((msg) => (
            <div key={msg}>{msg}</div>
          ))}
        </div>
      )}

      <div className="formgrid">
        <div className="fieldset">
          {/* Basics */}
          <div className="acard pad">
            <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 16 }}>{t('prod_basics')}</h3>
            <div className="langtab">
              {(['en', 'ar', 'tr'] as Locale[]).map((l) => (
                <button key={l} className={langTab === l ? 'on' : ''} onClick={() => setLangTab(l)}>
                  {l.toUpperCase()}
                </button>
              ))}
            </div>
            <label className="alabel">{t('prod_title_label', { lang: langTab.toUpperCase() })}</label>
            <input
              className="afield"
              value={title[langTab]}
              onChange={(e) => setTitle({ ...title, [langTab]: e.target.value })}
              placeholder={langTab === 'ar' ? 'اسم المنتج بالعربية' : langTab === 'tr' ? 'Ürün adı' : 'Product title'}
              dir={langTab === 'ar' ? 'rtl' : 'ltr'}
            />
            <div className="ahint">{t('prod_title_hint')}</div>
            <div style={{ marginTop: 16 }}>
              <label className="alabel">{t('prod_desc_label', { lang: langTab.toUpperCase() })}</label>
              <textarea
                className="afield"
                value={description[langTab]}
                onChange={(e) => setDescription({ ...description, [langTab]: e.target.value })}
                placeholder={t('prod_desc_ph')}
                dir={langTab === 'ar' ? 'rtl' : 'ltr'}
              />
              <div className="ahint">{t('prod_desc_hint')}</div>
            </div>
            <div style={{ marginTop: 16 }}>
              <label className="alabel">{t('prod_category')}</label>
              {hasTree ? (
                <select className="select" style={{ width: '100%' }} value={categoryId} onChange={(e) => setCategoryId(e.target.value)}>
                  <option value="">{t('prod_cat_none')}</option>
                  {catTreeOptions.map((n) => (
                    <option key={n.id} value={n.id}>
                      {pathLabel(n, catTreeById)}
                    </option>
                  ))}
                </select>
              ) : (
                <select className="select" style={{ width: '100%' }} value={category} onChange={(e) => setCategory(e.target.value)}>
                  {catOptions.map((c) => (
                    <option key={c} value={c}>
                      {categoryLabel(c)}
                    </option>
                  ))}
                </select>
              )}
              <div className="ahint">{t('prod_cat_hint')}</div>
            </div>
          </div>

          {/* Fulfillment selector */}
          <div className="acard pad">
            <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 4 }}>{t('prod_ff_title')}</h3>
            <div className="ahint" style={{ marginTop: 0, marginBottom: 14 }}>
              {t('prod_ff_hint')}
            </div>
            <div className="g3">
              {FF_OPTIONS.map(([k, , color, labelKey, descKey]) => {
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
                    title={locked ? t('prod_ff_locked') : undefined}
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
                      <b style={{ fontSize: 13.5 }}>{t(labelKey)}</b>
                      {ff === k && <Icon name="checkc" size={16} />}
                    </div>
                    <div style={{ fontSize: 12, color: 'var(--text-dim)', lineHeight: 1.4 }}>{t(descKey)}</div>
                  </div>
                )
              })}
            </div>
          </div>

          {/* Variants builder */}
          <div className="acard">
            <div className="panelhead">
              <Icon name="layers" size={17} />
              <h3>{t('prod_variants_title')}</h3>
              {!perUnit && (
                <div className="ph-act">
                  <button
                    className="abtn xs"
                    onClick={() => setVariants([...variants, { denomination: '', price: '', resellerPrice: '', faceValue: '' }])}
                  >
                    <Icon name="plus" size={13} /> {t('prod_add_variant')}
                  </button>
                </div>
              )}
            </div>
            {unitEligible && (
              <div style={{ padding: '0 18px 12px' }}>
                <div className="aseg">
                  <button className={!perUnit ? 'on' : ''} onClick={() => setPricingMode(false)}>
                    {t('prod_mode_fixed')}
                  </button>
                  <button className={perUnit ? 'on' : ''} onClick={() => setPricingMode(true)}>
                    {t('prod_mode_unit')}
                  </button>
                </div>
                <div className="ahint" style={{ margin: '6px 0 0' }}>
                  {perUnit
                    ? ff === 'transfer'
                      ? t('prod_mode_unit_hint_transfer')
                      : t('prod_mode_unit_hint_credit')
                    : t('prod_mode_fixed_hint')}
                </div>
                {unitNotice && (
                  <div style={{ color: 'var(--warn)', fontSize: 12.5, fontWeight: 600, marginTop: 6 }}>{unitNotice}</div>
                )}
                {droppedPkgs.length > 0 && (
                  <div style={{ color: 'var(--warn)', fontSize: 12.5, fontWeight: 600, marginTop: 6 }}>
                    {t('prod_dropped_pkgs', { count: droppedPkgs.length, list: quoted(droppedPkgs) })}
                  </div>
                )}
              </div>
            )}
            {ignoredAmounts.length > 0 && (
              <div
                style={{ padding: '0 18px 12px', color: 'var(--warn)', fontSize: 12.5, fontWeight: 600 }}
                role="alert"
              >
                {t('prod_ignored_amounts', { fields: quoted(ignoredAmounts) })}
                {unitEligible ? t('prod_ignored_fix_unit') : t('prod_ignored_fix_type')}
              </div>
            )}
            {perUnit ? (
              <div style={{ padding: '0 18px 18px' }}>
                {(() => {
                  const v = variants[0] ?? { denomination: '', price: '', resellerPrice: '', faceValue: '' }
                  const setV = (u: Partial<VariantRow>) => setVariants([{ ...v, ...u }, ...variants.slice(1)])
                  const amounts = amountRows(inputFields)
                  const driver = driverRow(unit, inputFields)
                  const line = unitPreview(unitForm, inputFields, v.price)
                  return (
                    <>
                      <div style={{ display: 'flex', gap: 16, flexWrap: 'wrap', alignItems: 'flex-end' }}>
                        <div>
                          <label className="alabel">{t('prod_customer_price')}</label>
                          <input
                            className="afield"
                            style={{ width: 110 }}
                            inputMode="decimal"
                            value={v.price}
                            onChange={(e) => setV({ price: e.target.value })}
                          />
                        </div>
                        <div>
                          <label className="alabel">{t('prod_reseller_price')}</label>
                          <input
                            className="afield"
                            style={{ width: 110 }}
                            inputMode="decimal"
                            value={v.resellerPrice}
                            onChange={(e) => setV({ resellerPrice: e.target.value })}
                          />
                        </div>
                        <div>
                          <label className="alabel">{t('prod_per_units')}</label>
                          <input
                            className="afield"
                            style={{ width: 110 }}
                            inputMode="decimal"
                            placeholder="1000"
                            value={unit.blockUnits}
                            onChange={(e) => setUnit((u) => ({ ...u, blockUnits: e.target.value }))}
                          />
                        </div>
                        <div>
                          <label className="alabel">{t('prod_amount_field')}</label>
                          <select
                            className="select"
                            value={driver?.key ?? ''}
                            onChange={(e) => setUnit((u) => ({ ...u, fieldKey: e.target.value }))}
                          >
                            {amounts.length === 0 && <option value="">{t('prod_add_amount_field_opt')}</option>}
                            {amounts.map((f, i) => (
                              <option key={i} value={f.key}>
                                {f.label.en || f.key || t('prod_field_n', { n: i + 1 })}
                              </option>
                            ))}
                          </select>
                        </div>
                        <div style={{ paddingBottom: 8 }}>
                          <span className="st st-ok" style={{ fontSize: 11 }}>
                            {margin(v)}%
                          </span>
                        </div>
                      </div>
                      <div className="ahint" style={{ margin: '10px 0 0' }}>
                        {line || t('prod_preview_hint')}
                      </div>
                    </>
                  )
                })()}
              </div>
            ) : (
            <div className="tablewrap">
              <table className="tbl">
                <thead>
                  <tr>
                    <th>{ff === 'credit' ? t('prod_denomination') : t('prod_face_value')}</th>
                    <th>{t('prod_customer_price')}</th>
                    <th>{t('prod_reseller_price')}</th>
                    {ff === 'credit' && bridgeOn && bridgeMethod === 'transfer_credit' && (
                      <th>{t('prod_transfer_amount')}</th>
                    )}
                    <th>{t('prod_margin')}</th>
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
            )}
          </div>

          {/* Input fields — define what the customer enters at checkout */}
          <div className="acard">
            <div className="panelhead">
              <Icon name="layers" size={17} />
              <h3>{t('prod_fields_title')}</h3>
              <div className="ph-act">
                <button
                  className="abtn xs"
                  onClick={() =>
                    setInputFields([
                      ...inputFields,
                      { key: '', label: { en: '', ar: '' }, type: 'text', sensitive: false, min: '', max: '', options: [] },
                    ])
                  }
                >
                  <Icon name="plus" size={13} /> {t('prod_add_field')}
                </button>
              </div>
            </div>
            <div className="ahint" style={{ margin: '0 0 10px' }}>
              <Trans i18nKey="prod_fields_hint" components={{ b: <b />, c: <code /> }} />
            </div>
            {inputFields.length === 0 ? (
              <div className="ahint" style={{ margin: 0 }}>
                {t('prod_fields_empty')}
              </div>
            ) : (
              <div className="tablewrap">
                <table className="tbl">
                  <thead>
                    <tr>
                      <th>{t('prod_col_key')}</th>
                      <th>{t('col_type')}</th>
                      <th>{t('prod_col_label_en')}</th>
                      <th>{t('prod_col_label_ar')}</th>
                      <th>{t('prod_col_constraints')}</th>
                      <th>{t('prod_col_sensitive')}</th>
                      <th></th>
                    </tr>
                  </thead>
                  <tbody>
                    {inputFields.map((f, i) => {
                      const patch = (u: Partial<InputFieldRow>) => {
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
                              onChange={(e) => patch({ type: e.target.value as InputFieldType })}
                            >
                              {(['text', 'select', 'amount', 'quantity', 'file'] as const)
                                // Per-unit products are bought by amount — no pack
                                // count (kept only so a stray row still renders).
                                .filter((ft) => ft !== 'quantity' || !perUnit || f.type === 'quantity')
                                .map((ft) => (
                                  <option key={ft} value={ft}>
                                    {t(`prod_ftype_${ft}`)}
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
                          <td>
                            {f.type === 'quantity' || f.type === 'amount' ? (
                              <div style={{ display: 'flex', gap: 6, alignItems: 'center' }} dir="ltr">
                                <input
                                  className="afield"
                                  style={{ padding: '7px 10px', width: 70 }}
                                  inputMode="decimal"
                                  placeholder={t('prod_min')}
                                  value={f.min}
                                  onChange={(e) => patch({ min: e.target.value })}
                                />
                                <span style={{ opacity: 0.5 }}>–</span>
                                <input
                                  className="afield"
                                  style={{ padding: '7px 10px', width: 70 }}
                                  inputMode="decimal"
                                  placeholder={t('prod_max')}
                                  value={f.max}
                                  onChange={(e) => patch({ max: e.target.value })}
                                />
                              </div>
                            ) : f.type === 'select' ? (
                              <OptionsEditor value={f.options} onChange={(options) => patch({ options })} />
                            ) : (
                              <span style={{ opacity: 0.4 }}>—</span>
                            )}
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
                  {t('prod_code_inventory')}
                </span>
                <h3 style={{ fontSize: 15, fontWeight: 800 }}>{t('prod_codes_available', { n: stock })}</h3>
                <div style={{ marginInlineStart: 'auto' }}>
                  <button
                    className="abtn sm primary"
                    disabled={isNew}
                    title={isNew ? t('prod_save_first') : undefined}
                    onClick={() => id && navigate(`/inventory?upload=${id}`)}
                  >
                    <Icon name="upload" size={14} /> {t('prod_upload_codes')}
                  </button>
                </div>
              </div>
              <div className="ahint" style={{ marginTop: 0 }}>
                {t('prod_code_hint')}
              </div>
              {/* Supplier-API delivery: codes come from the upstream supplier
                  instead of this product's inventory pool. */}
              {supplierSection}
            </div>
          )}
          {ff === 'credit' && (
            <div className="acard pad" style={{ borderColor: 'var(--ff-credit-bd)' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 10 }}>
                <span className="bdg ff-credit">
                  <i className="d" />
                  {t('ff_credit')}
                </span>
                <h3 style={{ fontSize: 15, fontWeight: 800 }}>{t('prod_player_id_title')}</h3>
              </div>
              <div className="ahint" style={{ marginTop: 0 }}>
                {t('prod_credit_hint')}
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
                    onChange={(e) => {
                      setBridgeOn(e.target.checked)
                      // Mutually exclusive with supplier-API delivery — both
                      // imply the product's fulfillmentMode.
                      if (e.target.checked) setApiOn(false)
                    }}
                  />
                  <b style={{ fontSize: 13.5 }}>{t('prod_bridge_toggle')}</b>
                </label>
                <div className="ahint" style={{ margin: '6px 0 0' }}>
                  <Trans i18nKey="prod_bridge_hint" components={{ b: <b />, c: <code /> }} />
                </div>
                {bridgeOn && (
                  <div style={{ display: 'flex', gap: 16, flexWrap: 'wrap', marginTop: 12 }}>
                    <div>
                      <label className="alabel">{t('prod_operator')}</label>
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
                      <label className="alabel">{t('col_method')}</label>
                      <select
                        className="select"
                        value={bridgeMethod}
                        onChange={(e) =>
                          setBridgeMethod(e.target.value as 'transfer_credit' | 'recharge_line')
                        }
                      >
                        <option value="transfer_credit">{t('prod_bridge_transfer')}</option>
                        <option value="recharge_line">{t('prod_bridge_scratch')}</option>
                      </select>
                    </div>
                  </div>
                )}
                {bridgeOn && bridgeMethod === 'transfer_credit' && (
                  <div className="ahint" style={{ margin: '10px 0 0' }}>
                    <Trans i18nKey="prod_bridge_transfer_hint" components={{ b: <b /> }} />
                  </div>
                )}
                {bridgeOn && bridgeMethod === 'recharge_line' && (
                  <div className="ahint" style={{ margin: '10px 0 0' }}>
                    <Trans i18nKey="prod_bridge_scratch_hint" components={{ b: <b /> }} />
                  </div>
                )}
              </div>

              {/* Supplier-API delivery (alternative to the bridge / manual queue) */}
              {supplierSection}
            </div>
          )}
          {ff === 'transfer' && (
            <div className="acard pad" style={{ borderColor: 'var(--ff-transfer-bd)' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 10 }}>
                <span className="bdg ff-transfer">
                  <i className="d" />
                  {t('ff_transfer')}
                </span>
                <h3 style={{ fontSize: 15, fontWeight: 800 }}>{t('prod_service_settings')}</h3>
              </div>
              <div className="ahint" style={{ marginTop: 0 }}>
                {t('prod_transfer_hint')}
              </div>

              {/* Buy/sell exchange-rate board — shown as a rate board + live
                  calculator on the product page. */}
              <div style={{ marginTop: 16, paddingTop: 16, borderTop: '1px solid var(--border)' }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 4 }}>
                  <Icon name="coins" size={17} />
                  <h3 style={{ fontSize: 15, fontWeight: 800 }}>{t('prod_rates_title')}</h3>
                </div>
                <div className="ahint" style={{ marginTop: 0 }}>
                  {t('prod_rates_hint')}
                </div>
                <div className="g2" style={{ marginTop: 12 }}>
                  <div>
                    <label className="alabel">{t('prod_base_currency')}</label>
                    <input
                      className="afield"
                      maxLength={8}
                      placeholder={t('prod_eg', { v: 'USD' })}
                      value={mt.base}
                      onChange={(e) => setMt((s) => ({ ...s, base: e.target.value }))}
                    />
                  </div>
                  <div>
                    <label className="alabel">{t('prod_quote_currency')}</label>
                    <input
                      className="afield"
                      maxLength={8}
                      placeholder={t('prod_eg', { v: 'LBP' })}
                      value={mt.quote}
                      onChange={(e) => setMt((s) => ({ ...s, quote: e.target.value }))}
                    />
                  </div>
                </div>
                <div className="g2" style={{ marginTop: 12 }}>
                  <div>
                    <label className="alabel">{t('prod_buy_rate')}</label>
                    <input
                      className="afield"
                      type="number"
                      min={0}
                      placeholder={t('prod_eg', { v: '89000' })}
                      value={mt.buy}
                      onChange={(e) => setMt((s) => ({ ...s, buy: e.target.value }))}
                    />
                  </div>
                  <div>
                    <label className="alabel">{t('prod_sell_rate')}</label>
                    <input
                      className="afield"
                      type="number"
                      min={0}
                      placeholder={t('prod_eg', { v: '89500' })}
                      value={mt.sell}
                      onChange={(e) => setMt((s) => ({ ...s, sell: e.target.value }))}
                    />
                  </div>
                </div>
                <div className="g2" style={{ marginTop: 12 }}>
                  <div>
                    <label className="alabel">{t('prod_min_amount')}</label>
                    <input
                      className="afield"
                      type="number"
                      min={0}
                      placeholder={t('prod_eg', { v: '1' })}
                      value={mt.min}
                      onChange={(e) => setMt((s) => ({ ...s, min: e.target.value }))}
                    />
                  </div>
                  <div>
                    <label className="alabel">{t('prod_max_amount')}</label>
                    <input
                      className="afield"
                      type="number"
                      min={0}
                      placeholder={t('prod_eg', { v: '5000' })}
                      value={mt.max}
                      onChange={(e) => setMt((s) => ({ ...s, max: e.target.value }))}
                    />
                  </div>
                </div>
                {mtError && (
                  <div style={{ color: 'var(--danger)', fontSize: 12.5, fontWeight: 600, marginTop: 8 }}>
                    {mtError}
                  </div>
                )}
              </div>
            </div>
          )}

          {/* Purchase-time ID verification (check_name) */}
          <div className="acard pad">
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 4 }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                <Icon name="shield" size={17} />
                <h3 style={{ fontSize: 15, fontWeight: 800 }}>{t('prod_verify_title')}</h3>
              </div>
              <Toggle on={verifyEnabled} onClick={() => setVerifyEnabled(!verifyEnabled)} />
            </div>
            <div className="ahint" style={{ marginTop: 0 }}>
              {t('prod_verify_hint')}
            </div>
            {verifyEnabled && (
              <div style={{ marginTop: 14 }}>
                <label className="alabel">{t('prod_verify_game')}</label>
                <input
                  className="afield"
                  list="game-slugs"
                  value={verifyApp}
                  onChange={(e) => setVerifyApp(e.target.value)}
                  placeholder={t('prod_eg', { v: 'pubgm-global' })}
                />
                <datalist id="game-slugs">
                  {KNOWN_GAME_SLUGS.map((s) => (
                    <option key={s} value={s} />
                  ))}
                </datalist>
                <div className="ahint">
                  <Trans i18nKey="prod_verify_slug_hint" components={{ b: <b /> }} />
                </div>
                {!verifyApp.trim() && (
                  <div style={{ color: 'var(--danger)', fontSize: 12.5, fontWeight: 600, marginTop: 6 }}>
                    {t('prod_verify_need_slug')}
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
              <h3 style={{ fontSize: 15, fontWeight: 800 }}>{t('status')}</h3>
              <Toggle on={active} onClick={() => setActive(!active)} />
            </div>
            <StatusBadge s={active ? 'active' : 'draft'} />
            <div className="ahint">
              {active ? t('prod_status_on_hint') : t('prod_status_off_hint')}
            </div>
          </div>

          <div className="acard pad">
            <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 12 }}>{t('prod_image')}</h3>
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
                        <Icon name="upload" size={13} /> {t('prod_img_replace')}
                      </button>
                      <button className="abtn sm danger" disabled={uploadImage.isPending || !canManage} onClick={onRemoveImage}>
                        <Icon name="x" size={13} /> {t('prod_img_remove')}
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
                  <Icon name="upload" size={13} /> {uploadImage.isPending ? t('prod_uploading') : t('prod_img_choose')}
                </button>
              )
            })()}
            <div className="ahint">{t('prod_img_hint')}</div>
            {imageError && (
              <div style={{ color: 'var(--danger)', fontSize: 12.5, fontWeight: 600, marginTop: 6 }}>{imageError}</div>
            )}
          </div>

          {ff === 'code' && (
            <div className="acard pad">
              <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 12 }}>{t('prod_stock')}</h3>
              <label className="alabel">{t('prod_codes_stock')}</label>
              <input
                className="afield"
                type="number"
                value={stock}
                onChange={(e) => setStock(parseInt(e.target.value, 10) || 0)}
              />
              <div className="ahint">{t('prod_stock_hint')}</div>
            </div>
          )}

          <div className="acard pad">
            <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 12 }}>{t('prod_cost')}</h3>
            <label className="alabel">{t('prod_unit_cost')}</label>
            <input
              className="afield"
              type="number"
              min={0}
              step="0.01"
              value={cost}
              onChange={(e) => setCost(e.target.value)}
              placeholder="—"
            />
            <div className="ahint">{t('prod_cost_hint')}</div>
          </div>

        </div>
      </div>
    </div>
  )
}

// OptionsEditor edits a select field's choices as removable chips plus a text
// box: Enter or comma commits the typed option (trimmed; blanks and exact
// duplicates ignored), and the box also commits on blur so a typed-but-not-
// entered option isn't silently lost on save.
function OptionsEditor({ value, onChange }: { value: string[]; onChange: (options: string[]) => void }) {
  const { t } = useTranslation()
  const [draft, setDraft] = useState('')

  const commit = () => {
    const opt = draft.trim()
    setDraft('')
    if (!opt || value.includes(opt)) return
    onChange([...value, opt])
  }

  return (
    <div style={{ display: 'flex', flexWrap: 'wrap', gap: 4, alignItems: 'center', minWidth: 160 }} dir="ltr">
      {value.map((opt) => (
        <span
          key={opt}
          className="bdg"
          style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}
        >
          {opt}
          <span className="iact danger" style={{ lineHeight: 0 }} onClick={() => onChange(value.filter((o) => o !== opt))}>
            <Icon name="x" size={11} />
          </span>
        </span>
      ))}
      <input
        className="afield"
        style={{ padding: '5px 8px', width: 90, fontSize: 12 }}
        placeholder={t('prod_opt_add')}
        value={draft}
        onChange={(e) => setDraft(e.target.value)}
        onBlur={commit}
        onKeyDown={(e) => {
          if (e.key === 'Enter' || e.key === ',') {
            e.preventDefault()
            commit()
          }
        }}
      />
    </div>
  )
}
