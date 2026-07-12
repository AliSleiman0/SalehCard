import { useQuery } from '@tanstack/react-query'
import { fetchOffers } from '../api/offers'
import { unwrap } from '@/lib/api-error'
import { useAuthStore } from '@/stores/auth'

// useOffers loads live sale deals (GET /offers is AuthRequired).
export function useOffers() {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)
  return useQuery({
    queryKey: ['offers'],
    queryFn: async () => unwrap(await fetchOffers(), 'Failed to load offers'),
    enabled: isAuthenticated,
  })
}
