import { useEffect, useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { getDashboardStats } from '@/features/dashboard/api/dashboard'
import { useCan } from '@/stores/auth'
import { toast } from '@/stores/toast'
import { isNewTopUp, playChime, titleWithCount } from '../lib/topupAlert'

/**
 * Alerts the admin when a customer files a new wallet top-up request: toast,
 * chime, a browser notification (when permitted) and a "(N)" tab-title prefix.
 * Shares the dashboard-stats cache (same query key as the sidebar badge) but
 * keeps polling while the tab is in the background, so an admin working in
 * another tab still hears about it. Mounted once in AdminLayout.
 */
export function useNewTopUpAlert(): void {
  const can = useCan()
  const navigate = useNavigate()
  const enabled = can('dashboard.view') && can('topups.view')
  const { data } = useQuery({
    queryKey: ['admin', 'dashboard', 'stats'],
    queryFn: () => getDashboardStats(),
    enabled,
    refetchInterval: 15_000,
    refetchIntervalInBackground: true,
  })
  const pending = data?.data?.pendingTopups
  const prev = useRef<number | undefined>(undefined)

  // Ask once for notification permission on the first click anywhere (browsers
  // only allow the prompt from a user gesture).
  useEffect(() => {
    if (!enabled || typeof Notification === 'undefined' || Notification.permission !== 'default') return
    const ask = () => void Notification.requestPermission()
    window.addEventListener('click', ask, { once: true })
    return () => window.removeEventListener('click', ask)
  }, [enabled])

  useEffect(() => {
    if (pending === undefined) return
    document.title = titleWithCount(document.title, pending)
    if (isNewTopUp(prev.current, pending)) {
      const msg = 'New top-up request waiting for approval'
      toast.info(msg)
      playChime()
      if (typeof Notification !== 'undefined' && Notification.permission === 'granted') {
        const n = new Notification('FlashCash — new top-up request', {
          body: `${pending} pending top-up request${pending > 1 ? 's' : ''}`,
          tag: 'topup-pending',
        })
        n.onclick = () => {
          window.focus()
          navigate('/topups')
          n.close()
        }
      }
    }
    prev.current = pending
  }, [pending, navigate])
}
