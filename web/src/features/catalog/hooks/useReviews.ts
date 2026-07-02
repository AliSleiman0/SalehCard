import { useQuery } from '@tanstack/react-query'
import { fetchReviews } from '../api/reviews'

export function useReviews(productId: string) {
  return useQuery({
    queryKey: ['reviews', productId],
    queryFn: async () => {
      const res = await fetchReviews(productId)
      if (!res.success || !res.data) {
        throw new Error(res.error?.message ?? 'Failed to load reviews')
      }
      return res.data
    },
    enabled: !!productId,
  })
}
