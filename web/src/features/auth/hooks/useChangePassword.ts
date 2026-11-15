import { useMutation } from '@tanstack/react-query'
import { changePassword } from '../api/auth'
import { ApiError, unwrap } from '@/lib/api-error'

// useChangePassword changes the signed-in user's own password. Failures are
// 400s carrying a code (see changePasswordErrorKey), so the session survives.
export function useChangePassword() {
  return useMutation<{ changed: boolean }, ApiError, { currentPassword: string; newPassword: string }>({
    mutationFn: async ({ currentPassword, newPassword }) =>
      unwrap(await changePassword(currentPassword, newPassword), 'Could not change password'),
  })
}
