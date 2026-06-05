import { create } from 'zustand'

interface UiState {
  agent: boolean
  setAgent(v: boolean): void
  acctDrawer: boolean
  setAcctDrawer(v: boolean): void
  toggleAcctDrawer(): void
}

export const useUiStore = create<UiState>((set) => ({
  agent: false,
  setAgent(v: boolean) {
    set({ agent: v })
  },
  acctDrawer: false,
  setAcctDrawer(v: boolean) {
    set({ acctDrawer: v })
  },
  toggleAcctDrawer() {
    set((s) => ({ acctDrawer: !s.acctDrawer }))
  },
}))
