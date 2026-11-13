import { useEffect, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useQueryClient } from '@tanstack/react-query'
import { Icon } from '@/components'
import { useAuthStore } from '@/stores/auth'
import { fetchNotifications } from '../api/notifications'
import { useUnreadCount } from '../hooks/useUnreadCount'
import { baselineFrom, newPopups } from '../lib/popups'
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
  // Baseline = newest notification time already in the inbox (server clock),
  // seeded on the first observation; null until then. A device clock running
  // ahead of the server would otherwise hide notes created right after load.
  const since = useRef<number | null>(null)
  const prevUnread = useRef<number | undefined>(undefined)
  const [shown, setShown] = useState<AppNotification[]>([])

  useEffect(() => {
    if (!isAuthenticated || unread === undefined) return
    const first = prevUnread.current === undefined
    const rose = !first && unread > prevUnread.current!
    prevUnread.current = unread
    if (!first && !rose) return
    let cancelled = false
    void fetchNotifications().then((res) => {
      if (!res.success || !res.data) return
      if (since.current === null) {
        // First look at the inbox: everything in it is "old" — no pop-ups.
        // Seeded even if this effect was superseded, so it always gets set.
        since.current = baselineFrom(res.data)
        return
      }
      if (cancelled) return
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
