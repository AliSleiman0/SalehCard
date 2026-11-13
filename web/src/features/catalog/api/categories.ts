import { apiClient } from '@/lib/api-client'
import type { Category, CategoryNodeView, ApiResponse } from '@/types'

export interface CategoriesParams {
  depth?: number
  rootDomain?: string
  parentLegacyId?: number
  /** Children of a tree node (its subcategories). */
  parentId?: string
  withCounts?: boolean
}

export async function fetchCategories(
  params: CategoriesParams = {},
): Promise<ApiResponse<Category[]>> {
  const query = new URLSearchParams()

  if (params.depth !== undefined) query.set('depth', String(params.depth))
  if (params.rootDomain !== undefined) query.set('rootDomain', params.rootDomain)
  if (params.parentLegacyId !== undefined) query.set('parentLegacyId', String(params.parentLegacyId))
  if (params.parentId !== undefined) query.set('parentId', params.parentId)
  if (params.withCounts !== undefined) query.set('withCounts', String(params.withCounts))

  const qs = query.toString()
  const path = qs ? `/api/v1/categories?${qs}` : '/api/v1/categories'

  return apiClient.get<Category[]>(path)
}

/** One visible category node + its breadcrumb (404 when hidden/unknown). */
export async function fetchCategoryNode(id: string): Promise<ApiResponse<CategoryNodeView>> {
  return apiClient.get<CategoryNodeView>(`/api/v1/categories/${encodeURIComponent(id)}`)
}
