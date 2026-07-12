import { apiClient } from '@/lib/api-client'
import type { Product, ApiResponse, PaginationMeta } from '@/types'

export interface ProductsParams {
  page?: number
  limit?: number
  category?: string
  rootDomain?: string
  available?: boolean
  q?: string
}

export async function fetchProducts(
  params: ProductsParams,
): Promise<ApiResponse<Product[]> & { meta?: PaginationMeta }> {
  const query = new URLSearchParams()

  if (params.page !== undefined) query.set('page', String(params.page))
  if (params.limit !== undefined) query.set('limit', String(params.limit))
  if (params.category !== undefined) query.set('category', params.category)
  if (params.rootDomain !== undefined) query.set('rootDomain', params.rootDomain)
  if (params.available !== undefined) query.set('available', String(params.available))
  if (params.q !== undefined) query.set('q', params.q)

  const qs = query.toString()
  const path = qs ? `/api/v1/products?${qs}` : '/api/v1/products'

  return apiClient.get<Product[]>(path) as Promise<ApiResponse<Product[]> & { meta?: PaginationMeta }>
}

export async function fetchProduct(id: string): Promise<ApiResponse<Product>> {
  return apiClient.get<Product>(`/api/v1/products/${id}`)
}
