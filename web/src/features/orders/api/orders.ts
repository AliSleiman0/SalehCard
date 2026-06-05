import { apiClient } from '@/lib/api-client'
import type { ApiResponse, Order, PlaceOrderInput } from '@/types'

// placeOrder submits a checkout. The idempotency key dedupes retries so a
// double-submit returns the same order rather than charging/delivering twice.
export async function placeOrder(
  input: PlaceOrderInput,
  idempotencyKey: string,
): Promise<ApiResponse<Order>> {
  return apiClient.post<Order>('/api/v1/orders', input, {
    'Idempotency-Key': idempotencyKey,
  })
}

export async function fetchOrders(): Promise<ApiResponse<Order[]>> {
  return apiClient.get<Order[]>('/api/v1/orders')
}

export async function fetchOrder(id: string): Promise<ApiResponse<Order>> {
  return apiClient.get<Order>(`/api/v1/orders/${id}`)
}
