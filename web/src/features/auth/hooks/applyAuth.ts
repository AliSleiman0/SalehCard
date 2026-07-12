import { setAccessToken } from '@/lib/api-client'
import { ApiError } from '@/lib/api-error'
import { markSessionHint } from '@/lib/sessionHint'
import { useAuthStore } from '@/stores/auth'
import type { AuthResponse } from '@/types'

// assertCustomerSession rejects a half-session before it is stored. Admins with
// SMS 2FA enabled get `twoFactorRequired` (and no access token) from the phone
// endpoints; the storefront doesn't implement the challenge, so we fail loudly
// instead of persisting an undefined token.
export function assertCustomerSession(data: AuthResponse): void {
  if (data.twoFactorRequired || !data.accessToken) {
    throw new ApiError(
      'This account must sign in through the admin console.',
      'TWO_FACTOR_REQUIRED',
    )
  }
}

// applyAuth stores the access token in memory and the user in the auth store.
// Shared by the login/register mutations and on-load session restoration.
export function applyAuth(data: AuthResponse): void {
  setAccessToken(data.accessToken)
  useAuthStore.getState().setUser(data.user)
  // Record that a session now exists so the next cold load attempts a refresh.
  markSessionHint()
}
