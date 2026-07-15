import { apiClient } from '@/lib/api-client'
import type { ApiResponse, PaginationMeta } from '@/types'

/** Health of a supplier's upstream connection (probed with a short TTL cache). */
export type SupplierHealth =
  | 'ok'
  | 'low_balance'
  | 'auth_error'
  | 'ip_blocked'
  | 'unreachable'
  | 'maintenance'
  | 'not_probed'

/** A configured upstream supplier (panel / telecom) with its live balance + health. */
export interface SupplierView {
  id: number
  name: string
  kind: string
  currency: string
  baseUrl: string
  health: SupplierHealth
  balance: number | null
  balanceText: string
  mappedProducts: number
  lowBalanceThreshold: number
  markupPercent: number
}

/** A product line in a supplier's live upstream catalog. */
export interface CatalogProductView {
  upstreamId: string
  name: string
  category: string
  parentId: string
  price: number
  basePrice: number
  currency: string
  available: boolean
  productType: string
  params: string[]
  qtyMin: number | null
  qtyMax: number | null
  qtyValues: string[]
  mapped: boolean
}

/** Result of a catalog sync for already-mapped products. */
export interface SyncResult {
  checked: number
  updated: number
  unavailable: number
  drift: { upstreamId: string; name: string; oldPrice: number; newPrice: number }[]
}

/** One item to import into the SalehCard catalog from the live upstream entry. */
export interface ImportItem {
  upstreamId: string
  categoryId?: string | null
  markupPercent?: number | null
}

export interface ImportResult {
  created: number
  skipped: { upstreamId: string; reason: string }[]
}

export interface SupplierSettings {
  lowBalanceThreshold: number
  markupPercent: number
}

/** A recent api-mode order fulfilled through this supplier. */
export interface SupplierOrder {
  id: string
  status: string
  total: number
  currency: string
  createdAt: string
  upstreamRef: string
  deliveredCode: string
  playerId: string
  stuckAt: string | null
}

export interface SupplierOrdersParams {
  limit?: number
  offset?: number
}

const ADMIN = '/api/admin/suppliers'

export function listSuppliers(): Promise<ApiResponse<{ suppliers: SupplierView[] }>> {
  return apiClient.get<{ suppliers: SupplierView[] }>(ADMIN)
}

export function getSupplierCatalog(id: number): Promise<ApiResponse<{ products: CatalogProductView[] }>> {
  return apiClient.get<{ products: CatalogProductView[] }>(`${ADMIN}/${id}/catalog`)
}

export function syncSupplier(id: number): Promise<ApiResponse<SyncResult>> {
  return apiClient.post<SyncResult>(`${ADMIN}/${id}/sync`)
}

export function importSupplierProducts(
  id: number,
  items: ImportItem[],
): Promise<ApiResponse<ImportResult>> {
  return apiClient.post<ImportResult>(`${ADMIN}/${id}/import`, { items })
}

export function updateSupplierSettings(
  id: number,
  patch: { lowBalanceThreshold?: number | null; markupPercent?: number | null },
): Promise<ApiResponse<SupplierSettings>> {
  return apiClient.put<SupplierSettings>(`${ADMIN}/${id}/settings`, patch)
}

export function getSupplierOrders(
  id: number,
  params: SupplierOrdersParams,
): Promise<ApiResponse<{ orders: SupplierOrder[] }> & { meta?: PaginationMeta }> {
  const q = new URLSearchParams()
  if (params.limit !== undefined) q.set('limit', String(params.limit))
  if (params.offset !== undefined) q.set('offset', String(params.offset))
  const qs = q.toString()
  return apiClient.get<{ orders: SupplierOrder[] }>(qs ? `${ADMIN}/${id}/orders?${qs}` : `${ADMIN}/${id}/orders`)
}
