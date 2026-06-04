import { create } from 'zustand'
import { setAccessToken } from '@/lib/api-client'
import type { AdminUser } from '@/types'

interface AuthState {
  user: AdminUser | null
  isAuthenticated: boolean
  isAdmin: boolean
  login(email: string, password: string): Promise<void>
  logout(): void
}

// NOTE: Token issuance / a real login endpoint is NOT part of this slice's
// backend scope (the backend work here is the AdminOnly middleware + admin
// route map). For local development the AdminOnly middleware bypasses auth when
// no JWT secret is configured, so this store performs a *mock* admin login that
// lets the wired product/inventory pages work end-to-end against the API.
//
// TODO: replace `login()` with a real call to the auth module once it lands
// (POST /api/v1/auth/login → { accessToken, user }); keep the access token in
// memory only (never localStorage), exactly as set up here.
export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  isAuthenticated: false,
  isAdmin: false,
  async login(email: string) {
    // Mock admin session for dev. The api-client will send this token; the
    // dev-mode AdminOnly middleware ignores it (and enforces in production).
    const user: AdminUser = {
      id: 'U-9005',
      name: 'Omar Farouk',
      email: email || 'omar.f@proton.me',
      role: 'admin',
    }
    setAccessToken('dev-admin-token')
    set({ user, isAuthenticated: true, isAdmin: true })
  },
  logout() {
    setAccessToken(null)
    set({ user: null, isAuthenticated: false, isAdmin: false })
  },
}))
