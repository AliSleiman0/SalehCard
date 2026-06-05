import { useQuery } from '@tanstack/react-query'
import { fetchWallet } from '../api/wallet'

export function useWallet(enabled = true) {
  return useQuery({
    queryKey: ['wallet'],
    queryFn: async () => {
      const res = await fetchWallet()
      if (!res.success || !res.data) {
        throw new Error(res.error?.message ?? 'Failed to load wallet')
      }
      return res.data
    },
    enabled,
  })
}
