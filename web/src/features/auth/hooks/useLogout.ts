import { useMutation } from '@tanstack/react-query'
import { logout } from '../api/auth'
import { useAuthStore } from '@/stores/auth'

export function useLogout() {
  return useMutation<void, Error, void>({
    // Always clear local session, even if the network call fails — the server
    // also clears the cookie, but the client must not stay "logged in".
    mutationFn: async () => {
      try {
        await logout()
      } finally {
        useAuthStore.getState().logout()
      }
    },
  })
}
