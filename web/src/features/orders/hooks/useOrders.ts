import { useQuery } from '@tanstack/react-query'
import { fetchOrders } from '../api/orders'

export function useOrders() {
  return useQuery({
    queryKey: ['orders'],
    queryFn: async () => {
      const res = await fetchOrders()
      if (!res.success || !res.data) {
        throw new Error(res.error?.message ?? 'Failed to load orders')
      }
      return res.data
    },
  })
}
