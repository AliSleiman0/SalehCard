import { useQuery } from '@tanstack/react-query'
import { LIVE_REFETCH_MS } from '@/lib/live'
import { fetchOrders } from '../api/orders'

/** `live`: re-check every 30s while the page is visible (order history / dashboard). */
export function useOrders(live = false) {
  return useQuery({
    queryKey: ['orders'],
    queryFn: async () => {
      const res = await fetchOrders()
      if (!res.success || !res.data) {
        throw new Error(res.error?.message ?? 'Failed to load orders')
      }
      return res.data
    },
    refetchInterval: live ? LIVE_REFETCH_MS : false,
  })
}
