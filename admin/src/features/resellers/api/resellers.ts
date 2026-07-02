import { apiClient } from '@/lib/api-client'
import type { ApiResponse, PaginationMeta } from '@/types'
import type { WalletTx } from '@/features/users/api/users'

/** Account status as stored by the API (resellers are users with role=reseller). */
export type ResellerStatus = 'active' | 'suspended'

/** A reseller tier definition plus its current reseller headcount. */
export interface TierDef {
  id: string
  name: string
  marginPercent: number
  count: number
}

/** A reseller enriched with their tier margin + order aggregate (admin list). */
export interface AdminReseller {
  id: string
  email: string
  phone?: string
  status: ResellerStatus
  resellerTier?: string
  margin: number
  walletBalance: number
  loyaltyPoints: number
  savedPlayerIds: string[]
  createdAt: string
  updatedAt: string
  orders: number
  volume: number
}

/** The reseller detail adds the recent wallet ledger rows. */
export interface AdminResellerDetail extends AdminReseller {
  transactions: WalletTx[]
}

export interface ResellerListParams {
  page?: number
  limit?: number
  tier?: string
  status?: ResellerStatus | ''
  q?: string
}

export interface BalanceAdjustInput {
  direction: 'credit' | 'debit'
  amount: number
  reason: string
}

export interface TierInput {
  name: string
  marginPercent: number
}

const RESELLERS = '/api/admin/resellers'
const TIERS = '/api/admin/reseller-tiers'

export function listResellers(
  params: ResellerListParams,
): Promise<ApiResponse<AdminReseller[]> & { meta?: PaginationMeta }> {
  const q = new URLSearchParams()
  if (params.page !== undefined) q.set('page', String(params.page))
  if (params.limit !== undefined) q.set('limit', String(params.limit))
  if (params.tier) q.set('tier', params.tier)
  if (params.status) q.set('status', params.status)
  if (params.q) q.set('q', params.q)
  const qs = q.toString()
  return apiClient.get<AdminReseller[]>(qs ? `${RESELLERS}?${qs}` : RESELLERS)
}

export function getReseller(id: string): Promise<ApiResponse<AdminResellerDetail>> {
  return apiClient.get<AdminResellerDetail>(`${RESELLERS}/${id}`)
}

export function updateResellerTier(id: string, tier: string): Promise<ApiResponse<AdminReseller>> {
  return apiClient.put<AdminReseller>(`${RESELLERS}/${id}/tier`, { tier })
}

export function adjustResellerBalance(
  id: string,
  input: BalanceAdjustInput,
): Promise<ApiResponse<{ walletBalance: number }>> {
  return apiClient.post<{ walletBalance: number }>(`${RESELLERS}/${id}/balance-adjust`, input)
}

export function listTiers(): Promise<ApiResponse<TierDef[]>> {
  return apiClient.get<TierDef[]>(TIERS)
}

export function createTier(input: TierInput): Promise<ApiResponse<TierDef>> {
  return apiClient.post<TierDef>(TIERS, input)
}

export function updateTier(id: string, input: TierInput): Promise<ApiResponse<TierDef>> {
  return apiClient.put<TierDef>(`${TIERS}/${id}`, input)
}

export function deleteTier(id: string): Promise<ApiResponse<{ deleted: boolean }>> {
  return apiClient.delete<{ deleted: boolean }>(`${TIERS}/${id}`)
}
