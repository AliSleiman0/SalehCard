import { apiClient } from '@/lib/api-client'
import type { ApiResponse, PaginationMeta, UserRole } from '@/types'

/** Account status as stored by the API. */
export type UserStatus = 'active' | 'suspended'

/** A wallet ledger row, as returned in the user detail payload. */
export interface WalletTx {
  id: string
  userId: string
  type: string
  amount: number
  balanceAfter: number
  method: string
  ref: string
  createdAt: string
}

/** A user enriched with their order aggregate, as returned by the admin list. */
export interface AdminUser {
  id: string
  email: string
  phone?: string
  role: UserRole
  status: UserStatus
  locale: string
  savedPlayerIds: string[]
  walletBalance: number
  loyaltyPoints: number
  createdAt: string
  updatedAt: string
  lastSeen?: string
  orders: number
  spent: number
}

/** The user detail adds the recent wallet ledger rows. */
export interface AdminUserDetail extends AdminUser {
  transactions: WalletTx[]
}

export interface UserListParams {
  page?: number
  limit?: number
  role?: UserRole | ''
  status?: UserStatus | ''
  q?: string
}

export interface WalletAdjustInput {
  direction: 'credit' | 'debit'
  amount: number
  reason: string
}

const ADMIN = '/api/admin/users'

export function listUsers(params: UserListParams): Promise<ApiResponse<AdminUser[]> & { meta?: PaginationMeta }> {
  const q = new URLSearchParams()
  if (params.page !== undefined) q.set('page', String(params.page))
  if (params.limit !== undefined) q.set('limit', String(params.limit))
  if (params.role) q.set('role', params.role)
  if (params.status) q.set('status', params.status)
  if (params.q) q.set('q', params.q)
  const qs = q.toString()
  return apiClient.get<AdminUser[]>(qs ? `${ADMIN}?${qs}` : ADMIN)
}

export function getUser(id: string): Promise<ApiResponse<AdminUserDetail>> {
  return apiClient.get<AdminUserDetail>(`${ADMIN}/${id}`)
}

export function updateUserRole(id: string, role: UserRole): Promise<ApiResponse<AdminUser>> {
  return apiClient.put<AdminUser>(`${ADMIN}/${id}/role`, { role })
}

export function updateUserStatus(id: string, status: UserStatus): Promise<ApiResponse<AdminUser>> {
  return apiClient.put<AdminUser>(`${ADMIN}/${id}/status`, { status })
}

export function adjustWallet(id: string, input: WalletAdjustInput): Promise<ApiResponse<{ walletBalance: number }>> {
  return apiClient.post<{ walletBalance: number }>(`${ADMIN}/${id}/wallet-adjust`, input)
}
