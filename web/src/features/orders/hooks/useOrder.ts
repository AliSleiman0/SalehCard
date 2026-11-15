import { useQuery } from '@tanstack/react-query'
import { LIVE_REFETCH_MS } from '@/lib/live'
import { fetchOrder } from '../api/orders'

/** `live`: re-check every 30s while the page is visible (order detail). */
export function useOrder(id: string, live = false) {
  return useQuery({
    queryKey: ['orders', id],
    queryFn: async () => {
      const res = await fetchOrder(id)
      if (!res.success || !res.data) {
        throw new Error(res.error?.message ?? 'Failed to load order')
      }
      return res.data
    },
    enabled: !!id,
    refetchInterval: live ? LIVE_REFETCH_MS : false,
  })
}
