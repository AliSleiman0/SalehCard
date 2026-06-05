import { apiClient } from '@/lib/api-client'
import type { ApiResponse, WalletTransaction, WalletView } from '@/types'

export interface TopUpInput {
  amount: number
  method: string
  ref?: string
}

export async function fetchWallet(): Promise<ApiResponse<WalletView>> {
  return apiClient.get<WalletView>('/api/v1/wallet')
}

export async function topUp(input: TopUpInput): Promise<ApiResponse<WalletTransaction>> {
  return apiClient.post<WalletTransaction>('/api/v1/wallet/topups', input)
}
