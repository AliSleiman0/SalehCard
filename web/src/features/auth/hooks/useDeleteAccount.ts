import { useMutation, useQueryClient } from '@tanstack/react-query'
import { deleteAccount } from '../api/auth'
import { ApiError, unwrap } from '@/lib/api-error'
import { useAuthStore } from '@/stores/auth'

// useDeleteAccount deletes the account, then clears every cached query and the
// auth session. Does NOT call /auth/logout — the server already revoked all
// sessions and the account is gone. Pass {force: true} for the explicit
// "Delete anyway" path (see canForceDelete).
export function useDeleteAccount() {
  const qc = useQueryClient()
  return useMutation<{ deleted: boolean }, ApiError, { force?: boolean } | void>({
    mutationFn: async (vars) =>
      unwrap(await deleteAccount(!!vars?.force), 'Could not delete account'),
    onSuccess: () => {
      qc.clear()
      useAuthStore.getState().logout()
    },
  })
}
