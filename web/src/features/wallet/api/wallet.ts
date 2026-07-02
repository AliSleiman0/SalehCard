import { apiClient } from '@/lib/api-client'
import type { ApiResponse, WalletView } from '@/types'

/** Out-of-band payment channels accepted by the top-up request API. */
export type TopUpChannel = 'usdt' | 'whish' | 'omt' | 'cash' | 'other'

export interface TopUpInput {
  amount: number
  channel: TopUpChannel
  note?: string
}

/** A top-up request: filed as pending, credited only on admin approval. */
export interface TopUpRequest {
  id: string
  amount: number
  currency: string
  channel: string
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
