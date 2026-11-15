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
  /** USD per one unit of the supplier's currency (non-USD suppliers; the
   *  sync's loss check needs it). Absent = not set. */
  fxToUsd?: number
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

/** One supplier sync pass (the "Sync now" button or the scheduled job). */
export interface SyncReport {
  dryRun: boolean
  supplier?: number
  startedAt: string
  finishedAt: string
  suppliers: { id: number; name: string; status: 'ok' | 'stale' | 'unavailable' | 'not_supported'; error?: string }[]
  checked: number
  snapshots: number
  costChanges: number
  paused: number
  recovered: number
  promoted: number
  hidden: number
  shown: number
  conflicts: number
  alerts: number
  errors: number
}

/** The scheduled supplier sync's config + its last pass on this server. */
export interface SyncStatus {
  enabled: boolean
  dryRun: boolean
  intervalSeconds: number
  running: boolean
  last?: SyncReport
}

export type SupplierAlertKind =
  | 'cost_changed'
  | 'loss_paused'
  | 'loss_recovered'
  | 'product_hidden'
  | 'product_shown'
  | 'fx_missing'
  | 'fallback_exhausted'

export interface SupplierAlert {
  id: string
  kind: SupplierAlertKind
  key: string
  open: boolean
  message: string
  provider?: number
  productId?: string
  productName?: string
  linkId?: string
  orderId?: string
  dryRun?: boolean
  count: number
  firstAt: string
  lastAt: string
  ackAt?: string
  ackBy?: string
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
  fxToUsd?: number
}

export interface SupplierSettingsPatch {
  lowBalanceThreshold?: number | null
  markupPercent?: number | null
  fxToUsd?: number | null
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

export function syncSupplier(id: number): Promise<ApiResponse<SyncReport>> {
  return apiClient.post<SyncReport>(`${ADMIN}/${id}/sync`)
}

export function getSyncStatus(): Promise<ApiResponse<SyncStatus>> {
  return apiClient.get<SyncStatus>(`${ADMIN}/sync-status`)
}

export function listSupplierAlerts(
  status: 'open' | 'all',
): Promise<ApiResponse<{ alerts: SupplierAlert[]; open: number }>> {
  return apiClient.get<{ alerts: SupplierAlert[]; open: number }>(`${ADMIN}/alerts?status=${status}`)
}

export function ackSupplierAlert(id: string): Promise<ApiResponse<SupplierAlert>> {
  return apiClient.post<SupplierAlert>(`${ADMIN}/alerts/${id}/ack`)
}

export function ackAllSupplierAlerts(): Promise<ApiResponse<{ acknowledged: number }>> {
  return apiClient.post<{ acknowledged: number }>(`${ADMIN}/alerts/ack-all`)
}

export function importSupplierProducts(
  id: number,
  items: ImportItem[],
): Promise<ApiResponse<ImportResult>> {
  return apiClient.post<ImportResult>(`${ADMIN}/${id}/import`, { items })
}

export function updateSupplierSettings(
  id: number,
  patch: SupplierSettingsPatch,
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
