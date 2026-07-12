import { apiClient } from '@/lib/api-client'
import type { ApiResponse } from '@/types'

export interface VerifyResult {
  found: boolean
  username?: string
  banned?: boolean
  // 'id_not_found' → block the purchase; 'unavailable' → fail open (allow).
  reason?: string
}

export async function verifyAccount(
  productId: string,
  playerId: string,
): Promise<ApiResponse<VerifyResult>> {
  return apiClient.post<VerifyResult>(`/api/v1/products/${productId}/verify-account`, { playerId })
}
