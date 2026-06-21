import { artForCategory } from '@/lib/art'
import type { Product } from '@/types'

export interface ViewVariant {
  id: string
  l: string
  p: number
  agentP?: number
}

export interface ViewProduct {
  id: string
  brand: string
  title: string
  cat: string
  rootDomain: string
  art: string
  variants: ViewVariant[]
  rating: number
  reviews: number
  sold?: string
  instant: boolean
  needsId: boolean
  idLabel?: string
  fulfill: 'code' | 'credit' | 'transfer'
  agentDisc: number
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

export function adaptProduct(api: Product, locale: 'en' | 'ar' | 'tr'): ViewProduct {
  const brand = api.title[locale] || api.title.en
  const variants: ViewVariant[] = api.variants.map((v) => ({
    id: v.id,
    l: v.denomination,
    p: v.price,
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
      : 0.08

  return {
    id: api.id,
    brand,
    title: humanize(api.category),
    cat: api.category,
    rootDomain: api.rootDomain ?? '',
    art: artForCategory(api.category),
    variants,
    rating: api.ratings.average || 4.8,
    reviews: api.ratings.count,
    instant: api.fulfillmentType !== 'transfer',
    needsId,
    idLabel: fulfill === 'credit' ? 'Account / Player ID' : undefined,
    fulfill,
    agentDisc,
  }
}
