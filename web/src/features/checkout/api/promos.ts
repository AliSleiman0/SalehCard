import { apiClient } from '@/lib/api-client'
import type { ApiResponse } from '@/types'

export interface PromoResult {
  valid: boolean
  code: string
  type: 'percent' | 'fixed'
  value: number
  discount: number
}

// validatePromo previews a code's discount for the given order total. The order
// endpoint re-applies it authoritatively at submit.
export async function validatePromo(input: {
  code: string
  orderTotal: number
}): Promise<ApiResponse<PromoResult>> {
  return apiClient.post<PromoResult>('/api/v1/promos/validate', input)
}
