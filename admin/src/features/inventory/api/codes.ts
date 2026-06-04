import { apiClient } from '@/lib/api-client'
import type { ApiResponse, PaginationMeta, Code, CodeStatus, InventoryStats, CodeUploadResult } from '@/types'

export interface CodeAudit {
  code: Code
  product?: { id: string; title: string }
}

/** Per-product code stats across all code-type products. */
export function listInventory(): Promise<ApiResponse<InventoryStats[]>> {
  return apiClient.get<InventoryStats[]>('/api/admin/inventory')
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

export function lookupCode(code: string): Promise<ApiResponse<CodeAudit>> {
  return apiClient.get<CodeAudit>(`/api/admin/codes/${encodeURIComponent(code)}`)
}

export function setStockThreshold(productId: string, threshold: number): Promise<ApiResponse<InventoryStats>> {
  return apiClient.put<InventoryStats>(`/api/admin/products/${productId}/stock-threshold`, { threshold })
}

/** Parse a pasted/loaded codes file: one code per line, or `code,pin` per line. */
export function parseCodesFile(text: string): { items: UploadItem[]; invalid: number } {
  const items: UploadItem[] = []
  let invalid = 0
  for (const raw of text.split(/\r?\n/)) {
    const line = raw.trim()
    if (!line) continue
    const [code, pin] = line.split(',').map((s) => s.trim())
    if (!code) {
      invalid++
      continue
    }
    items.push(pin ? { code, pin } : { code })
  }
  return { items, invalid }
}
