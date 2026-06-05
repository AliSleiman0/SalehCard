import { useQuery } from '@tanstack/react-query'
import { fetchOrder } from '../api/orders'

export function useOrder(id: string) {
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
  })
}
