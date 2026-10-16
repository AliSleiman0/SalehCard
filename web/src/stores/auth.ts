import { create } from 'zustand'
import type { User } from '@/types'
import { setAccessToken } from '@/lib/api-client'
import { clearSessionHint } from '@/lib/sessionHint'
import { queryClient } from '@/lib/query-client'

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
export const useAuthStore = create<AuthState>((set, get) => ({
  user: null,
  isAuthenticated: false,
  hydrated: false,
  setUser(user: User | null) {
    const changed = get().user?.id !== user?.id
    set({ user, isAuthenticated: !!user })
    // Catalog prices (retail vs reseller) and the wallet are cached per query
    // key, not per user — drop them when the signed-in identity changes so the
    // next person on this browser never sees the previous user's data.
    if (changed) resetUserScopedCache()
  },
  setHydrated(v: boolean) {
    set({ hydrated: v })
  },
  logout() {
    const hadUser = !!get().user
    set({ user: null, isAuthenticated: false })
    setAccessToken(null)
    // Drop the session hint so the next cold load stays silent (no refresh probe).
    clearSessionHint()
    if (hadUser) resetUserScopedCache()
  },
}))

function resetUserScopedCache() {
  void queryClient.resetQueries()
}
