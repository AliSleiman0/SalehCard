import { apiClient } from '@/lib/api-client'
import type { ApiResponse, Order, PlacedOrder, PlaceOrderInput } from '@/types'

// placeOrder submits a checkout. The idempotency key dedupes retries so a
// double-submit returns the same order rather than charging/delivering twice.
// A usdt order returns pending with an attached paymentIntent (PlacedOrder).
export async function placeOrder(
  input: PlaceOrderInput,
  idempotencyKey: string,
): Promise<ApiResponse<PlacedOrder>> {
  return apiClient.post<PlacedOrder>('/api/v1/orders', input, {
    'Idempotency-Key': idempotencyKey,
  })
}

export async function fetchOrders(): Promise<ApiResponse<Order[]>> {
  return apiClient.get<Order[]>('/api/v1/orders')
}

export async function fetchOrder(id: string): Promise<ApiResponse<Order>> {
  return apiClient.get<Order>(`/api/v1/orders/${id}`)
}

/** Uploads a checkout photo-field image (QR code, barcode …; multipart
 * `image`) → its stored URL, submitted as that field's value on the order. */
export async function uploadOrderFieldFile(file: Blob): Promise<ApiResponse<{ url: string }>> {
  const form = new FormData()
  form.append('image', file, 'photo.jpg')
  return apiClient.postForm<{ url: string }>('/api/v1/orders/field-files', form)
}
