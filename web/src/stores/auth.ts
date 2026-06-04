import { create } from 'zustand'
import type { User } from '@/types'
import { setAccessToken } from '@/lib/api-client'

interface AuthState {
  user: User | null
  isAuthenticated: boolean
  setUser(u: User | null): void
  logout(): void
}

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  isAuthenticated: false,
  setUser(user: User | null) {
    set({ user, isAuthenticated: !!user })
  },
  logout() {
    set({ user: null, isAuthenticated: false })
    setAccessToken(null)
  },
}))
