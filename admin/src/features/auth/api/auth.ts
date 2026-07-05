import { apiClient } from '@/lib/api-client'
import type { ApiResponse, UserRole } from '@/types'

export interface AuthUser {
  id: string
  email: string
  role: UserRole
  name?: string
}

export interface AuthResponse {
  accessToken: string
  user: AuthUser
}

/**
 * Admin SMS-2FA challenge. Returned by /auth/login (in place of tokens) when the
 * account is an admin and 2FA is enabled: the client must post the SMS code to
 * /auth/2fa/verify with pendingToken. phoneHint is a masked number for display.
 */
export interface TwoFactorChallenge {
  twoFactorRequired: true
  pendingToken: string
  phoneHint: string
}

/** login() returns either a completed session or a pending 2FA challenge. */
export type LoginResult = AuthResponse | TwoFactorChallenge

/** Narrow a login/refresh payload to the 2FA challenge shape. */
export function isTwoFactorChallenge(d: unknown): d is TwoFactorChallenge {
  return !!d && typeof d === 'object' && (d as { twoFactorRequired?: unknown }).twoFactorRequired === true
}

// Real auth against the shared backend (POST /api/v1/auth/*). The access token
// is returned in the body (kept in memory); a rotating refresh token is set as
// an httpOnly cookie so the session can be restored on reload via refresh().
export function login(email: string, password: string): Promise<ApiResponse<LoginResult>> {
  return apiClient.post<LoginResult>('/api/v1/auth/login', { email, password })
}

/** Complete an admin login by submitting the SMS second factor. */
export function verifyTwoFactor(pendingToken: string, code: string): Promise<ApiResponse<AuthResponse>> {
  return apiClient.post<AuthResponse>('/api/v1/auth/2fa/verify', { pendingToken, code })
}

/** Re-send the SMS code; returns a fresh challenge (new pending token + hint). */
export function resendTwoFactor(pendingToken: string): Promise<ApiResponse<TwoFactorChallenge>> {
  return apiClient.post<TwoFactorChallenge>('/api/v1/auth/2fa/resend', { pendingToken })
}

export function refresh(): Promise<ApiResponse<AuthResponse>> {
  return apiClient.post<AuthResponse>('/api/v1/auth/refresh')
}
