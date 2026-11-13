import { useQuery } from '@tanstack/react-query'
import { fetchUnreadCount } from '../api/notifications'
import { useAuthStore } from '@/stores/auth'

// useUnreadCount feeds the bell badge. It NEVER throws — any failure resolves to
// 0 so a hiccup can't render an error where a count should be. Polls every 20s
// and on window focus.
export function useUnreadCount() {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)
  return useQuery({
    queryKey: ['notifications', 'unread'],
    queryFn: async () => {
      try {
        const res = await fetchUnreadCount()
        return res.success && res.data ? res.data.count : 0
      } catch {
        return 0
      }
    },
    enabled: isAuthenticated,
    refetchInterval: 20_000, // also drives the top-up pop-up (NotificationPopups)
    retry: false,
  })
}
