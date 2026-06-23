import { apiClient } from '@/lib/api-client'
import type { Category, ApiResponse } from '@/types'

export interface CategoriesParams {
  depth?: number
  rootDomain?: string
  parentLegacyId?: number
  withCounts?: boolean
}

export async function fetchCategories(
  params: CategoriesParams = {},
): Promise<ApiResponse<Category[]>> {
  const query = new URLSearchParams()

  if (params.depth !== undefined) query.set('depth', String(params.depth))
  if (params.rootDomain !== undefined) query.set('rootDomain', params.rootDomain)
  if (params.parentLegacyId !== undefined) query.set('parentLegacyId', String(params.parentLegacyId))
  if (params.withCounts !== undefined) query.set('withCounts', String(params.withCounts))

  const qs = query.toString()
  const path = qs ? `/api/v1/categories?${qs}` : '/api/v1/categories'

  return apiClient.get<Category[]>(path)
}
