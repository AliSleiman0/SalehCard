import type { Product } from '@/types'

/** Display status derived from a product's flags + stock. */
export function productStatus(p: Product): 'active' | 'draft' | 'out' {
  if (!p.available) return 'draft'
  if (p.fulfillmentType === 'code' && p.stock <= 0) return 'out'
  return 'active'
}

/** Customer price range across variants, formatted "10.00 – 100.00". */
export function priceRange(p: Product): string {
  if (p.variants.length === 0) return '—'
  const prices = p.variants.map((v) => v.price)
  const min = Math.min(...prices)
  const max = Math.max(...prices)
  const fmt = (n: number) => n.toFixed(2)
  return min === max ? fmt(min) : `${fmt(min)} – ${fmt(max)}`
}
