import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { Currency } from '@/lib/utils'

interface CurrencyState {
  currency: Currency
  setCurrency(c: Currency): void
}

export const useCurrencyStore = create<CurrencyState>()(
  persist(
    (set) => ({
      currency: 'USD',
      setCurrency(c: Currency) {
        set({ currency: c })
      },
    }),
    {
      name: 'salehcard-currency',
    },
  ),
)
