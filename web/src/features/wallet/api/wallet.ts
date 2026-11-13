import { apiClient } from '@/lib/api-client'
import type { ApiResponse, WalletView } from '@/types'

/** Legacy out-of-band channels (kept for old request rows / USDT fallback). */
export type TopUpChannel = 'usdt' | 'whish' | 'omt' | 'cash' | 'other'

/** One customer input an admin defined on a manual top-up method. */
export interface TopUpMethodField {
  key: string
  label: string
  type: 'text' | 'number' | 'select' | 'file'
  required: boolean
  options?: string[]
  placeholder?: string
}

/** An admin-defined manual funding method (Bank transfer, Whish, Sham Cash…). */
export interface TopUpMethod {
  id: string
  name: string
  instructions: string
  enabled: boolean
  sortOrder: number
  fields: TopUpMethodField[]
}

export interface TopUpInput {
  amount: number
  /** Admin-defined method (preferred). */
  methodId?: string
  fields?: { key: string; value: string }[]
  /** Legacy bare channel — only for clients without methods. */
  channel?: TopUpChannel
  note?: string
}

/** A top-up request: filed as pending, credited only on admin approval. */
export interface TopUpRequest {
  id: string
  amount: number
  currency: string
  channel: string
  methodName?: string
  note?: string
  status: 'pending' | 'approved' | 'rejected'
  decisionReason?: string
  createdAt: string
}

export async function fetchWallet(): Promise<ApiResponse<WalletView>> {
  return apiClient.get<WalletView>('/api/v1/wallet')
}

export async function topUp(input: TopUpInput): Promise<ApiResponse<TopUpRequest>> {
  return apiClient.post<TopUpRequest>('/api/v1/wallet/topups', input)
}

export async function fetchTopUpRequests(): Promise<ApiResponse<TopUpRequest[]>> {
  return apiClient.get<TopUpRequest[]>('/api/v1/wallet/topups')
}

export async function fetchTopUpMethods(): Promise<ApiResponse<TopUpMethod[]>> {
  return apiClient.get<TopUpMethod[]>('/api/v1/wallet/topup-methods')
}

/** Uploads a transfer-proof image (multipart `image`) → its public URL, which
 * the customer submits as a file field's value. */
export async function uploadTopUpDocument(file: Blob): Promise<ApiResponse<{ url: string }>> {
  const form = new FormData()
  form.append('image', file, 'proof.jpg')
  return apiClient.postForm<{ url: string }>('/api/v1/wallet/topups/documents', form)
}
