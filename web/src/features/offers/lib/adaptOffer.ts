import { artForCategory } from '@/lib/art'
import type { ApiOffer } from '../api/offers'

export interface ViewOffer {
  id: string
  productId: string
  name: string
  image?: string
  art: string
  inStock: boolean
  label: string // "-25%" or "-$3"
  was: number
  now: number
  endsAt?: string
}

function discountLabel(o: ApiOffer): string {
  if (o.discountType === 'percent') return `-${o.discountValue}%`
  const v = o.discountValue % 1 === 0 ? o.discountValue.toFixed(0) : o.discountValue.toFixed(2)
  return `-$${v}`
}

export function adaptOffer(o: ApiOffer, locale: 'en' | 'ar' | 'tr'): ViewOffer {
  return {
    id: o.id,
    productId: o.productId,
    name: o.product.title[locale] || o.product.title.en,
    image: o.product.images?.[0] || undefined,
    art: artForCategory(o.product.category),
    inStock: o.product.inStock,
    label: discountLabel(o),
    was: o.originalFromPrice,
    now: o.offerFromPrice,
    endsAt: o.endsAt,
  }
}
