import { apiClient } from '@/lib/api-client'
import type { ApiResponse, PaginationMeta } from '@/types'

/** Lifecycle of an on-chain USDT payment intent (mirrors the backend). */
export type PaymentIntentStatus = 'pending' | 'confirming' | 'confirmed' | 'expired'

/** What a confirmed intent settled. */
export type PaymentPurpose = 'topup' | 'order'

/** A payment intent enriched with its owning customer (admin view). */
export interface AdminPaymentIntent {
  id: string
  userId: string
  purpose: PaymentPurpose
  orderId?: string
  network: string
  address: string
  amountUsd: number
  receivedUsd: number
  status: PaymentIntentStatus
  txHash?: string
  fromAddress?: string
  settlement?: string
  createdAt: string
  expiresAt: string
  confirmedAt?: string
  user: { email: string; phone: string }
}

export interface PaymentListParams {
  page?: number
  limit?: number
  status?: PaymentIntentStatus | ''
  purpose?: PaymentPurpose | ''
}

const ADMIN = '/api/admin/payments'

export function listPayments(
  params: PaymentListParams,
): Promise<ApiResponse<AdminPaymentIntent[]> & { meta?: PaginationMeta }> {
  const q = new URLSearchParams()
  if (params.page !== undefined) q.set('page', String(params.page))
  if (params.limit !== undefined) q.set('limit', String(params.limit))
  if (params.status) q.set('status', params.status)
  if (params.purpose) q.set('purpose', params.purpose)
  const qs = q.toString()
  return apiClient.get<AdminPaymentIntent[]>(qs ? `${ADMIN}?${qs}` : ADMIN)
}

export function getPayment(id: string): Promise<ApiResponse<AdminPaymentIntent>> {
  return apiClient.get<AdminPaymentIntent>(`${ADMIN}/${id}`)
}

/** Tronscan deep links for a transaction / address (mainnet). */
export const tronscanTx = (hash: string) => `https://tronscan.org/#/transaction/${hash}`
export const tronscanAddress = (addr: string) => `https://tronscan.org/#/address/${addr}`
