import { apiClient } from '@/lib/api-client'
import type { ApiResponse } from '@/types'

/** Map of product id -> total units sold across completed orders. Products with
 *  no completed sales are absent (treat as 0). Powers the product list "Sold"
 *  column; one fetch covers every product regardless of the list's pagination. */
export function listSoldByProduct(): Promise<ApiResponse<Record<string, number>>> {
  return apiClient.get<Record<string, number>>('/api/admin/orders/sold-by-product')
}
