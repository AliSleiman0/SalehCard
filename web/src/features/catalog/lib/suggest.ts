import type { Product } from '@/types'

/** One search suggestion row. */
export interface Suggestion {
  id: string
  name: string
  image?: string
}

const collator = new Intl.Collator(undefined, { numeric: true, sensitivity: 'base' })

/** Display name of a product for the active locale (falls back to English). */
export function productName(p: Pick<Product, 'title'>, locale: string): string {
  const t = p.title as unknown as Record<string, string | undefined>
  return (t[locale] || t.en || t.ar || '').trim()
}

/**
 * Suggestions for what the customer has typed so far: products whose name
 * contains the letters (case-insensitive), names that START with them first,
 * each group A to Z (client feedback #11/#86). Capped at `max`.
 */
export function buildSuggestions(products: Product[], term: string, locale: string, max = 8): Suggestion[] {
  const q = term.trim().toLowerCase()
  if (!q) return []
  const rows = products
    .map((p) => ({ p, name: productName(p, locale) }))
    .filter((x) => x.name && x.name.toLowerCase().includes(q))
  const seen = new Set<string>()
  return rows
    .sort((a, b) => {
      const sa = a.name.toLowerCase().startsWith(q) ? 0 : 1
      const sb = b.name.toLowerCase().startsWith(q) ? 0 : 1
      return sa - sb || collator.compare(a.name, b.name)
    })
    .filter((x) => (seen.has(x.p.id) ? false : (seen.add(x.p.id), true)))
    .slice(0, max)
    .map(({ p, name }) => ({ id: p.id, name, image: p.thumbnail || p.images?.[0] || undefined }))
}

/** Splits `name` around the first case-insensitive match of `term`, for bolding. */
export function highlight(name: string, term: string): [string, string, string] {
  const q = term.trim()
  const i = q ? name.toLowerCase().indexOf(q.toLowerCase()) : -1
  if (i < 0) return [name, '', '']
  return [name.slice(0, i), name.slice(i, i + q.length), name.slice(i + q.length)]
}
