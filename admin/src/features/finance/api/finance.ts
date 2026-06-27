import { apiClient } from '@/lib/api-client'
import type { ApiResponse, PaginationMeta } from '@/types'

/** Ledger transaction type as stored by the API. */
export type TxType = 'topup' | 'purchase' | 'refund' | 'adjustment'

/** A wallet-ledger row enriched with its owning user, as returned by the admin
 *  transactions feed. */
export interface AdminTx {
  id: string
  userId: string
  type: TxType
  amount: number
  balanceAfter: number
  method: string
  ref: string
  createdAt: string
  user: { name: string; role: string }
}

export interface TxListParams {
  page?: number
  limit?: number
  type?: TxType | ''
  method?: string
  q?: string
}

/** One labeled bucket of a revenue breakdown (payment method / currency / category). */
export interface LabelValue {
  label: string
  value: number
}

/** The revenue-summary payload: KPIs, the time series for the requested range,
 *  and the by-method / by-category / by-currency breakdowns. */
export interface RevenueSummary {
  totalRevenue: number
  monthRevenue: number
  refundRate: number
  walletTopups: number
  range: string
  labels: string[]
  series: number[]
  byMethod: LabelValue[]
  byCategory: LabelValue[]
  byCurrency: LabelValue[]
}

const TX = '/api/admin/transactions'
const REV = '/api/admin/revenue-summary'

export function listTransactions(
  params: TxListParams,
): Promise<ApiResponse<AdminTx[]> & { meta?: PaginationMeta }> {
  const q = new URLSearchParams()
  if (params.page !== undefined) q.set('page', String(params.page))
  if (params.limit !== undefined) q.set('limit', String(params.limit))
  if (params.type) q.set('type', params.type)
  if (params.method) q.set('method', params.method)
  if (params.q) q.set('q', params.q)
  const qs = q.toString()
  return apiClient.get<AdminTx[]>(qs ? `${TX}?${qs}` : TX)
}

export function getRevenueSummary(range: string): Promise<ApiResponse<RevenueSummary>> {
  const qs = range ? `?range=${encodeURIComponent(range)}` : ''
  return apiClient.get<RevenueSummary>(`${REV}${qs}`)
}
