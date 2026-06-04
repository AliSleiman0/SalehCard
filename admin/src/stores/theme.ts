import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { Theme } from '@/types'

interface ThemeState {
  theme: Theme
  setTheme(t: Theme): void
  toggle(): void
  init(): void
}

function applyTheme(theme: Theme): void {
  document.documentElement.setAttribute('data-theme', theme)
  document.documentElement.classList.toggle('dark', theme === 'dark')
}

export const useThemeStore = create<ThemeState>()(
  persist(
    (set, get) => ({
      theme: 'dark' as Theme,
      setTheme(t: Theme) {
        set({ theme: t })
        applyTheme(t)
      },
      toggle() {
        get().setTheme(get().theme === 'dark' ? 'light' : 'dark')
      },
      init() {
        applyTheme(get().theme)
      },
    }),
    { name: 'salehcard-admin-theme' }
  )
)
