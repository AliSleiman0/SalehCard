import { apiClient } from '@/lib/api-client'
import type { ApiResponse, PaginationMeta, UserRole } from '@/types'

/** Moderation state of a top-up request. */
export type TopUpStatus = 'pending' | 'approved' | 'rejected'

/** A top-up request enriched with its customer contact (admin view). */
export interface AdminTopUp {
  id: string
  userId: string
  amount: number
  currency: string
  channel: string
  note?: string
  status: TopUpStatus
  decidedBy?: string
  decisionReason?: string
  txId?: string
  createdAt: string
  decidedAt?: string
  customerEmail: string
  customerPhone?: string
  role?: UserRole
}

export interface TopUpListParams {
  page?: number
  limit?: number
  status?: TopUpStatus | ''
}

const ADMIN = '/api/admin/wallet/topups'

export function listTopUps(
  params: TopUpListParams,
): Promise<ApiResponse<AdminTopUp[]> & { meta?: PaginationMeta }> {
  const q = new URLSearchParams()
  if (params.page !== undefined) q.set('page', String(params.page))
  if (params.limit !== undefined) q.set('limit', String(params.limit))
  if (params.status) q.set('status', params.status)
  const qs = q.toString()
  return apiClient.get<AdminTopUp[]>(qs ? `${ADMIN}?${qs}` : ADMIN)
}

export function approveTopUp(id: string): Promise<ApiResponse<AdminTopUp>> {
  return apiClient.post<AdminTopUp>(`${ADMIN}/${id}/approve`)
}

export function rejectTopUp(id: string, reason: string): Promise<ApiResponse<AdminTopUp>> {
  return apiClient.post<AdminTopUp>(`${ADMIN}/${id}/reject`, { reason })
}
