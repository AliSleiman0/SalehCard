import { apiClient } from '@/lib/api-client'
import type { ApiResponse, PaginationMeta, I18nString } from '@/types'

/** Discount mechanism of an offer. */
export type DiscountType = 'percent' | 'fixed'

/** Derived lifecycle status returned by the admin list. */
export type OfferStatus = 'active' | 'scheduled' | 'expired' | 'paused'

/** The product slice the offer views carry (joined server-side). */
export interface OfferProduct {
  id: string
  title: I18nString
  images: string[]
  category: string
  fromPrice: number
  inStock: boolean
}

/** An offer as returned by the admin API (the model + derived status + product). */
export interface AdminOffer {
  id: string
  productId: string
  discountType: DiscountType
  discountValue: number
  startsAt?: string
  endsAt?: string
  active: boolean
  sortOrder: number
  status: OfferStatus
  product?: OfferProduct
  originalFromPrice: number
  offerFromPrice: number
  createdAt: string
  updatedAt: string
}

export interface OfferListParams {
  page?: number
  limit?: number
  status?: OfferStatus | ''
  productId?: string
}

/** Create/update payload. Dates are ISO strings or null (no bound). */
export interface OfferInput {
  productId: string
  discountType: DiscountType
  discountValue: number
  startsAt?: string | null
  endsAt?: string | null
  active: boolean
  sortOrder: number
}

const ADMIN = '/api/admin/offers'

export function listOffers(
  params: OfferListParams,
): Promise<ApiResponse<AdminOffer[]> & { meta?: PaginationMeta }> {
  const q = new URLSearchParams()
  if (params.page !== undefined) q.set('page', String(params.page))
  if (params.limit !== undefined) q.set('limit', String(params.limit))
  if (params.status) q.set('status', params.status)
  if (params.productId) q.set('productId', params.productId)
  const qs = q.toString()
  return apiClient.get<AdminOffer[]>(qs ? `${ADMIN}?${qs}` : ADMIN)
}

export function getOffer(id: string): Promise<ApiResponse<AdminOffer>> {
  return apiClient.get<AdminOffer>(`${ADMIN}/${id}`)
}

export function createOffer(input: OfferInput): Promise<ApiResponse<AdminOffer>> {
  return apiClient.post<AdminOffer>(ADMIN, input)
}

export function updateOffer(id: string, input: OfferInput): Promise<ApiResponse<AdminOffer>> {
  return apiClient.put<AdminOffer>(`${ADMIN}/${id}`, input)
}

export function deleteOffer(id: string): Promise<ApiResponse<{ deleted: boolean }>> {
  return apiClient.delete<{ deleted: boolean }>(`${ADMIN}/${id}`)
}
