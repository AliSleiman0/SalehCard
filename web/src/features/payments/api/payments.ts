import { apiClient } from '@/lib/api-client'
import type { ApiResponse, PaymentConfig, PaymentIntent } from '@/types'

export async function fetchPaymentConfig(): Promise<ApiResponse<PaymentConfig>> {
  return apiClient.get<PaymentConfig>('/api/v1/payments/config')
}

export async function createTopUpIntent(
  input: { amount: number; network?: string },
  idempotencyKey: string,
): Promise<ApiResponse<PaymentIntent>> {
  return apiClient.post<PaymentIntent>('/api/v1/payments/usdt/topup-intents', input, {
    'Idempotency-Key': idempotencyKey,
  })
}

export async function fetchPaymentIntent(id: string): Promise<ApiResponse<PaymentIntent>> {
  return apiClient.get<PaymentIntent>(`/api/v1/payments/intents/${id}`)
}
