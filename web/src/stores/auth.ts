import { create } from 'zustand'
import type { User } from '@/types'
import { setAccessToken } from '@/lib/api-client'

interface AuthState {
  user: User | null
  isAuthenticated: boolean
  // hydrated flips true once the on-load session-restore attempt completes
  // (success or failure). Route guards wait for it to avoid flashing /login.
  hydrated: boolean
  setUser(u: User | null): void
  setHydrated(v: boolean): void
  logout(): void
}

// Not persisted — the httpOnly refresh cookie is the source of truth, and the
// session is rehydrated on load via /api/v1/auth/refresh (see app/providers).
export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  isAuthenticated: false,
  hydrated: false,
  setUser(user: User | null) {
    set({ user, isAuthenticated: !!user })
  },
  setHydrated(v: boolean) {
    set({ hydrated: v })
  },
  logout() {
    set({ user: null, isAuthenticated: false })
    setAccessToken(null)
  },
}))
