import { create } from 'zustand'

interface UiState {
  agent: boolean
  setAgent(v: boolean): void
}

export const useUiStore = create<UiState>((set) => ({
  agent: false,
  setAgent(v: boolean) {
    set({ agent: v })
  },
}))
