import { create } from 'zustand'
import { persist } from 'zustand/middleware'

type Theme = 'light' | 'dark'

interface ThemeState {
  theme: Theme
  setTheme(t: Theme): void
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
      init() {
        const stored = get().theme
        if (stored) {
          applyTheme(stored)
        } else {
          const prefersDark =
            typeof window !== 'undefined' &&
            window.matchMedia('(prefers-color-scheme: dark)').matches
          const resolved: Theme = prefersDark ? 'dark' : 'light'
          set({ theme: resolved })
          applyTheme(resolved)
        }
      },
    }),
    {
      name: 'salehcard-theme',
    },
  ),
)
