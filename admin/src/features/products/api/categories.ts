import { apiClient } from '@/lib/api-client'
import type { ApiResponse } from '@/types'

/** One distinct product category and how many products carry it. `value` is the
 *  exact string the product list filter matches on. */
export interface CategoryFacet {
  value: string
  count: number
}

export function listProductCategories(): Promise<ApiResponse<CategoryFacet[]>> {
  return apiClient.get<CategoryFacet[]>('/api/admin/products/categories')
}

/** Humanize a category slug for display, e.g. "giftcards" → "Giftcards",
 *  "account_credit" → "Account credit". */
export function categoryLabel(slug: string): string {
  const s = slug.replace(/[_-]+/g, ' ').trim()
  return s ? s.charAt(0).toUpperCase() + s.slice(1) : slug
}
