import { useEffect, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useQueryClient } from '@tanstack/react-query'
import { Icon } from '@/components'
import { useAuthStore } from '@/stores/auth'
import { fetchNotifications } from '../api/notifications'
import { useUnreadCount } from '../hooks/useUnreadCount'
import { newPopups } from '../lib/popups'
import type { AppNotification } from '../types'

const SHOW_MS = 6000

/**
 * Pops a banner at the top of the screen when a wallet top-up is approved or
 * rejected (client feedback #64), and refreshes the balance. Driven by the
 * bell's unread-count poll: when the count rises it fetches the inbox once and
 * shows any new top-up notes. Mounted once in the app Layout.
 */
export function NotificationPopups() {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)
  const unread = useUnreadCount().data
  const qc = useQueryClient()
  const navigate = useNavigate()
  const since = useRef(Date.now())
  const prevUnread = useRef<number | undefined>(undefined)
  const [shown, setShown] = useState<AppNotification[]>([])

  useEffect(() => {
    if (!isAuthenticated || unread === undefined) return
    const rose = prevUnread.current !== undefined && unread > prevUnread.current
    prevUnread.current = unread
    if (!rose) return
    let cancelled = false
    void fetchNotifications().then((res) => {
      if (cancelled || !res.success || !res.data) return
      void qc.invalidateQueries({ queryKey: ['notifications'] })
      const fresh = newPopups(res.data, since.current)
      if (fresh.length === 0) return
      since.current = Math.max(...fresh.map((x) => Date.parse(x.createdAt)))
      void qc.invalidateQueries({ queryKey: ['wallet'] })
      void qc.invalidateQueries({ queryKey: ['topup-requests'] })
      setShown((s) => [...s, ...fresh])
      for (const f of fresh) {
        setTimeout(() => setShown((s) => s.filter((x) => x.id !== f.id)), SHOW_MS)
      }
    })
    return () => {
      cancelled = true
    }
  }, [unread, isAuthenticated, qc])

  if (shown.length === 0) return null
  return (
    <div className="notif-pop-wrap" role="status" aria-live="polite">
      {shown.map((n) => {
        const bad = n.kind === 'topup_rejected'
        return (
          <div
            key={n.id}
            className={'notif-pop' + (bad ? ' bad' : '')}
            onClick={() => {
              setShown((s) => s.filter((x) => x.id !== n.id))
              navigate('/wallet')
            }}
          >
            <span className="notif-pop-ic">
              <Icon name={bad ? 'close' : 'wallet'} size={18} />
            </span>
            <div className="col" style={{ gap: 2 }}>
              <strong>{n.title}</strong>
              {n.body && <span className="small">{n.body}</span>}
            </div>
          </div>
        )
      })}
    </div>
  )
}
