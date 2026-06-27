import { apiClient } from '@/lib/api-client'
import type { ApiResponse, PaginationMeta, I18nString, FulfillmentType } from '@/types'

/** Order status as stored by the API. */
export type OrderStatus = 'pending' | 'processing' | 'completed' | 'failed' | 'refunded'

/** Payment method as stored by the API. */
export type PaymentMethod = 'wallet' | 'card' | 'usdt'

export interface AdminOrderItem {
  productId: string
  variantId: string
  title: I18nString
  denomination?: string
  category: string
  qty: number
  price: number
  fulfillmentType: FulfillmentType
  playerId?: string
  recipient?: { name: string; country: string; detail: string }
}

export interface OrderFulfillment {
  deliveredCode?: string
  creditedToId?: string
  transferRef?: string
  statusTimeline: { status: string; note: string; at: string }[]
}

export interface OrderCustomer {
  id: string
  email: string
  phone?: string
}

/** An order enriched with its customer, as returned by the admin endpoints. */
export interface AdminOrder {
  id: string
  userId: string
  items: AdminOrderItem[]
  subtotal: number
  total: number
  currency: string
  paymentMethod: PaymentMethod
  status: OrderStatus
  fulfillment: OrderFulfillment
  createdAt: string
  updatedAt: string
  customer: OrderCustomer | null
}

export interface OrderListParams {
  page?: number
  limit?: number
  status?: OrderStatus | ''
  fulfillmentType?: FulfillmentType | ''
  paymentMethod?: PaymentMethod | ''
  q?: string
}

const ADMIN = '/api/admin/orders'

export function listOrders(params: OrderListParams): Promise<ApiResponse<AdminOrder[]> & { meta?: PaginationMeta }> {
  const q = new URLSearchParams()
  if (params.page !== undefined) q.set('page', String(params.page))
  if (params.limit !== undefined) q.set('limit', String(params.limit))
  if (params.status) q.set('status', params.status)
  if (params.fulfillmentType) q.set('fulfillmentType', params.fulfillmentType)
  if (params.paymentMethod) q.set('paymentMethod', params.paymentMethod)
  if (params.q) q.set('q', params.q)
  const qs = q.toString()
  return apiClient.get<AdminOrder[]>(qs ? `${ADMIN}?${qs}` : ADMIN)
}

export function getOrder(id: string): Promise<ApiResponse<AdminOrder>> {
  return apiClient.get<AdminOrder>(`${ADMIN}/${id}`)
}
