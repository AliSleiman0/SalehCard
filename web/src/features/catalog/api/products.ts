import { apiClient } from '@/lib/api-client'
import type { Product, ApiResponse, PaginationMeta } from '@/types'

export interface ProductsParams {
  page?: number
  limit?: number
  category?: string
  rootDomain?: string
  /** A taxonomy node: lists the node + all its subcategories' products. */
  categoryId?: string
  /** With categoryId: only products attached directly to that node. */
  directOnly?: boolean
  available?: boolean
  q?: string
  /** 'name' = natural A to Z on the server (ignored by older APIs). */
  sort?: 'name'
}

export async function fetchProducts(
  params: ProductsParams,
): Promise<ApiResponse<Product[]> & { meta?: PaginationMeta }> {
  const query = new URLSearchParams()

  if (params.page !== undefined) query.set('page', String(params.page))
  if (params.limit !== undefined) query.set('limit', String(params.limit))
  if (params.category !== undefined) query.set('category', params.category)
  if (params.rootDomain !== undefined) query.set('rootDomain', params.rootDomain)
  if (params.categoryId !== undefined) query.set('categoryId', params.categoryId)
  if (params.directOnly) query.set('directOnly', 'true')
  if (params.available !== undefined) query.set('available', String(params.available))
  if (params.q !== undefined) query.set('q', params.q)
  if (params.sort !== undefined) query.set('sort', params.sort)

  const qs = query.toString()
  const path = qs ? `/api/v1/products?${qs}` : '/api/v1/products'

  return apiClient.get<Product[]>(path) as Promise<ApiResponse<Product[]> & { meta?: PaginationMeta }>
}

export async function fetchProduct(id: string): Promise<ApiResponse<Product>> {
  return apiClient.get<Product>(`/api/v1/products/${id}`)
}
