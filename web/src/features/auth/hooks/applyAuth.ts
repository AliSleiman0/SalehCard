import { setAccessToken } from '@/lib/api-client'
import { markSessionHint } from '@/lib/sessionHint'
import { useAuthStore } from '@/stores/auth'
import type { AuthResponse } from '@/types'

// applyAuth stores the access token in memory and the user in the auth store.
// Shared by the login/register mutations and on-load session restoration.
export function applyAuth(data: AuthResponse): void {
  setAccessToken(data.accessToken)
  useAuthStore.getState().setUser(data.user)
  // Record that a session now exists so the next cold load attempts a refresh.
  markSessionHint()
}
