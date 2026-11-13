import type { Category, CategoryCrumb } from '@/types'
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
  // Admin-uploaded category image (rendered as the tile background); absent →
  // the curated gradient art.
  image?: string
  // Where the tile links: /category/<rootDomain> for roots, /catalog/<id> for
  // tree nodes (subcategory drill-down). Absent → /category/<key>.
  to?: string
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
    image: c.image || undefined,
  }
}

// localizedName picks a node's display name: the locale's value, else English,
// else the slug (Arabic legacy names live in name.en for some nodes).
export function localizedName(
  n: { name?: { en?: string; ar?: string; tr?: string }; slug?: string },
  locale: 'en' | 'ar' | 'tr',
): string {
  return n.name?.[locale]?.trim() || n.name?.en?.trim() || n.slug || ''
}

// adaptNode maps any taxonomy node to a tile. Roots keep the curated root
// presentation (and link to the whole-domain page); deeper nodes use their real
// name and link to the /catalog/<id> drill-down page.
export function adaptNode(c: Category, locale: 'en' | 'ar' | 'tr'): ViewCategory {
  if (c.depth === 0) return adaptRootCategory(c, locale)
  const meta = rootMeta(c.rootDomain)
  return {
    key: c.id,
    name: localizedName(c, locale),
    art: meta.art,
    tag: '',
    count: c.productCount,
    order: c.sortOrder ?? 0,
    image: c.image || undefined,
    to: `/catalog/${c.id}`,
  }
}

// crumbLink is where a breadcrumb ancestor links: roots go to their whole-domain
// page (same as the header nav), deeper nodes to their drill-down page.
export function crumbLink(c: CategoryCrumb): string {
  return c.depth === 0 && c.rootDomain ? `/category/${c.rootDomain}` : `/catalog/${c.id}`
}

// crumbName labels a breadcrumb: roots use the curated English label (legacy
// root names are inconsistent), everything else its localized name.
export function crumbName(c: CategoryCrumb, locale: 'en' | 'ar' | 'tr'): string {
  if (c.depth === 0 && c.rootDomain) {
    const meta = rootMeta(c.rootDomain)
    if (!(locale === 'ar' && c.name?.ar) && meta.label) return meta.label
  }
  return localizedName(c, locale)
}
