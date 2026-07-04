import { apiClient } from '@/lib/api-client'
import type { ApiResponse, PaginationMeta, SavedPlayerId, UserRole } from '@/types'

/** Account status as stored by the API. `deleted` is a soft-deleted (anonymized)
 *  account — hidden from the default listing. */
export type UserStatus = 'active' | 'suspended' | 'deleted'

/** Bulk status action applied to many users at once. */
export type BulkUserAction = 'suspend' | 'activate'

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
  savedPlayerIds: SavedPlayerId[]
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

/** Suspend or activate many users in one call. Admin accounts are skipped for
 *  suspend server-side (the last-admin guard). Returns the modified count. */
export function bulkUserAction(ids: string[], action: BulkUserAction): Promise<ApiResponse<{ modified: number }>> {
  return apiClient.post<{ modified: number }>(`${ADMIN}/bulk`, { ids, action })
}

/** Soft-delete (anonymize) an account. The row is kept so its orders/ledger
 *  still resolve; the person can no longer sign in. */
export function deleteUser(id: string): Promise<ApiResponse<AdminUser>> {
  return apiClient.delete<AdminUser>(`${ADMIN}/${id}`)
}

/** Send a single-segment SMS to many users at once. Users without a phone number
 *  are skipped server-side; the result reports how many were queued. The backend
 *  hard-caps the recipient count (BULK_SMS_LIMIT) — these are real, paid messages. */
export function bulkSmsUsers(ids: string[], message: string): Promise<ApiResponse<{ queued: number }>> {
  return apiClient.post<{ queued: number }>(`${ADMIN}/bulk-sms`, { ids, message })
}
