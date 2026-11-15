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
  /** Per-unit line: the amount the customer typed (e.g. 45000). qty is then 1
   *  and price is the whole line price; denomination reads "45,000". */
  amount?: number
  price: number
  fulfillmentType: FulfillmentType
  /** Supplier routing snapshot ("api" lines are fulfilled by a supplier). */
  fulfillmentMode?: string
  fulfillmentProvider?: number
  upstreamProductId?: string
  playerId?: string
  recipient?: { name: string; country: string; detail: string }
  /** Labeled customer inputs captured at checkout (Account ID, Zone ID, Email…). */
  fields?: { key: string; label: I18nString; value: string }[]
  /** Delivered codes for an inventory/code line, one per unit (omitted when none). */
  codes?: string[]
}

/** One abandoned supplier attempt (kept for reconciling a late delivery). */
export interface SupplierAttempt {
  provider?: number
  upstreamProductId?: string
  providerRef?: string
  orderUuid?: string
  stuckAt?: string
  reroutedAt: string
  reroutedBy?: string
}

export interface OrderFulfillment {
  deliveredCode?: string
  creditedToId?: string
  transferRef?: string
  /** Supplier order id of the current attempt (api-mode orders). */
  providerRef?: string
  /** Set once the supplier settler gave up waiting — eligible for a reroute. */
  supplierStuckAt?: string
  supplierAttempt?: number
  supplierAttemptAt?: string
  /** Abandoned attempts, archived on each "send to another supplier". */
  supplierAttempts?: SupplierAttempt[]
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
  /** Line product ids whose product has since been deleted (omitted when none). */
  missingProductIds?: string[]
}

export interface OrderListParams {
  page?: number
  limit?: number
  status?: OrderStatus | ''
  fulfillmentType?: FulfillmentType | ''
  paymentMethod?: PaymentMethod | ''
  q?: string
  /** "today" = the current business-timezone day (server-computed window). */
  date?: 'today' | ''
  /** Only processing orders the supplier settler flagged stuck (overrides status). */
  supplierStuck?: boolean
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
  if (params.date) q.set('date', params.date)
  if (params.supplierStuck) q.set('supplierStuck', 'true')
  const qs = q.toString()
  return apiClient.get<AdminOrder[]>(qs ? `${ADMIN}?${qs}` : ADMIN)
}

export function getOrder(id: string): Promise<ApiResponse<AdminOrder>> {
  return apiClient.get<AdminOrder>(`${ADMIN}/${id}`)
}

/** Refund a processing or completed order (credits a wallet charge back). An
 *  order a supplier may still deliver answers 409 ORDER_AT_SUPPLIER unless
 *  acknowledgeSupplier is set (double-delivery guard). */
export function refundOrder(id: string, reason = '', acknowledgeSupplier = false): Promise<ApiResponse<AdminOrder>> {
  return apiClient.post<AdminOrder>(`${ADMIN}/${id}/refund`, {
    reason,
    ...(acknowledgeSupplier ? { acknowledgeSupplier: true } : {}),
  })
}

/** Per-order outcome of a bulk refund. */
export interface BulkRefundOutcome {
  id: string
  /** at_supplier = skipped: a supplier may still deliver it (refund it from its page to confirm). */
  status: 'refunded' | 'conflict' | 'at_supplier' | 'error'
  message?: string
}

export interface BulkRefundResult {
  refunded: number
  total: number
  results: BulkRefundOutcome[]
}

/** Refund many orders at once. Partial failure is isolated: the result reports a
 *  per-order status (refunded / conflict / error). */
export function refundBulkOrders(ids: string[], reason = ''): Promise<ApiResponse<BulkRefundResult>> {
  return apiClient.post<BulkRefundResult>(`${ADMIN}/refund-bulk`, { ids, reason })
}

/**
 * Mark a stuck pending order as failed (admin cleanup). Moves no money — if the
 * wallet was charged, the admin reverses it via the customer's wallet adjustment.
 */
export function failOrder(id: string, reason = ''): Promise<ApiResponse<AdminOrder>> {
  return apiClient.post<AdminOrder>(`${ADMIN}/${id}/fail`, { reason })
}

/** Manually complete a processing order (account credit / transfer / parked). */
/** Completes a processing order by hand. A supplier may still be delivering an
 *  order it accepted (or will retry): the API then answers 409
 *  ORDER_AT_SUPPLIER unless acknowledgeSupplier is set (double-delivery guard). */
export function completeOrder(
  id: string,
  opts: { note?: string; transferRef?: string; acknowledgeSupplier?: boolean } = {},
): Promise<ApiResponse<AdminOrder>> {
  return apiClient.put<AdminOrder>(`${ADMIN}/${id}/status`, {
    status: 'completed',
    note: opts.note ?? '',
    transferRef: opts.transferRef ?? '',
    ...(opts.acknowledgeSupplier ? { acknowledgeSupplier: true } : {}),
  })
}

/** A supplier a stuck order can be sent to. */
export interface RerouteProvider {
  id: number
  name: string
}

export function listRerouteProviders(): Promise<ApiResponse<RerouteProvider[]>> {
  return apiClient.get<RerouteProvider[]>('/api/admin/orders/reroute-providers')
}

export interface RerouteInput {
  /** One of the product's supplier links (preferred — nothing is typed). */
  linkId?: string
  /** A raw supplier + supplier product ("other supplier product…"). */
  providerId?: number
  upstreamProductId?: string
  /** Re-send although the current supplier may still deliver (double-delivery risk). */
  acknowledgeAtSupplier?: boolean
  acknowledgeAmbiguous?: boolean
  /** Send even though the supplier's balance looks below this order's cost. */
  acknowledgeLowBalance?: boolean
}

/** Sends a processing order to a supplier (another one, or the same one again)
 *  and dispatches it there now. */
export function rerouteOrder(id: string, input: RerouteInput): Promise<ApiResponse<AdminOrder>> {
  return apiClient.post<AdminOrder>(`/api/admin/orders/${id}/reroute`, input)
}

/** One of the order's product supplier links, priced for this order. */
export interface SupplierOption {
  linkId: string
  provider: number
  providerName?: string
  upstreamId: string
  name?: string
  enabled: boolean
  active: boolean
  /** The order's current attempt went here. */
  current: boolean
  /** An earlier attempt of this order went here. */
  tried: boolean
  catalogStatus: 'ok' | 'unknown' | 'unavailable' | 'not_checked'
  available: boolean
  stale?: boolean
  unitPrice?: number
  currency?: string
  /** Supplier cost of this order (unit price × the units it sends). */
  cost?: number
  qtyOk: boolean
  lowBalance?: boolean
  /** Only for admins who can see supplier accounts. */
  balance?: number
  balanceCurrency?: string
  /** Why it can't take the order ("" / absent = it can). */
  blocked?: string
}

export interface SupplierOptions {
  canSend: boolean
  reason?: string
  message?: string
  fromManual: boolean
  /** The current supplier may still deliver: re-sending needs acknowledgeAtSupplier. */
  needsAck: boolean
  /** The last outcome is unknown: needs acknowledgeAmbiguous. */
  ambiguous: boolean
  qty: number
  /** What the customer paid for the line (the margin base). */
  linePrice: number
  currency: string
  productMissing?: boolean
  options: SupplierOption[]
}

export function getSupplierOptions(id: string): Promise<ApiResponse<SupplierOptions>> {
  return apiClient.get<SupplierOptions>(`/api/admin/orders/${id}/supplier-options`)
}

/** At most this many orders per bulk re-send (each one is a billed supplier order). */
export const MAX_BULK_RESEND = 5

export interface BulkRerouteInput {
  ids: string[]
  /** Send each order to its product's link at this supplier (else its first usable link). */
  providerId?: number
  acknowledgeAtSupplier?: boolean
  acknowledgeAmbiguous?: boolean
  acknowledgeLowBalance?: boolean
}

export interface BulkRerouteOutcome {
  id: string
  status: 'completed' | 'sent' | 'refused' | 'needs_ack' | 'no_link' | 'not_eligible' | 'conflict' | 'error'
  message?: string
  provider?: number
  upstreamId?: string
}

export interface BulkRerouteResult {
  sent: number
  total: number
  results: BulkRerouteOutcome[]
}

export function rerouteBulkOrders(input: BulkRerouteInput): Promise<ApiResponse<BulkRerouteResult>> {
  return apiClient.post<BulkRerouteResult>('/api/admin/orders/reroute-bulk', input)
}
