import { create } from 'zustand'

interface WalletState {
  balance: number
  topUp(amt: number): void
  charge(amt: number): void
  setBalance(n: number): void
}

const round2 = (x: number): number => +x.toFixed(2)

export const useWalletStore = create<WalletState>((set, get) => ({
  balance: 142.6,
  topUp(amt: number) {
    set({ balance: round2(get().balance + amt) })
  },
  charge(amt: number) {
    set({ balance: round2(get().balance - amt) })
  },
  setBalance(n: number) {
    set({ balance: round2(n) })
  },
}))
