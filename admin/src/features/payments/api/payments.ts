import { apiClient } from '@/lib/api-client'
import type { ApiResponse, PaginationMeta } from '@/types'

/** Lifecycle of an on-chain USDT payment intent (mirrors the backend). */
export type PaymentIntentStatus = 'pending' | 'confirming' | 'confirmed' | 'expired'

/** What a confirmed intent settled. */
export type PaymentPurpose = 'topup' | 'order'

/** How the intent's deposit address identifies the payer. */
export type AddressMode = 'derived' | 'shared'

/** A payment intent enriched with its owning customer (admin view). */
export interface AdminPaymentIntent {
  id: string
  userId: string
  purpose: PaymentPurpose
  orderId?: string
  network: string
  address: string
  addressMode: AddressMode
  amountUsd: number
  saltUsd?: number
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

/** Reconciliation state of a shared-address transfer no intent matched. */
export type DepositStatus = 'unmatched' | 'credited' | 'ignored'

/** A transfer into the shared deposit address the watcher couldn't match. */
export interface UnmatchedDeposit {
  id: string
  network: string
  txHash: string
  fromAddress: string
  toAddress: string
  amountUsd: number
  blockTime: string
  seenAt: string
  status: DepositStatus
  attributedUserId?: string
  attributedBy?: string
  attributedAt?: string
  note?: string
}

export interface DepositListParams {
  page?: number
  limit?: number
  status?: DepositStatus | ''
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

export function listDeposits(
  params: DepositListParams,
): Promise<ApiResponse<UnmatchedDeposit[]> & { meta?: PaginationMeta }> {
  const q = new URLSearchParams()
  if (params.page !== undefined) q.set('page', String(params.page))
  if (params.limit !== undefined) q.set('limit', String(params.limit))
  if (params.status) q.set('status', params.status)
  const qs = q.toString()
  return apiClient.get<UnmatchedDeposit[]>(qs ? `${ADMIN}/deposits?${qs}` : `${ADMIN}/deposits`)
}

/** Credit an unmatched deposit to a customer's wallet (id-hex or email). */
export function attributeDeposit(
  id: string,
  body: { user: string; note?: string },
): Promise<ApiResponse<UnmatchedDeposit>> {
  return apiClient.post<UnmatchedDeposit>(`${ADMIN}/deposits/${id}/attribute`, body)
}

/** Mark an unmatched deposit ignored (dust/spam/unknown sender). */
export function ignoreDeposit(
  id: string,
  body: { note?: string },
): Promise<ApiResponse<{ ignored: boolean }>> {
  return apiClient.post<{ ignored: boolean }>(`${ADMIN}/deposits/${id}/ignore`, body)
}

/** Tronscan deep links for a transaction / address (mainnet). */
export const tronscanTx = (hash: string) => `https://tronscan.org/#/transaction/${hash}`
export const tronscanAddress = (addr: string) => `https://tronscan.org/#/address/${addr}`
