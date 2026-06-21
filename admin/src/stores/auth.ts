import { create } from 'zustand'
import { setAccessToken, ApiError } from '@/lib/api-client'
import { login as apiLogin, refresh as apiRefresh, type AuthUser } from '@/features/auth/api/auth'
import type { AdminUser } from '@/types'

interface AuthState {
  user: AdminUser | null
  isAuthenticated: boolean
  isAdmin: boolean
  // hydrated flips true once the on-load session-restore attempt completes
  // (success or failure). RequireAdmin waits for it so a reload doesn't bounce
  // an authenticated admin to /login before the refresh resolves.
  hydrated: boolean
  login(email: string, password: string): Promise<void>
  restore(): Promise<void>
  logout(): void
}

// Map the backend auth user onto the admin identity shape (the API user has no
// `name` field — fall back to the email).
function toAdminUser(u: AuthUser): AdminUser {
  return { id: u.id, name: u.name ?? u.email, email: u.email, role: 'admin' }
}

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  isAuthenticated: false,
  isAdmin: false,
  hydrated: false,

  // Real login: POST /api/v1/auth/login. Non-admins are rejected here (the
  // AdminOnly middleware would 401 them on every admin call anyway).
  async login(email: string, password: string) {
    const res = await apiLogin(email, password)
    const u = res.data?.user
    if (!u || !res.data?.accessToken) {
      throw new ApiError(500, 'invalid_response', 'Login failed — unexpected response.')
    }
    if (u.role !== 'admin') {
      setAccessToken(null)
      set({ user: null, isAuthenticated: false, isAdmin: false })
      throw new ApiError(403, 'forbidden', 'This account is not an administrator.')
    }
    setAccessToken(res.data.accessToken)
    set({ user: toAdminUser(u), isAuthenticated: true, isAdmin: true })
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
    set({ user: null, isAuthenticated: false, isAdmin: false })
  },
}))
