import { useQuery } from '@tanstack/react-query'
import { fetchNotifications } from '../api/notifications'
import { unwrap } from '@/lib/api-error'
import { useAuthStore } from '@/stores/auth'

export function useNotifications() {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)
  return useQuery({
    queryKey: ['notifications'],
    queryFn: async () => unwrap(await fetchNotifications(), 'Failed to load notifications'),
    enabled: isAuthenticated,
  })
}
