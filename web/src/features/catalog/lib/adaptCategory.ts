import type { Category } from '@/types'
import { rootMeta } from '@/lib/categoryPresentation'

// ViewCategory is the shape the storefront tiles/nav render. `key` is the
// rootDomain — it drives both the /category/<key> route and the product
// browse filter (?rootDomain=<key>).
export interface ViewCategory {
  key: string
  name: string
  art: string
  tag: string
  count?: number
  order: number
}

// adaptRootCategory maps an API root category (depth 0) to a tile view. English
// uses the curated label (legacy en names are inconsistent); Arabic uses the
// real DB name. Counts and routing come straight from the API data.
export function adaptRootCategory(c: Category, locale: 'en' | 'ar' | 'tr'): ViewCategory {
  const meta = rootMeta(c.rootDomain)
  const name =
    locale === 'ar' && c.name?.ar ? c.name.ar : meta.label || c.name?.en || c.slug
  return {
    key: c.rootDomain || c.slug,
    name,
    art: meta.art,
    tag: meta.tag,
    count: c.productCount,
    order: meta.order,
  }
}
