import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  listUsers,
  getUser,
  updateUserRole,
  updateUserStatus,
  adjustWallet,
  bulkUserAction,
  deleteUser,
  bulkSmsUsers,
  type UserListParams,
  type UserStatus,
  type BulkUserAction,
  type WalletAdjustInput,
} from '../api/users'
import type { UserRole } from '@/types'

export function useUsers(params: UserListParams = {}, opts?: { enabled?: boolean }) {
  return useQuery({
    queryKey: ['admin', 'users', params],
    queryFn: () => listUsers(params),
    // Keep the current page visible while the next page/filter/search loads.
    placeholderData: keepPreviousData,
    // Cross-domain callers (Settings' admin-accounts card) pass enabled:false
    // when the admin lacks users.view, so the fetch isn't fired to 403.
    enabled: opts?.enabled ?? true,
  })
}

export function useUser(id: string | undefined) {
  return useQuery({
    queryKey: ['admin', 'user', id],
    queryFn: () => getUser(id!),
    enabled: !!id,
  })
}

export function useUpdateUserRole(id: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ role, adminRoleId }: { role: UserRole; adminRoleId?: string | null }) =>
      updateUserRole(id, role, adminRoleId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['admin', 'users'] })
      qc.invalidateQueries({ queryKey: ['admin', 'user', id] })
    },
  })
}

export function useUpdateUserStatus(id: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (status: UserStatus) => updateUserStatus(id, status),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['admin', 'users'] })
      qc.invalidateQueries({ queryKey: ['admin', 'user', id] })
    },
  })
}

export function useAdjustWallet(id: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: WalletAdjustInput) => adjustWallet(id, input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['admin', 'users'] })
      qc.invalidateQueries({ queryKey: ['admin', 'user', id] })
    },
  })
}

export function useBulkUserAction() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ ids, action }: { ids: string[]; action: BulkUserAction }) => bulkUserAction(ids, action),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin', 'users'] }),
  })
}

export function useBulkSms() {
  return useMutation({
    mutationFn: ({ ids, message }: { ids: string[]; message: string }) => bulkSmsUsers(ids, message),
  })
}

export function useDeleteUser(id: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => deleteUser(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['admin', 'users'] })
      qc.invalidateQueries({ queryKey: ['admin', 'user', id] })
    },
  })
}
