import { useQuery } from '@tanstack/react-query'
import { fetchPaymentConfig } from '../api/payments'
import { unwrap } from '@/lib/api-error'
import { useAuthStore } from '@/stores/auth'

// usePaymentConfig reports whether USDT is enabled and which networks are offered.
export function usePaymentConfig() {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)
  return useQuery({
    queryKey: ['payment-config'],
    queryFn: async () => unwrap(await fetchPaymentConfig(), 'Failed to load payment config'),
    staleTime: 5 * 60_000,
    enabled: isAuthenticated,
  })
}
