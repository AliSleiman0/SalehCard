import { artForCategory } from '@/lib/art'
import type { InputField, OfferInfo, Product } from '@/types'

export interface ViewVariant {
  id: string
  l: string
  p: number
  offerP?: number
  agentP?: number
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
  /** non-quantity checkout fields defined on the product (legacy `quantity` rows dropped) */
  inputFields: InputField[]
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

  const needsId = fulfill !== 'code'
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
    idLabel: fulfill === 'credit' ? 'Account / Player ID' : undefined,
    fulfill,
    agentDisc,
    inputFields: (api.inputFields ?? []).filter((f) => f.type !== 'quantity'),
    verifyEnabled: !!api.verification,
    offer: api.offer
      ? {
          label: discountLabel(api.offer),
          endsAt: api.offer.endsAt,
          from: api.offer.offerFromPrice,
          wasFrom: api.offer.originalFromPrice,
        }
      : undefined,
  }
}
