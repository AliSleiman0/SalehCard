import { useMutation, useQueryClient } from '@tanstack/react-query'
import { topUp, type TopUpInput } from '../api/wallet'
import type { WalletTransaction } from '@/types'

export function useTopUp() {
  const qc = useQueryClient()
  return useMutation<WalletTransaction, Error, TopUpInput>({
    mutationFn: async (input) => {
      const res = await topUp(input)
      if (!res.success || !res.data) {
        throw new Error(res.error?.message ?? 'Top-up failed')
      }
      return res.data
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['wallet'] })
    },
  })
}
