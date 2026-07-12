import { useMutation, useQueryClient } from '@tanstack/react-query'
import { deleteAccount } from '../api/auth'
import { ApiError, unwrap } from '@/lib/api-error'
import { useAuthStore } from '@/stores/auth'

// useDeleteAccount deletes the account, then clears every cached query and the
// auth session. Does NOT call /auth/logout — the server already revoked all
// sessions and the account is gone.
export function useDeleteAccount() {
  const qc = useQueryClient()
  return useMutation<{ deleted: boolean }, ApiError, void>({
    mutationFn: async () => unwrap(await deleteAccount(), 'Could not delete account'),
    onSuccess: () => {
      qc.clear()
      useAuthStore.getState().logout()
    },
  })
}
