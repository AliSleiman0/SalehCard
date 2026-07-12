import { apiClient } from '@/lib/api-client'
import type { ApiResponse } from '@/types'

// ApiReview mirrors the Go review.ReviewRow JSON returned by
// GET /api/v1/products/{id}/reviews (only approved reviews are surfaced).
export interface ApiReview {
  id: string
  productId: string
  userId: string
  rating: number
  body: string
  verifiedPurchase: boolean
  status: string
  createdAt: string
  productName: string
  art: string
  userName: string
}

export async function fetchReviews(productId: string): Promise<ApiResponse<ApiReview[]>> {
  return apiClient.get<ApiReview[]>(`/api/v1/products/${productId}/reviews`)
}

export interface MyReview {
  reviewed: boolean
  status?: string
}

export async function fetchMyReview(productId: string): Promise<ApiResponse<MyReview>> {
  return apiClient.get<MyReview>(`/api/v1/reviews/mine?productId=${encodeURIComponent(productId)}`)
}

export interface SubmitReviewInput {
  productId: string
  rating: number
  body: string
}

export async function submitReview(input: SubmitReviewInput): Promise<ApiResponse<{ id: string }>> {
  return apiClient.post<{ id: string }>('/api/v1/reviews', input)
}
