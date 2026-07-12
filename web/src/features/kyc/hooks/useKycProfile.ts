import { useQuery } from '@tanstack/react-query'
import { fetchKycProfile } from '../api/kyc'
import { unwrap } from '@/lib/api-error'
import { useAuthStore } from '@/stores/auth'

// useKycProfile reads the signed-in customer's verification status.
export function useKycProfile() {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)
  return useQuery({
    queryKey: ['kyc'],
    queryFn: async () => unwrap(await fetchKycProfile(), 'Failed to load verification status'),
    enabled: isAuthenticated,
  })
}
