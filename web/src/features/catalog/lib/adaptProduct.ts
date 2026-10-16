import { artForCategory } from '@/lib/art'
import { qtyBounds, unitAmountField, unitLinePrice, type QtyBounds } from '@/lib/productFields'
import type { InputField, OfferInfo, Product } from '@/types'

export interface ViewVariant {
  id: string
  l: string
  p: number
  offerP?: number
  agentP?: number
}

export interface ViewUnitPricing {
  fieldKey: string
  blockUnits: number
  /** driving amount field's label (locale-resolved) */
  label: string
  /** amount bounds; max <= 0 = unbounded */
  minAmount: number
  maxAmount: number
}

export interface ViewOffer {
  /** discount chip text, e.g. "-25%" or "-$3" */
  label: string
  endsAt?: string
  from: number
  wasFrom: number
}

export interface ViewProduct {
  id: string
  brand: string
  title: string
  cat: string
  rootDomain: string
  art: string
  /** real product image for cards/lists (thumbnail preferred); art is the fallback */
  image?: string
  /** real product image for the detail page (full-size preferred) */
  imageFull?: string
  variants: ViewVariant[]
  rating: number
  reviews: number
  sold?: string
  instant: boolean
  available: boolean
  needsId: boolean
  idLabel?: string
  fulfill: 'code' | 'credit' | 'transfer'
  agentDisc: number
  /** customer-typed checkout fields defined on the product (legacy `quantity`
   * rows and a unit-priced product's driving amount field dropped) */
  inputFields: InputField[]
  /** purchase-quantity bounds (from the legacy `quantity` field, else 1..99) */
  qty: QtyBounds
  /** per-unit ("sold by amount") pricing, when the product is validly configured for it */
  unit?: ViewUnitPricing
  verifyEnabled: boolean
  offer?: ViewOffer
}

const CATEGORY_LABELS: Record<string, string> = {
  games: 'Game Top-up',
  gaming: 'Game Top-up',
  credits: 'App & Live Credits',
  recharge: 'Mobile Recharge',
  giftcards: 'Gift Card',
  gift: 'Gift Card',
  software: 'Software & Apps',
  crypto: 'Crypto',
  transfer: 'Money Transfer',
  finance: 'Money Transfer',
}

function humanize(cat: string): string {
  return (
    CATEGORY_LABELS[cat] ??
    cat
      .split(/[-_ ]+/)
      .filter(Boolean)
      .map((w) => w.charAt(0).toUpperCase() + w.slice(1))
      .join(' ')
  )
}

function discountLabel(o: OfferInfo): string {
  if (o.discountType === 'percent') return `-${o.discountValue}%`
  const v = o.discountValue % 1 === 0 ? o.discountValue.toFixed(0) : o.discountValue.toFixed(2)
  return `-$${v}`
}

function unitFrom(blockPrice: number, u: ViewUnitPricing): number {
  return unitLinePrice(blockPrice, u.minAmount > 0 ? u.minAmount : u.blockUnits, u.blockUnits)
}

export function adaptProduct(api: Product, locale: 'en' | 'ar' | 'tr'): ViewProduct {
  const brand = api.title[locale] || api.title.en
  const variants: ViewVariant[] = api.variants.map((v) => ({
    id: v.id,
    l: v.denomination,
    p: v.price,
    offerP: v.offerPrice,
    agentP: v.resellerPrice,
  }))

  const fulfill: 'code' | 'credit' | 'transfer' =
    api.fulfillmentType === 'account_credit'
      ? 'credit'
      : api.fulfillmentType === 'transfer'
        ? 'transfer'
        : 'code'

  const amountField = unitAmountField(api)
  const unit: ViewUnitPricing | undefined =
    amountField && api.unitPricing
      ? {
          fieldKey: amountField.key,
          blockUnits: api.unitPricing.blockUnits,
          label: amountField.label[locale] || amountField.label.en || amountField.key,
          minAmount: Math.max(0, amountField.constraints?.min ?? 0),
          maxAmount: Math.max(0, amountField.constraints?.max ?? 0),
        }
      : undefined

  // Customer-typed fields: legacy `quantity` rows (driven by the stepper) and a
  // unit-priced product's driving amount field (the amount box) are not typed.
  const inputFields = (api.inputFields ?? []).filter(
    (f) => f.type !== 'quantity' && f.key !== unit?.fieldKey,
  )
  // The hardcoded ID box is the fallback for products with no customer-typed
  // field of their own — counted after the filter above, so a product whose
  // only field is the amount (or a quantity row) still collects the player ID.
  const needsId = fulfill !== 'code' && inputFields.length === 0
  const v0 = api.variants[0]
  const agentDisc =
    v0 && v0.resellerPrice !== undefined && v0.price > 0
      ? +(1 - v0.resellerPrice / v0.price).toFixed(2)
      : 0

  return {
    id: api.id,
    brand,
    title: humanize(api.category),
    cat: api.category,
    rootDomain: api.rootDomain ?? '',
    art: artForCategory(api.category),
    image: api.thumbnail || api.images?.[0] || undefined,
    imageFull: api.images?.[0] || api.thumbnail || undefined,
    variants,
    rating: api.ratings.average,
    reviews: api.ratings.count,
    instant: api.fulfillmentType !== 'transfer',
    available: api.available,
    needsId,
    idLabel: needsId && fulfill === 'credit' ? 'Account / Player ID' : undefined,
    fulfill,
    agentDisc,
    inputFields,
    qty: qtyBounds(api),
    unit,
    verifyEnabled: !!api.verification,
    offer: api.offer
      ? {
          label: discountLabel(api.offer),
          endsAt: api.offer.endsAt,
          // Unit-priced: the server's from-prices are per block; rescale them to
          // the minimum amount so they line up with fromPrice() on the card.
          from: unit && v0 ? unitFrom(v0.offerPrice ?? v0.price, unit) : api.offer.offerFromPrice,
          wasFrom: unit && v0 ? unitFrom(v0.price, unit) : api.offer.originalFromPrice,
        }
      : undefined,
  }
}
