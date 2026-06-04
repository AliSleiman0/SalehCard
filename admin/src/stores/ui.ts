import { create } from 'zustand'
import { persist } from 'zustand/middleware'

interface UiState {
  collapsed: boolean
  setCollapsed(v: boolean): void
  toggleCollapsed(): void
}

// Sidebar collapsed state persists across sessions (matches the prototype).
export const useUiStore = create<UiState>()(
  persist(
    (set, get) => ({
      collapsed: false,
      setCollapsed(v: boolean) {
        set({ collapsed: v })
      },
      toggleCollapsed() {
        set({ collapsed: !get().collapsed })
      },
    }),
    { name: 'salehcard-admin-ui' }
  )
)
