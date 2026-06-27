import { apiClient } from '@/lib/api-client'
import type { ApiResponse, PaginationMeta } from '@/types'

/** Moderation state of a review. */
export type ReviewStatus = 'pending' | 'approved' | 'rejected'

/** A review as returned by the admin API (the model + joined product/user). */
export interface AdminReview {
  id: string
  productId: string
  productName: string
  art: string
  userId: string
  userName: string
  rating: number
  body: string
  verifiedPurchase: boolean
  status: ReviewStatus
  createdAt: string
}

export interface ReviewListParams {
  page?: number
  limit?: number
  status?: ReviewStatus | ''
  q?: string
}

const ADMIN = '/api/admin/reviews'

export function listReviews(
  params: ReviewListParams,
): Promise<ApiResponse<AdminReview[]> & { meta?: PaginationMeta }> {
  const q = new URLSearchParams()
  if (params.page !== undefined) q.set('page', String(params.page))
  if (params.limit !== undefined) q.set('limit', String(params.limit))
  if (params.status) q.set('status', params.status)
  if (params.q) q.set('q', params.q)
  const qs = q.toString()
  return apiClient.get<AdminReview[]>(qs ? `${ADMIN}?${qs}` : ADMIN)
}

export function setReviewStatus(id: string, status: ReviewStatus): Promise<ApiResponse<AdminReview>> {
  return apiClient.put<AdminReview>(`${ADMIN}/${id}`, { status })
}

export function deleteReview(id: string): Promise<ApiResponse<{ deleted: boolean }>> {
  return apiClient.delete<{ deleted: boolean }>(`${ADMIN}/${id}`)
}
