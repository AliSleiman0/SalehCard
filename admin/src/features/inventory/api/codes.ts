import { apiClient } from '@/lib/api-client'
import type { ApiResponse, PaginationMeta, Code, CodeStatus, InventoryStats, CodeUploadResult } from '@/types'

export interface CodeAudit {
  code: Code
  product?: { id: string; title: string }
}

/** One persisted bulk-upload record, for the inventory upload-history panel. */
export interface UploadBatchView {
  id: string
  productId: string
  productTitle: string
  batch: string
  inserted: number
  duplicates: number
  invalid: number
  uploadedBy?: string
  createdAt: string
}

/** Global KPI totals returned alongside a paginated inventory page. */
export interface InventoryTotals {
  uploaded: number
  available: number
  delivered: number
  lowStock: number
  /** Σ (available × unit cost) across priced products; unvaluedProducts counts those
   *  with no unit price set (excluded from the sum, not treated as $0). */
  totalValue: number
  unvaluedProducts: number
}

/** Meta for the paginated inventory listing: pagination + global KPI totals. */
export type InventoryMeta = PaginationMeta & { totals: InventoryTotals }

/**
 * Per-product code stats. Called without params it returns the full list (used
 * by the thresholds editor and the upload product picker). With `page` it hits
 * the backend-paginated endpoint, returning one page plus `meta` (pagination +
 * global totals); `low` restricts the page to low-stock products.
 */
export function listInventory(
  params: { page?: number; limit?: number; low?: boolean; q?: string } = {}
): Promise<ApiResponse<InventoryStats[]> & { meta?: InventoryMeta }> {
  const q = new URLSearchParams()
  if (params.page !== undefined) q.set('page', String(params.page))
  if (params.limit !== undefined) q.set('limit', String(params.limit))
  if (params.low) q.set('low', 'true')
  if (params.q) q.set('q', params.q)
  const qs = q.toString()
  return apiClient.get<InventoryStats[]>(`/api/admin/inventory${qs ? `?${qs}` : ''}`) as Promise<
    ApiResponse<InventoryStats[]> & { meta?: InventoryMeta }
  >
}

/** Recent upload batches (newest first). */
export function listUploadHistory(limit = 20): Promise<ApiResponse<UploadBatchView[]>> {
  return apiClient.get<UploadBatchView[]>(`/api/admin/upload-history?limit=${limit}`)
}

export function listCodes(
  productId: string,
  params: { status?: CodeStatus; page?: number; limit?: number } = {}
): Promise<ApiResponse<Code[]> & { meta?: PaginationMeta }> {
  const q = new URLSearchParams()
  if (params.status) q.set('status', params.status)
  if (params.page !== undefined) q.set('page', String(params.page))
  if (params.limit !== undefined) q.set('limit', String(params.limit))
  const qs = q.toString()
  return apiClient.get<Code[]>(`/api/admin/products/${productId}/codes${qs ? `?${qs}` : ''}`)
}

export interface UploadItem {
  code: string
  pin?: string
}

export function uploadCodes(productId: string, codes: UploadItem[]): Promise<ApiResponse<CodeUploadResult>> {
  return apiClient.post<CodeUploadResult>(`/api/admin/products/${productId}/codes`, { codes })
}

/** Add a single code to a product's pool (inventory codes popup). */
export function addCode(productId: string, body: { code: string; pin?: string }): Promise<ApiResponse<Code>> {
  return apiClient.post<Code>(`/api/admin/products/${productId}/codes/single`, body)
}

/** Edit an available code's value/pin (keyed by the code's ObjectID). */
export function editCode(
  productId: string,
  codeId: string,
  body: { code: string; pin?: string }
): Promise<ApiResponse<Code>> {
  return apiClient.put<Code>(`/api/admin/products/${productId}/codes/${codeId}`, body)
}

/** Delete a non-delivered code (keyed by the code's ObjectID). */
export function deleteCode(productId: string, codeId: string): Promise<ApiResponse<Code>> {
  return apiClient.delete<Code>(`/api/admin/products/${productId}/codes/${codeId}`)
}

/** Retire an available code from the pool via the existing expire endpoint. */
export function expireCode(code: string): Promise<ApiResponse<Code>> {
  return apiClient.put<Code>(`/api/admin/codes/${encodeURIComponent(code)}/expire`)
}

export function lookupCode(code: string): Promise<ApiResponse<CodeAudit>> {
  return apiClient.get<CodeAudit>(`/api/admin/codes/${encodeURIComponent(code)}`)
}

export function setStockThreshold(productId: string, threshold: number): Promise<ApiResponse<InventoryStats>> {
  return apiClient.put<InventoryStats>(`/api/admin/products/${productId}/stock-threshold`, { threshold })
}

/** Parse a pasted/loaded codes file: one code per line, or `code,pin` per line.
 *  Strips ALL whitespace (internal too) from each field so the preview count
 *  matches what the server stores — the server-side strip is the authoritative fix. */
export function parseCodesFile(text: string): { items: UploadItem[]; invalid: number } {
  const items: UploadItem[] = []
  let invalid = 0
  for (const raw of text.split(/\r?\n/)) {
    if (!raw.trim()) continue
    const [code, pin] = raw.split(',').map((s) => s.replace(/\s+/g, ''))
    if (!code) {
      invalid++
      continue
    }
    items.push(pin ? { code, pin } : { code })
  }
  return { items, invalid }
}
