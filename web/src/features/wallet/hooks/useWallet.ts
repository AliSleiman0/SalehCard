import { useQuery } from '@tanstack/react-query'
import { LIVE_REFETCH_MS } from '@/lib/live'
import { fetchWallet } from '../api/wallet'

/** `live`: re-check every 30s while the page is visible (wallet / dashboard pages). */
export function useWallet(enabled = true, live = false) {
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
    refetchInterval: live ? LIVE_REFETCH_MS : false,
  })
}
