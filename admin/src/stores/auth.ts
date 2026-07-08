import { create } from 'zustand'
import { setAccessToken, ApiError } from '@/lib/api-client'
import {
  login as apiLogin,
  refresh as apiRefresh,
  verifyTwoFactor as apiVerifyTwoFactor,
  resendTwoFactor as apiResendTwoFactor,
  isTwoFactorChallenge,
  type AuthUser,
  type AuthResponse,
} from '@/features/auth/api/auth'
import type { AdminUser } from '@/types'

interface AuthState {
  user: AdminUser | null
  isAuthenticated: boolean
  isAdmin: boolean
  // hydrated flips true once the on-load session-restore attempt completes
  // (success or failure). RequireAdmin waits for it so a reload doesn't bounce
  // an authenticated admin to /login before the refresh resolves.
  hydrated: boolean
  // twoFactor holds an in-progress admin SMS-2FA challenge (set after a correct
  // password when 2FA is required). While non-null the login page shows the code
  // step; it is cleared on success or cancel.
  twoFactor: { pendingToken: string; phoneHint: string } | null
  // login returns 'ok' when the session is established, or '2fa' when a second
  // factor is required (twoFactor is then set and the caller shows the code step).
  login(email: string, password: string): Promise<'ok' | '2fa'>
  verifyTwoFactor(code: string): Promise<void>
  resendTwoFactor(): Promise<string>
  cancelTwoFactor(): void
  restore(): Promise<void>
  logout(): void
}

// Map the backend auth user onto the admin identity shape (the API user has no
// `name` field — fall back to the email).
function toAdminUser(u: AuthUser): AdminUser {
  return {
    id: u.id,
    name: u.name ?? u.email,
    email: u.email,
    role: u.role,
    permissions: u.permissions ?? [],
  }
}

// establishSession accepts a completed auth response, rejecting non-admins, and
// flips the store to the authenticated state. Shared by password login and 2FA.
function establishSession(res: AuthResponse | undefined, set: (s: Partial<AuthState>) => void) {
  const u = res?.user
  if (!u || !res?.accessToken) {
    throw new ApiError(500, 'invalid_response', 'Login failed — unexpected response.')
  }
  if (u.role !== 'admin') {
    setAccessToken(null)
    set({ user: null, isAuthenticated: false, isAdmin: false, twoFactor: null })
    throw new ApiError(403, 'forbidden', 'This account is not an administrator.')
  }
  setAccessToken(res.accessToken)
  set({ user: toAdminUser(u), isAuthenticated: true, isAdmin: true, twoFactor: null })
}

export const useAuthStore = create<AuthState>((set, get) => ({
  user: null,
  isAuthenticated: false,
  isAdmin: false,
  hydrated: false,
  twoFactor: null,

  // Real login: POST /api/v1/auth/login. Admins with SMS-2FA enabled get a
  // challenge (no session yet) — the caller then drives verifyTwoFactor. Non-admins
  // are rejected here (the AdminOnly middleware would 401 them anyway).
  async login(email: string, password: string) {
    const res = await apiLogin(email, password)
    const d = res.data
    if (isTwoFactorChallenge(d)) {
      set({ twoFactor: { pendingToken: d.pendingToken, phoneHint: d.phoneHint } })
      return '2fa'
    }
    establishSession(d, set)
    return 'ok'
  },

  // Complete an admin login by submitting the SMS second factor.
  async verifyTwoFactor(code: string) {
    const tf = get().twoFactor
    if (!tf) {
      throw new ApiError(400, 'no_challenge', 'No verification in progress — sign in again.')
    }
    const res = await apiVerifyTwoFactor(tf.pendingToken, code)
    establishSession(res.data, set)
  },

  // Re-send the SMS code; returns the (possibly re-masked) phone hint for display.
  async resendTwoFactor() {
    const tf = get().twoFactor
    if (!tf) {
      throw new ApiError(400, 'no_challenge', 'No verification in progress — sign in again.')
    }
    const res = await apiResendTwoFactor(tf.pendingToken)
    const d = res.data
    if (!d?.pendingToken) {
      throw new ApiError(500, 'invalid_response', 'Could not resend the code.')
    }
    set({ twoFactor: { pendingToken: d.pendingToken, phoneHint: d.phoneHint } })
    return d.phoneHint
  },

  cancelTwoFactor() {
    set({ twoFactor: null })
  },

  // On-load session restore from the httpOnly refresh cookie. Marks hydrated
  // either way so the route guard can resolve.
  async restore() {
    try {
      const res = await apiRefresh()
      const u = res.data?.user
      if (u && res.data?.accessToken && u.role === 'admin') {
        setAccessToken(res.data.accessToken)
        set({ user: toAdminUser(u), isAuthenticated: true, isAdmin: true })
      }
    } catch {
      /* no/expired session — remain logged out */
    } finally {
      set({ hydrated: true })
    }
  },

  logout() {
    setAccessToken(null)
    set({ user: null, isAuthenticated: false, isAdmin: false, twoFactor: null })
  },
}))

/** Whether a permission set grants perm — the "*" wildcard (super admin) passes
 *  everything. The single source of truth for the RBAC check. */
export function hasPerm(permissions: string[] | undefined, perm: string): boolean {
  return !!permissions && (permissions.includes('*') || permissions.includes(perm))
}

/**
 * Reactive permission check for components: subscribes to the logged-in
 * admin's permission set so gated UI re-renders when the session changes.
 * `can('orders.manage')`; `can('*')` is true only for super admins.
 */
export function useCan(): (perm: string) => boolean {
  const permissions = useAuthStore((s) => s.user?.permissions)
  return (perm: string) => hasPerm(permissions, perm)
}

/** Reactive super-admin check (wildcard permission). */
export function useIsSuperAdmin(): boolean {
  return useCan()('*')
}
