import { useQuery } from '@tanstack/react-query'
import { fetchMyReview } from '../api/reviews'
import { unwrap } from '@/lib/api-error'
import { useAuthStore } from '@/stores/auth'

// useMyReview reports whether the signed-in customer has already reviewed a
// product (drives the "Write a review" ↔ "You reviewed this" CTA).
export function useMyReview(productId: string) {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)
  return useQuery({
    queryKey: ['my-review', productId],
    queryFn: async () => unwrap(await fetchMyReview(productId), 'Failed to load review'),
    enabled: isAuthenticated && !!productId,
  })
}
