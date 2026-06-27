import { apiClient } from '@/lib/api-client'
import type { ApiResponse, PaginationMeta } from '@/types'

/** Discount mechanism of a promo code. */
export type PromoType = 'percent' | 'fixed' | 'cashback'

/** Derived lifecycle status returned by the admin list. */
export type PromoStatus = 'active' | 'paused' | 'expired' | 'depleted'

/** A promo code as returned by the admin API (the model + a derived status). */
export interface AdminPromo {
  id: string
  code: string
  type: PromoType
  value: number
  minOrder: number
  maxUses: number
  uses: number
  startsAt?: string
  expiresAt?: string
  active: boolean
  status: PromoStatus
  createdAt: string
}

export interface PromoListParams {
  page?: number
  limit?: number
  type?: PromoType | ''
  status?: PromoStatus | ''
  q?: string
}

/** Create/update payload. Dates are ISO strings or omitted (no bound). */
export interface PromoInput {
  code: string
  type: PromoType
  value: number
  minOrder: number
  maxUses: number
  startsAt?: string | null
  expiresAt?: string | null
  active: boolean
}

const ADMIN = '/api/admin/promos'

export function listPromos(
  params: PromoListParams,
): Promise<ApiResponse<AdminPromo[]> & { meta?: PaginationMeta }> {
  const q = new URLSearchParams()
  if (params.page !== undefined) q.set('page', String(params.page))
  if (params.limit !== undefined) q.set('limit', String(params.limit))
  if (params.type) q.set('type', params.type)
  if (params.status) q.set('status', params.status)
  if (params.q) q.set('q', params.q)
  const qs = q.toString()
  return apiClient.get<AdminPromo[]>(qs ? `${ADMIN}?${qs}` : ADMIN)
}

export function getPromo(id: string): Promise<ApiResponse<AdminPromo>> {
  return apiClient.get<AdminPromo>(`${ADMIN}/${id}`)
}

export function createPromo(input: PromoInput): Promise<ApiResponse<AdminPromo>> {
  return apiClient.post<AdminPromo>(ADMIN, input)
}

export function updatePromo(id: string, input: PromoInput): Promise<ApiResponse<AdminPromo>> {
  return apiClient.put<AdminPromo>(`${ADMIN}/${id}`, input)
}

export function deletePromo(id: string): Promise<ApiResponse<{ deleted: boolean }>> {
  return apiClient.delete<{ deleted: boolean }>(`${ADMIN}/${id}`)
}
